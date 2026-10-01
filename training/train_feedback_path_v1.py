#!/usr/bin/env python3
"""Offline MPS tuning for finite Gooo path sets; deployed execution stays Go."""
import argparse
import copy
import json
import math
import resource
import subprocess
import time
from pathlib import Path

import numpy as np
import torch
import train_pilot_v2 as core
from train_typed_path_v1 import PATH_LABELS, positioned_features

core.LABELS = PATH_LABELS


def parent_state(directory):
    raw = (directory / "model.json").read_bytes()
    meta = json.loads(raw, object_pairs_hook=core.unique_pairs)
    blob = (directory / meta["weights_file"]).read_bytes()
    core.require(meta["schema"] == "gooo/tiny-path-decision-model/v1" and meta["variant"] == "fp32"
                 and meta["feature_version"] == "positioned_intent_ngrams_v1"
                 and meta["labels"] == PATH_LABELS and core.sha(blob) == meta["weights_sha256"], "pinned structural FP32 parent required")
    state, names = {}, {"w1": "first.weight", "b1": "first.bias", "w2": "last.weight", "b2": "last.bias"}
    for tensor in meta["tensors"]:
        core.require(tensor["encoding"] == "float32_le" and tensor["name"] in names, "parent encoding")
        values = np.frombuffer(blob[tensor["offset"]:tensor["offset"] + tensor["bytes"]], dtype="<f4").copy()
        shape = (tensor["rows"], tensor["cols"]) if tensor["name"].startswith("w") else (tensor["count"],)
        core.require(np.isfinite(values).all(), "finite parent tensors required")
        state[names[tensor["name"]]] = torch.from_numpy(values.reshape(shape))
    core.require(set(state) == set(names.values()), "complete parent state required")
    return state, {"metadata_sha256": core.sha(raw), "weights_sha256": core.sha(blob)}


def arrays(rows):
    x = np.stack([positioned_features(r["text"]) for r in rows])
    pairs = np.asarray([[PATH_LABELS.index(v) for v in r["eligible_labels"]] for r in rows], dtype=np.int64)
    y = np.asarray([[float(v in r["best_finite_labels"]) / len(r["best_finite_labels"])
                     for v in r["eligible_labels"]] for r in rows], dtype=np.float32)
    return x, pairs, y


def pair_probabilities(logits, pairs, temperature):
    eligible = logits[np.arange(len(logits))[:, None], pairs].astype(np.float64) / temperature
    eligible -= eligible.max(axis=1, keepdims=True)
    e = np.exp(eligible)
    return e / e.sum(axis=1, keepdims=True)


def score(logits, data, rows, temperature):
    _, pairs, targets = data
    probability = pair_probabilities(logits, pairs, temperature)
    chosen = probability.argmax(axis=1)
    best = targets[np.arange(len(rows)), chosen] > 0
    passed = sum(r["option_finite_passed"][int(c)] for r, c in zip(rows, chosen))
    return {"observed_views": len(rows), "finite_best_set_selected": int(best.sum()),
            "finite_cases_passed": passed, "finite_cases_total": sum(len(r["finite_cases"]) for r in rows),
            "multiple_best_finite_labels": sum(len(r["best_finite_labels"]) > 1 for r in rows),
            "original_intention_label_agreement": sum(r["eligible_labels"][int(c)] == r["intention_label"] for r, c in zip(rows, chosen)),
            "paired_soft_target_nll": float(-(targets * np.log(probability.clip(1e-12))).sum(axis=1).mean()),
            "mean_probability_on_best_set": float((probability * (targets > 0)).sum(axis=1).mean())}


