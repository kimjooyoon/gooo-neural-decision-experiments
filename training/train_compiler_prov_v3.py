#!/usr/bin/env python3
"""Bounded offline MPS fine tuning. All deployed inference/orchestration is Go."""
import argparse
import copy
import json
import math
import random
import resource
import time
from pathlib import Path

import numpy as np
import torch
from train_pilot_v2 import (Model, LABELS, features, scores, calibrate,
                            export_model, exported_logits, probabilities,
                            save_json, sha, require, unique_pairs)


def parent_state(directory):
    raw = (directory / "model.json").read_bytes()
    metadata = json.loads(raw, object_pairs_hook=unique_pairs)
    blob = (directory / "weights.bin").read_bytes()
    require(metadata["variant"] == "fp32" and metadata["labels"] == LABELS,
            "parent must be the compatible FP32 model")
    require(sha(blob) == metadata["weights_sha256"], "parent weight hash mismatch")
    result = {}
    names = {"w1": "first.weight", "b1": "first.bias", "w2": "last.weight", "b2": "last.bias"}
    for tensor in metadata["tensors"]:
        require(tensor["encoding"] == "float32_le", "parent tensor encoding")
        values = np.frombuffer(blob[tensor["offset"]:tensor["offset"] + tensor["bytes"]], dtype="<f4").copy()
        shape = (tensor["rows"], tensor["cols"]) if tensor["name"].startswith("w") else (tensor["count"],)
        require(np.isfinite(values).all(), "non-finite parent tensor")
        result[names[tensor["name"]]] = torch.from_numpy(values.reshape(shape))
    return result, {"metadata_sha256": sha(raw), "weights_sha256": sha(blob)}


