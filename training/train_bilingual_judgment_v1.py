#!/usr/bin/env python3
"""Offline paired Gooo judgment tuning; inference and evaluation stay in Go."""
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
import train_feedback_path_v1 as feedback
from train_typed_path_v1 import PATH_LABELS, positioned_features

core.LABELS = PATH_LABELS


def pair_data(pairs, rows):
    grouped = [[rows[i] for i in p["row_ids"]] for p in pairs]
    x = np.asarray([[positioned_features(r["text"]) for r in g] for g in grouped], dtype=np.float32)
    indices = np.asarray([[PATH_LABELS.index(v) for v in p["eligible_labels"]] for p in pairs], dtype=np.int64)
    targets = np.asarray([p["finite_soft_targets"] for p in pairs], dtype=np.float32)
    return x, indices, targets


def loss_parts(logits, indices, targets):
    eligible = logits.gather(2, indices[:, None, :].expand(-1, 2, -1))
    logp = torch.log_softmax(eligible, dim=2)
    nll = -(targets[:, None, :] * logp).sum(dim=2).mean()
    probability = logp.exp()
    mixture = probability.mean(dim=1, keepdim=True).clamp_min(1e-12)
    js = (probability * (logp - mixture.log())).sum(dim=2).mean()
    return nll, js