def fit(initial, data, rows, epochs, seed, qat):
    torch.manual_seed(seed)
    model = core.Model(qat=qat)
    model.load_state_dict(initial)
    model.to("mps")
    optimizer = torch.optim.AdamW(model.parameters(), lr=0.001, weight_decay=0.01)
    x, pairs, targets = [torch.from_numpy(v).to("mps") for v in data["train"]]
    cx, cp, cy = [torch.from_numpy(v).to("mps") for v in data["calibration"]]
    rng = np.random.default_rng(seed)
    best, best_loss, history, allocated_peak, driver_peak = None, float("inf"), [], 0, 0
    started = time.perf_counter()
    for epoch in range(epochs):
        order = rng.permutation(len(x))
        model.train()
        losses = []
        for begin in range(0, len(order), 256):
            index = torch.from_numpy(order[begin:begin + 256]).to("mps")
            optimizer.zero_grad(set_to_none=True)
            logits = model(x[index]).gather(1, pairs[index])
            loss = -(targets[index] * torch.log_softmax(logits, dim=1)).sum(dim=1).mean()
            loss.backward()
            optimizer.step()
            losses.append(float(loss.detach().cpu()))
        model.eval()
        with torch.no_grad():
            c_logits = model(cx).gather(1, cp)
            cal = float((-(cy * torch.log_softmax(c_logits, dim=1)).sum(dim=1).mean()).cpu())
        if cal < best_loss:
            best_loss = cal
            best = {key: value.detach().cpu().clone() for key, value in model.state_dict().items()}
        allocated_peak = max(allocated_peak, torch.mps.current_allocated_memory())
        driver_peak = max(driver_peak, torch.mps.driver_allocated_memory())
        history.append({"epoch": epoch + 1, "training_paired_nll": float(np.mean(losses)), "calibration_paired_nll": cal})
    torch.mps.synchronize()
    duration = time.perf_counter() - started
    model.cpu()
    model.load_state_dict(best)
    return model, {"epochs": epochs, "optimizer_steps": epochs * math.ceil(len(x) / 256),
                   "unique_training_views": len(x), "loop_wall_seconds": duration,
                   "mps_allocated_peak_sampled_bytes": allocated_peak, "mps_driver_peak_sampled_bytes": driver_peak,
                   "process_lifetime_peak_rss_bytes": resource.getrusage(resource.RUSAGE_SELF).ru_maxrss,
                   "selection": "minimum calibration paired soft-target NLL; test excluded", "history": history}