def fit(state, xs, ys, rows, epochs, seed, qat):
    random.seed(seed)
    np.random.seed(seed)
    torch.manual_seed(seed)
    model = Model(qat=qat)
    model.load_state_dict(state)
    model.to("mps")
    optimizer = torch.optim.AdamW(model.parameters(), lr=0.0015, weight_decay=0.01)
    x = torch.from_numpy(xs["train"]).to("mps")
    y = torch.from_numpy(ys["train"]).to("mps")
    cx = torch.from_numpy(xs["calibration"]).to("mps")
    # Repeated optimization exposure is explicit, not extra unique examples.
    repair_indices = [i for i, row in enumerate(rows["train"]) if row["template_id"].startswith("compiler-repair-")]
    indices = np.concatenate([np.arange(len(y)), np.repeat(repair_indices, 127)])
    rng = np.random.default_rng(seed)
    best, best_loss, history = None, float("inf"), []
    allocated_peak, driver_peak = 0, 0
    started = time.perf_counter()
    for epoch in range(epochs):
        order = rng.permutation(indices)
        model.train()
        loss_total = 0.0
        for begin in range(0, len(order), 256):
            batch = torch.from_numpy(order[begin:begin + 256]).to("mps")
            optimizer.zero_grad(set_to_none=True)
            loss = torch.nn.functional.cross_entropy(model(x[batch]), y[batch])
            loss.backward()
            optimizer.step()
            loss_total += float(loss.detach().cpu())
        model.eval()
        with torch.no_grad():
            logits = model(cx).cpu().numpy()
        calibration_loss = scores(logits, ys["calibration"], 1.0)["nll"]
        if calibration_loss < best_loss:
            best_loss = calibration_loss
            best = {key: value.detach().cpu().clone() for key, value in model.state_dict().items()}
        allocated_peak = max(allocated_peak, torch.mps.current_allocated_memory())
        driver_peak = max(driver_peak, torch.mps.driver_allocated_memory())
        history.append({"epoch": epoch + 1, "training_loss": loss_total / math.ceil(len(indices) / 256),
                        "calibration_nll": calibration_loss})
    torch.mps.synchronize()
    duration = time.perf_counter() - started
    model.cpu()
    model.load_state_dict(best)
    return model, {"epochs": epochs, "steps": epochs * math.ceil(len(indices) / 256),
                   "unique_training_rows": len(y), "optimization_rows_per_epoch": len(indices),
                   "repair_exposures_per_epoch_per_row": 128, "loop_wall_seconds": duration,
                   "mps_allocated_peak_sampled_bytes": allocated_peak, "mps_driver_peak_sampled_bytes": driver_peak,
                   "process_lifetime_peak_rss_bytes": resource.getrusage(resource.RUSAGE_SELF).ru_maxrss,
                   "selection": "minimum calibration NLL; test split excluded", "history": history}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--dataset", type=Path, required=True)
    parser.add_argument("--parent", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--epochs", type=int, default=60)
    parser.add_argument("--seed", type=int, default=20361001)
    args = parser.parse_args()
    require(1 <= args.epochs <= 120, "bounded epoch budget")
    require(torch.backends.mps.is_available(), "MPS GPU required")
    require(not args.output.exists(), "output must be fresh")
    raw = args.dataset.read_bytes()
    manifest_raw = args.dataset.with_name("manifest.json").read_bytes()
    manifest = json.loads(manifest_raw, object_pairs_hook=unique_pairs)
    require(sha(raw) == manifest["dataset_sha256"], "dataset SHA mismatch")
    records = [json.loads(line, object_pairs_hook=unique_pairs) for line in raw.splitlines()]
    require(len(records) == 6146 and len({row["id"] for row in records}) == len(records), "dataset row count/IDs")
    require(len({row["text"] for row in records}) == len(records), "duplicate prompts")
    fields = {"id", "template_id", "configuration_id", "language", "split", "text", "label"}
    require(all(set(row) == fields and row["label"] in LABELS and row["split"] in ("train", "calibration", "test")
                and row["language"] in ("en", "ko") and 1 <= len(row["text"].encode()) <= 512 for row in records), "invalid row")
    rows = {split: [row for row in records if row["split"] == split] for split in ("train", "calibration", "test")}
    require({key: len(value) for key, value in rows.items()} == manifest["rows"], "split count mismatch")
    for a, b in (("train", "calibration"), ("train", "test"), ("calibration", "test")):
        require(not {row["template_id"] for row in rows[a]} & {row["template_id"] for row in rows[b]}, "template leakage")
    require(all(row["split"] == "train" for row in records if row["template_id"].startswith("compiler-repair-")), "repair leakage")
    xs = {key: np.stack([features(row["text"]) for row in value]) for key, value in rows.items()}
    ys = {key: np.array([LABELS.index(row["label"]) for row in value], dtype=np.int64) for key, value in rows.items()}
    initial, parent_pins = parent_state(args.parent)
    args.output.mkdir(parents=True)
    root = Path(__file__).resolve().parents[1]
    preexecution = {"schema": "gooo/compiler-prov-finetune-preexecution/v1", "dataset_sha256": sha(raw),
                   "dataset_manifest_sha256": sha(manifest_raw), "parent": parent_pins,
                   "source_sha256": {"training/train_compiler_prov_v3.py": sha(Path(__file__).read_bytes()),
                                     "training/train_pilot_v2.py": sha((root / "training/train_pilot_v2.py").read_bytes()),
                                     "cmd/compiler-curriculum/main.go": sha((root / "cmd/compiler-curriculum/main.go").read_bytes())},
                   "epochs_per_variant": args.epochs, "seed": args.seed, "device": "mps", "torch": torch.__version__,
                   "numpy": np.__version__, "split_counts": {key: len(value) for key, value in rows.items()},
                   "test_used_for_training_or_calibration": False, "parent_training": "original train split only",
                   "scope": "Existing eight-operation MLP fine tune; Gooo and PROV-O prompt views, no OWL reasoner"}
    save_json(args.output / "preexecution.json", preexecution)
    torch.set_num_threads(2)
    models, training = {}, {}
    models["fp32"], training["fp32"] = fit(initial, xs, ys, rows, args.epochs, args.seed, False)
    models["qat_ternary"], training["qat_ternary"] = fit(copy.deepcopy(models["fp32"].state_dict()), xs, ys, rows,
                                                         args.epochs, args.seed + 1, True)
    models["ptq_ternary"] = models["fp32"]
    results, parity = {}, []
    for variant in ("fp32", "ptq_ternary", "qat_ternary"):
        directory = args.output / "models" / variant
        export_model(models[variant], variant, directory, 1.0, 0.0)
        temperature, threshold = calibrate(exported_logits(directory, xs["calibration"]), ys["calibration"])
        metadata = json.loads((directory / "model.json").read_text())
        metadata.update(temperature=temperature, confidence_threshold=threshold)
        save_json(directory / "model.json", metadata)
        results[variant] = {"scores": {key: scores(exported_logits(directory, value), ys[key], temperature) for key, value in xs.items()},
                            "old_parent_test": scores(exported_logits(args.parent.parent / variant, xs["test"]), ys["test"],
                                                       json.loads((args.parent.parent / variant / "model.json").read_text())["temperature"]),
                            "temperature": temperature, "confidence_threshold": threshold,
                            "weights_bytes": (directory / "weights.bin").stat().st_size}
        test_logits = exported_logits(directory, xs["test"])
        test_probabilities = probabilities(test_logits, temperature)
        # Interleave views and classes rather than report only one sorted class.
        for index in np.linspace(0, len(rows["test"]) - 1, 32, dtype=int):
            parity.append({"variant": variant, "text": rows["test"][index]["text"], "features": xs["test"][index].tolist(),
                           "logits": test_logits[index].tolist(), "probabilities": test_probabilities[index].tolist(),
                           "selected_label": LABELS[int(test_probabilities[index].argmax())]})
    report = {"schema": "gooo/compiler-prov-finetune-report/v1", "status": "TRAINED_AND_EXPORTED",
              "preexecution_sha256": sha((args.output / "preexecution.json").read_bytes()), "training": training,
              "variants": results, "limitations": ["768 test rows are three views of 256 held-out original instructions",
                                                     "Two known compiler failures are training repair cases, not unseen evaluation",
                                                     "Parent checkpoint selection used original calibration; test was excluded",
                                                     "PROV-O context is vocabulary conditioning, not formal provenance reasoning",
                                                     "Packed ternary matrices are 1.6 physical bits per matrix weight; runtime is decoded"]}
    save_json(args.output / "report.json", report)
    save_json(args.output / "go-parity.json", {"schema": "gooo/tiny-ir-decision-parity/v1", "rows": parity})
    print(json.dumps({"status": report["status"], "test_correct_of_768": {key: value["scores"]["test"]["correct"] for key, value in results.items()},
                      "old_test_correct_of_768": {key: value["old_parent_test"]["correct"] for key, value in results.items()}}))


if __name__ == "__main__":
    main()