def fit(initial, data, epochs, seed, weight, qat):
    torch.manual_seed(seed)
    model = core.Model(qat=qat).to("mps")
    model.load_state_dict(initial)
    optimizer = torch.optim.AdamW(model.parameters(), lr=0.001, weight_decay=0.01)
    train = [torch.from_numpy(v).to("mps") for v in data["train"]]
    calibration = [torch.from_numpy(v).to("mps") for v in data["calibration"]]
    rng = np.random.default_rng(seed)
    history, best, best_loss, allocated, driver = [], None, float("inf"), 0, 0
    started = time.perf_counter()
    for epoch in range(epochs):
        model.train()
        losses = []
        for begin in range(0, len(train[0]), 128):
            # The permutation is drawn once per epoch, independently of the arm.
            if begin == 0:
                order = rng.permutation(len(train[0]))
            ix = torch.from_numpy(order[begin:begin + 128]).to("mps")
            optimizer.zero_grad(set_to_none=True)
            logits = model(train[0][ix].reshape(-1, 256)).reshape(-1, 2, 8)
            nll, js = loss_parts(logits, train[1][ix], train[2][ix])
            loss = nll + weight * js
            loss.backward()
            optimizer.step()
            losses.append(float(loss.detach().cpu()))
        model.eval()
        with torch.no_grad():
            logits = model(calibration[0].reshape(-1, 256)).reshape(-1, 2, 8)
            nll, js = loss_parts(logits, calibration[1], calibration[2])
            nll, js = float(nll.cpu()), float(js.cpu())
        objective = nll + weight * js
        if objective < best_loss:
            best_loss = objective
            best = {k: v.detach().cpu().clone() for k, v in model.state_dict().items()}
        allocated = max(allocated, torch.mps.current_allocated_memory())
        driver = max(driver, torch.mps.driver_allocated_memory())
        history.append({"epoch": epoch + 1, "train_objective": float(np.mean(losses)),
                        "calibration_finite_nll": nll, "calibration_bilingual_js": js})
    torch.mps.synchronize()
    duration = time.perf_counter() - started
    model.cpu().load_state_dict(best)
    return model, {"epochs": epochs, "optimizer_steps": epochs * math.ceil(len(train[0]) / 128),
                   "training_pairs": len(train[0]), "training_views": len(train[0]) * 2,
                   "consistency_weight": weight, "loop_wall_seconds": duration,
                   "mps_allocated_peak_sampled_bytes": allocated, "mps_driver_peak_sampled_bytes": driver,
                   "process_lifetime_peak_rss_bytes": resource.getrusage(resource.RUSAGE_SELF).ru_maxrss,
                   "selection": "minimum calibration finite NLL plus weighted bilingual JS; development test excluded",
                   "history": history}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--source-revision", required=True)
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[1]
    core.require(torch.backends.mps.is_available() and not args.output.exists(), "MPS and fresh output required")
    head = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=root, text=True).strip()
    core.require(head == args.source_revision, "committed source revision required")
    core.require(not subprocess.check_output(["git", "status", "--porcelain"], cwd=root).strip(), "clean source required")
    dataset = root / "data/feedback-path-v1/dataset.jsonl"
    pair_file = root / "data/bilingual-gooo-pairs-v1/pairs.jsonl"
    manifest_raw = pair_file.with_name("manifest.json").read_bytes()
    manifest = json.loads(manifest_raw, object_pairs_hook=core.unique_pairs)
    raw, pair_raw = dataset.read_bytes(), pair_file.read_bytes()
    core.require(core.sha(raw) == manifest["dataset_sha256"] and core.sha(pair_raw) == manifest["pairs_sha256"], "data pins")
    for name, digest in manifest["source_sha256"].items():
        core.require(core.sha((root / name).read_bytes()) == digest, "pair generator source mismatch")
    rows = {r["id"]: r for r in (json.loads(v, object_pairs_hook=core.unique_pairs) for v in raw.splitlines())}
    pairs = [json.loads(v, object_pairs_hook=core.unique_pairs) for v in pair_raw.splitlines()]
    core.require(len(rows) == 6240 and len(pairs) == 1040 and len({p["id"] for p in pairs}) == 1040, "frozen counts")
    for p in pairs:
        a, b = [rows[i] for i in p["row_ids"]]
        core.require([a["language"], b["language"]] == ["en", "ko"] and a["split"] == b["split"] == p["split"]
                     and a["eligible_labels"] == b["eligible_labels"] == p["eligible_labels"]
                     and a["finite_cases"] == b["finite_cases"] and a["best_finite_labels"] == b["best_finite_labels"]
                     and a["intention_label"] == b["intention_label"] and a["original_view"] == b["original_view"] == "gooo"
                     and [core.sha(r["text"].encode()) for r in (a, b)] == p["input_sha256"], "pair binding")
        target = [float(label in a["best_finite_labels"]) / len(a["best_finite_labels"]) for label in p["eligible_labels"]]
        core.require(target == p["finite_soft_targets"], "ambiguous finite set target drift")
    grouped = {s: [p for p in pairs if p["split"] == s] for s in ("train", "calibration", "test")}
    core.require({s: len(v) for s, v in grouped.items()} == {"train": 800, "calibration": 80, "test": 160}, "split counts")
    for field in ("program_id", "template_pair_id"):
        for a, b in (("train", "calibration"), ("train", "test"), ("calibration", "test")):
            core.require(not {p[field] for p in grouped[a]} & {p[field] for p in grouped[b]}, "paired split leakage")
    parent_dir = root / "runs/feedback-path-soft-target-mps-20261001/models/fp32"
    initial, parent = feedback.parent_state(parent_dir)
    core.require(parent == {"metadata_sha256": "47bd3ed2037c8ba0af31ec4cad3a47fe1182af171845406aba01c5107f4b24b7",
                            "weights_sha256": "73eedef1d60f49129190683a41cbf82229c1e3603fd44cdd17792d015f499954"}, "fixed own parent")
    data = {s: pair_data(v, rows) for s, v in grouped.items()}
    sources = {name: core.sha((root / "training" / name).read_bytes()) for name in
               ("train_bilingual_judgment_v1.py", "train_feedback_path_v1.py", "train_typed_path_v1.py", "train_pilot_v2.py")}
    args.output.mkdir(parents=True)
    core.save_json(args.output / "preexecution.json", {"schema": "gooo/bilingual-judgment-training-preexecution/v1",
        "source_revision": head, "parent": parent, "dataset_sha256": core.sha(raw), "pairs_sha256": core.sha(pair_raw),
        "manifest_sha256": core.sha(manifest_raw), "training_sources_sha256": sources,
        "epochs_per_trainable_variant": 8, "seed": 20261008, "arms": {"control": 0.0, "paired": 0.25},
        "maximum_total_optimizer_steps": 224, "device": "mps", "torch": torch.__version__, "numpy": np.__version__,
        "scope": "Reused closed bilingual Gooo judgment pairs, finite-set supervision; agreement can be wrong. No new intentions, untouched holdout, online learning or Laya weight tuning."})
    torch.set_num_threads(2)
    training, exports = {}, {}
    for arm, weight in (("control", 0.0), ("paired", 0.25)):
        fp, fp_report = fit(copy.deepcopy(initial), data, 8, 20261008, weight, False)
        qat, qat_report = fit(copy.deepcopy(fp.state_dict()), data, 8, 20261009, weight, True)
        training[arm] = {"fp32": fp_report, "qat_ternary": qat_report}
        parity, exports[arm] = [], {}
        for variant, model in (("fp32", fp), ("ptq_ternary", fp), ("qat_ternary", qat)):
            directory = args.output / arm / "models" / variant
            core.export_model(model, variant, directory, 1.0, 1.0)
            cal_rows = [rows[i] for p in grouped["calibration"] for i in p["row_ids"]]
            flat_data = feedback.arrays(cal_rows)
            logits = core.exported_logits(directory, flat_data[0])
            # Temperature affects uncertainty; positive scaling cannot change eligible argmax.
            temperature = min((0.5, 1.0, 2.0, 4.0), key=lambda t: feedback.score(logits, flat_data, cal_rows, t)["paired_soft_target_nll"])
            meta = json.loads((directory / "model.json").read_text())
            meta.update(schema="gooo/tiny-path-decision-model/v1", feature_version="positioned_intent_ngrams_v1", temperature=temperature)
            core.save_json(directory / "model.json", meta)
            exports[arm][variant] = {"metadata_sha256": core.sha((directory / "model.json").read_bytes()),
                                    "weights_sha256": core.sha((directory / "weights.bin").read_bytes()),
                                    "packed_weights_bytes": (directory / "weights.bin").stat().st_size, "temperature": temperature}
            test_rows = [rows[i] for p in grouped["test"] for i in p["row_ids"]]
            features = np.stack([positioned_features(r["text"]) for r in test_rows])
            logits = core.exported_logits(directory, features)
            probs = core.probabilities(logits, temperature)
            for i in np.linspace(0, len(test_rows) - 1, 32, dtype=int):
                parity.append({"variant": variant, "text": test_rows[i]["text"], "features": features[i].tolist(),
                               "logits": logits[i].tolist(), "probabilities": probs[i].tolist(),
                               "selected_label": PATH_LABELS[int(probs[i].argmax())]})
        core.save_json(args.output / arm / "go-parity.json", {"schema": "gooo/typed-path-parity/v1", "rows": parity})
    steps = sum(v["optimizer_steps"] for arm in training.values() for v in arm.values())
    core.require(steps == 224, "bounded optimizer accounting")
    core.save_json(args.output / "report.json", {"schema": "gooo/bilingual-judgment-training-report/v1", "status": "TRAINED_AND_EXPORTED",
        "preexecution_sha256": core.sha((args.output / "preexecution.json").read_bytes()), "optimizer_steps": steps,
        "training": training, "exports": exports, "evaluation": "Separate Go parity, bilingual judgment and finite continuation reports required",
        "limitations": ["Control and paired arms share seed/data/initial FP32; QAT starts from each arm FP32",
                        "Agreement can select the same wrong path; finite tie targets do not prove linguistic intent",
                        "Existing development split reused, zero new independent intention groups",
                        "MPS memory samples and process lifetime RSS are not host GPU/CPU utilization",
                        "Ternary disk packing decodes to int8; weights are experimental without default promotion"]})
    print(json.dumps({"status": "TRAINED_AND_EXPORTED", "optimizer_steps": steps}))


if __name__ == "__main__":
    main()