def main():
    p = argparse.ArgumentParser()
    p.add_argument("--dataset", type=Path, required=True)
    p.add_argument("--parent", type=Path, required=True)
    p.add_argument("--output", type=Path, required=True)
    p.add_argument("--source-revision", required=True)
    p.add_argument("--epochs", type=int, default=20)
    p.add_argument("--seed", type=int, default=20261004)
    args = p.parse_args()
    root = Path(__file__).resolve().parents[1]
    core.require(1 <= args.epochs <= 40 and torch.backends.mps.is_available(), "bounded MPS training required")
    core.require(not args.output.exists(), "fresh output required")
    head = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=root, text=True).strip()
    core.require(head == args.source_revision, "committed source revision required")
    subprocess.run(["git", "diff", "--quiet", "HEAD", "--", "training", "data/feedback-path-v1", "internal/feedbackstudy", "cmd/feedback-curriculum"], cwd=root, check=True)
    raw, manifest_raw = args.dataset.read_bytes(), args.dataset.with_name("manifest.json").read_bytes()
    manifest = json.loads(manifest_raw, object_pairs_hook=core.unique_pairs)
    core.require(manifest["schema"] == "gooo/feedback-path-curriculum/v1" and core.sha(raw) == manifest["dataset_sha256"], "dataset binding")
    all_rows = [json.loads(v, object_pairs_hook=core.unique_pairs) for v in raw.splitlines()]
    core.require(len(all_rows) == 6240 and len({r["id"] for r in all_rows}) == 6240, "full curriculum and unique IDs required")
    for r in all_rows:
        core.require(r["input_sha256"] == core.sha(r["text"].encode()) and r["original_text_sha256"] == core.sha(r["original_text"].encode())
                     and r["text"].endswith("\nintent: " + r["original_text"]) and not r["finite_targets_prove_intent"]
                     and not r["ci_hint_is_authority"] and 1 <= len(r["best_finite_labels"]) <= 2
                     and set(r["best_finite_labels"]) <= set(r["eligible_labels"]), "finite target or original intention drift")
        core.require(r["training_eligible"] == (1 <= len(r["text"].encode()) <= 512), "input eligibility drift")
    rows = {s: [r for r in all_rows if r["split"] == s and r["training_eligible"]] for s in ("train", "calibration", "test")}
    for field in ("instruction_id", "program_id", "template_id", "configuration_id"):
        for a, b in (("train", "calibration"), ("train", "test"), ("calibration", "test")):
            core.require(not {r[field] for r in rows[a]} & {r[field] for r in rows[b]}, "group leakage")
    for source, digest in manifest["source_sha256"].items():
        core.require(core.sha((root / source).read_bytes()) == digest, "generator source mismatch")
    initial, parent = parent_state(args.parent)
    data = {key: arrays(value) for key, value in rows.items()}
    args.output.mkdir(parents=True)
    sources = {v: core.sha((root / v).read_bytes()) for v in ("training/train_feedback_path_v1.py", "training/train_typed_path_v1.py", "training/train_pilot_v2.py")}
    core.save_json(args.output / "preexecution.json", {"schema": "gooo/feedback-path-training-preexecution/v1",
        "source_revision": head, "dataset_sha256": core.sha(raw), "manifest_sha256": core.sha(manifest_raw),
        "training_sources_sha256": sources, "parent": parent, "epochs_per_trainable_variant": args.epochs, "seed": args.seed,
        "device": "mps", "torch": torch.__version__, "numpy": np.__version__, "split_eligible_views": {s: len(v) for s, v in rows.items()},
        "feature_version": "positioned_intent_ngrams_v1", "objective": "typed eligible-pair soft targets; tied best finite paths have equal target mass",
        "test_used_for_checkpoint_or_temperature_selection": False,
        "scope": "Fine tune the existing small structural model on finite-feedback contexts. Source/test groups reuse the development curriculum; no untouched holdout, arbitrary-language compiler or online optimizer claim."})
    torch.set_num_threads(2)
    models, training, results, parity = {}, {}, {}, []
    models["fp32"], training["fp32"] = fit(initial, data, rows, args.epochs, args.seed, False)
    models["qat_ternary"], training["qat_ternary"] = fit(copy.deepcopy(models["fp32"].state_dict()), data, rows, args.epochs, args.seed + 1, True)
    models["ptq_ternary"] = models["fp32"]
    for variant, model in models.items():
        directory = args.output / "models" / variant
        core.export_model(model, variant, directory, 1.0, 1.0)
        cal_logits = core.exported_logits(directory, data["calibration"][0])
        temperature = min((0.5, 1.0, 2.0, 4.0), key=lambda t: score(cal_logits, data["calibration"], rows["calibration"], t)["paired_soft_target_nll"])
        meta = json.loads((directory / "model.json").read_text())
        meta.update(schema="gooo/tiny-path-decision-model/v1", feature_version="positioned_intent_ngrams_v1", temperature=temperature)
        core.save_json(directory / "model.json", meta)
        results[variant] = {"temperature": temperature, "confidence_threshold": 1.0,
            "weights_bytes": (directory / "weights.bin").stat().st_size,
            "scores": {s: score(core.exported_logits(directory, d[0]), d, rows[s], temperature) for s, d in data.items()}}
        logits = core.exported_logits(directory, data["test"][0])
        probs = core.probabilities(logits, temperature)
        for i in np.linspace(0, len(rows["test"]) - 1, 32, dtype=int):
            parity.append({"variant": variant, "text": rows["test"][i]["text"], "features": data["test"][0][i].tolist(),
                           "logits": logits[i].tolist(), "probabilities": probs[i].tolist(), "selected_label": PATH_LABELS[int(probs[i].argmax())]})
    core.save_json(args.output / "report.json", {"schema": "gooo/feedback-path-training-report/v1", "status": "TRAINED_AND_EXPORTED",
        "preexecution_sha256": core.sha((args.output / "preexecution.json").read_bytes()), "training": training, "variants": results,
        "limitations": ["Finite acceptable sets do not prove every natural-language intention", "6240 views reuse 2080 existing instruction groups",
                        "Calibration/test templates and configurations are disjoint from training; reused test evaluation is development data",
                        "Global confidence threshold stays 1.0; this model is experimental for compiler-owned pair ranking and bounded TDD",
                        "Packed ternary storage decodes to int8; no packed arithmetic speedup or production online learning claim"]})
    core.save_json(args.output / "go-parity.json", {"schema": "gooo/typed-path-parity/v1", "rows": parity})
    print(json.dumps({"status": "TRAINED_AND_EXPORTED", "optimizer_steps": sum(v["optimizer_steps"] for v in training.values())}))


if __name__ == "__main__":
    main()
