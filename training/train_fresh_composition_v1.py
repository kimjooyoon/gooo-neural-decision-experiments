#!/usr/bin/env python3
"""Frozen fresh composition optimization/export only; orchestration stays Go."""
import argparse
import copy
import json
import subprocess
from pathlib import Path

import numpy as np
import torch
import train_pilot_v2 as core
import train_bilingual_judgment_v1 as paired
from train_typed_path_v1 import PATH_LABELS
import split_features_v2 as v2
import semantic_features_v3 as v3

core.LABELS = PATH_LABELS
DATASET = "44393fb6ae27d1dc75c9706ea50077fab8e2573928e4f59422e3e3a65aa79620"
MANIFEST = "92ee4dff08db92ec649fb5eb74a1e3ca3caa17111b2df88cca94bf90ac3b5148"
PROTOCOL = "8b38e7c333a1b46dc44c3ed57a8b0a6125ffa1b4f74a1161adb970c342fd5613"
ARMS = {"v2": v2, "v3": v3}


def load_pairs(directory):
    raw = (directory / "dataset.jsonl").read_bytes()
    manifest_raw = (directory / "manifest.json").read_bytes()
    manifest = json.loads(manifest_raw, object_pairs_hook=core.unique_pairs)
    audit = json.loads((directory / "audit.json").read_bytes(), object_pairs_hook=core.unique_pairs)
    core.require(core.sha(raw) == DATASET == manifest["dataset_sha256"] and core.sha(manifest_raw) == MANIFEST,
                 "frozen valid raw-v2 curriculum required")
    core.require(audit["status"] == "PASS" and audit["manifest_sha256"] == MANIFEST and
                 audit["decision_rows_audited"] == 9216, "independent native byte/oracle audit required")
    rows = [json.loads(line, object_pairs_hook=core.unique_pairs) for line in raw.splitlines()]
    core.require(len(rows) == len({r["id"] for r in rows}) == 9216, "fixed distinct row count")
    grouped = {}
    for row in rows:
        text = row["input"]["text"]
        core.require(core.sha(text.encode()) == row["input"]["input_sha256"].removeprefix("sha256:"), "full input SHA")
        core.require(0 <= row["configuration"] < 48 and row["coordinate"] in (0, 1), "fixed configuration/coordinate")
        core.require(row["split"] == ("train" if row["configuration"] < 32 else
                                     "calibration" if row["configuration"] < 40 else "development"), "fixed split")
        core.require(row["finite_soft_targets"] == row["full_contract_target"]["coordinate_marginals"][row["coordinate"]],
                     "complete finite coordinate target retained")
        pair = grouped.setdefault(row["bilingual_decision_pair"], {})
        core.require(row["language"] in ("en", "ko") and row["language"] not in pair, "distinct paired language")
        pair[row["language"]] = row
    pairs = []
    for pair in grouped.values():
        core.require(set(pair) == {"en", "ko"}, "both language views required")
        a, b = pair["en"], pair["ko"]
        for field in ("program_contract_group", "family", "configuration", "desired_mask", "split",
                      "feature_version", "coordinate", "eligible_labels", "finite_soft_targets", "full_contract_target", "source_sha256"):
            core.require(a[field] == b[field], "paired source/target drift")
        core.require(len(a["eligible_labels"]) == 2 and abs(sum(a["finite_soft_targets"]) - 1) < 1e-8, "target mass")
        pairs.append([a, b])
    return pairs


def arrays(pairs, module):
    return (np.asarray([[module.features(r["input"]["text"]) for r in p] for p in pairs], dtype=np.float32),
            np.asarray([[PATH_LABELS.index(v) for v in p[0]["eligible_labels"]] for p in pairs], dtype=np.int64),
            np.asarray([p[0]["finite_soft_targets"] for p in pairs], dtype=np.float32))


def finite_nll(logits, indices, targets, temperature):
    z = logits[np.arange(len(logits))[:, None], indices].astype(np.float64) / temperature
    z -= z.max(axis=1, keepdims=True)
    p = np.exp(z)
    p /= p.sum(axis=1, keepdims=True)
    return float(-(targets * np.log(p.clip(1e-12))).sum(axis=1).mean())


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--source-revision", required=True)
    parser.add_argument("--curriculum", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[1]
    head = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=root, text=True).strip()
    core.require(head == args.source_revision and not subprocess.check_output(["git", "status", "--porcelain"], cwd=root).strip(),
                 "clean exact committed training source required")
    core.require(torch.backends.mps.is_available() and not args.output.exists(), "MPS and fresh output required")
    protocol = root / "docs/fresh-composition-semantic-v3-preregistration-20261002.md"
    core.require(core.sha(protocol.read_bytes()) == PROTOCOL, "immutable preregistration required")
    pairs = load_pairs(args.curriculum)
    torch.set_num_threads(2)
    torch.manual_seed(20261022)
    initial = {k: v.detach().clone() for k, v in core.Model().state_dict().items()}
    initial_sha = core.sha(b"".join(v.numpy().astype("<f4").tobytes() for v in initial.values()))
    sources = {"training/" + f: core.sha((root / "training" / f).read_bytes()) for f in
               ("train_fresh_composition_v1.py", "semantic_features_v3.py", "split_features_v2.py",
                "train_bilingual_judgment_v1.py", "train_pilot_v2.py", "train_typed_path_v1.py")}
    args.output.mkdir(parents=True)
    core.save_json(args.output / "preexecution.json", {"schema": "gooo/fresh-composition-training-preexecution/v1",
        "source_revision": head, "sources_sha256": sources, "dataset_sha256": DATASET, "curriculum_manifest_sha256": MANIFEST,
        "protocol_sha256": PROTOCOL, "initial_state_sha256": initial_sha, "initial_seed": 20261022, "shuffle_seed": 20261023,
        "device": "mps", "torch": torch.__version__, "numpy": np.__version__, "maximum_optimizer_steps": 960,
        "initialization": "matched own random state; no pretrained, Laya or earlier own-model weights",
        "consistency_weight": 0.0, "gpu_utilization_measured": False})
    reports, exports = {}, {}
    for arm, module in ARMS.items():
        groups = {s: [p for p in pairs if p[0]["feature_version"] == module.VERSION and p[0]["split"] == s]
                  for s in ("train", "calibration", "development")}
        core.require({s: len(v) for s, v in groups.items()} == {"train": 1536, "calibration": 384, "development": 384}, "fixed pair counts")
        for field in ("program_contract_group", "template_pair_id"):
            split_sets = {s: {r[field] for p in v for r in p} for s, v in groups.items()}
            core.require(all(not split_sets[a] & split_sets[b] for a, b in
                             (("train", "calibration"), ("train", "development"), ("calibration", "development"))), "split group leakage")
        data = {s: arrays(v, module) for s, v in groups.items()}
        fp, fp_report = paired.fit(copy.deepcopy(initial), data, 20, 20261023, 0.0, False)
        qat, qat_report = paired.fit(copy.deepcopy(fp.state_dict()), data, 20, 20261023, 0.0, True)
        reports[arm], exports[arm], parity = {"fp32": fp_report, "qat_ternary": qat_report}, {}, []
        for variant, model in (("fp32", fp), ("ptq_ternary", fp), ("qat_ternary", qat)):
            directory = args.output / arm / "models" / variant
            core.export_model(model, variant, directory, 1.0, 1.0)
            cx, ci, cy = data["calibration"]
            cx, ci, cy = cx.reshape(-1, 256), np.repeat(ci, 2, axis=0), np.repeat(cy, 2, axis=0)
            cal_logits = core.exported_logits(directory, cx)
            temperature = min((0.5, 1.0, 2.0, 4.0), key=lambda t: finite_nll(cal_logits, ci, cy, t))
            meta = json.loads((directory / "model.json").read_text())
            meta.update(schema="gooo/tiny-path-decision-model/v1", feature_version=module.VERSION, temperature=temperature)
            core.save_json(directory / "model.json", meta)
            exports[arm][variant] = {"metadata_sha256": core.sha((directory / "model.json").read_bytes()),
                                    "weights_sha256": core.sha((directory / "weights.bin").read_bytes()),
                                    "packed_weights_bytes": (directory / "weights.bin").stat().st_size}
            dev_rows = [r for p in groups["development"] for r in p]
            selected = np.linspace(0, len(dev_rows) - 1, 32, dtype=int)
            x = np.stack([module.features(dev_rows[i]["input"]["text"]) for i in selected])
            logits = core.exported_logits(directory, x)
            probabilities = core.probabilities(logits, temperature)
            for j, i in enumerate(selected):
                parity.append({"variant": variant, "row_id": dev_rows[i]["id"], "text": dev_rows[i]["input"]["text"],
                               "features": x[j].tolist(), "logits": logits[j].tolist(), "probabilities": probabilities[j].tolist(),
                               "selected_label": PATH_LABELS[int(probabilities[j].argmax())]})
        core.save_json(args.output / arm / "go-parity.json", {"schema": "gooo/typed-path-parity/v1", "rows": parity})
    steps = sum(v["optimizer_steps"] for arm in reports.values() for v in arm.values())
    core.require(steps == 960, "frozen optimizer budget")
    core.save_json(args.output / "report.json", {"schema": "gooo/fresh-composition-training-report/v1", "status": "TRAINED_AND_EXPORTED",
        "preexecution_sha256": core.sha((args.output / "preexecution.json").read_bytes()), "optimizer_steps": steps,
        "training": reports, "exports": exports, "default_model_promoted": False,
        "scope": "Fresh six compositions; 3072 train decision views per arm with paired finite marginal targets. Matched random state and fixed 20+20 epochs. Go parity/complete assembly/native execution remain separate evidence; no Laya weights or online learning."})
    print(json.dumps({"status": "TRAINED_AND_EXPORTED", "optimizer_steps": steps, "exports": 6}))


if __name__ == "__main__":
    main()
