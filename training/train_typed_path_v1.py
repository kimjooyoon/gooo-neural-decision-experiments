#!/usr/bin/env python3
"""Offline structural-head transfer training; all deployed execution is Go."""
import argparse
import copy
import json
import math
from pathlib import Path

import numpy as np
import torch
import train_pilot_v2 as core
from train_compiler_prov_v3 import fit

PATH_LABELS = ["reference_first", "reference_second", "assign_first", "assign_second",
               "layout_forward", "layout_reverse", "schedule_forward", "schedule_reverse"]
OLD_LABELS = list(core.LABELS)
core.LABELS = PATH_LABELS


def positioned_features(text):
    raw = text.encode("utf-8")
    core.require(1 <= len(raw) <= 512, "bounded positioned feature input")
    marker = raw.rfind(b"intent: ")
    raw = bytes(c + 32 if 65 <= c <= 90 else c for c in raw)
    context, intent = (raw[:marker], raw[marker + 8:]) if marker >= 0 else (b"", raw)
    counts = np.zeros(256, dtype=np.float32)
    for width in (2, 3):
        for data, weight, positioned in ((intent, 1.0, True), (context, 0.125, False)):
            for start in range(max(0, len(data) - width + 1)):
                value = 2166136261
                for c in data[start:start + width]:
                    value = ((value ^ c) * 16777619) & 0xffffffff
                index = (start * 4 // len(data)) * 64 + value % 64 if positioned else value % 256
                counts[index] += np.float32(weight)
    norm = math.sqrt(sum(float(c) ** 2 for c in counts))
    if norm:
        counts *= np.float32(1 / norm)
    return counts


def transferred_state(parent, seed):
    raw = (parent / "model.json").read_bytes()
    metadata = json.loads(raw, object_pairs_hook=core.unique_pairs)
    blob = (parent / metadata["weights_file"]).read_bytes()
    core.require(metadata["schema"] == "gooo/tiny-ir-decision-model/v1" and metadata["variant"] == "fp32"
                 and metadata["labels"] == OLD_LABELS and core.sha(blob) == metadata["weights_sha256"], "invalid FP32 parent")
    torch.manual_seed(seed)
    state = core.Model().state_dict()
    transferred = []
    for tensor in metadata["tensors"]:
        if tensor["name"] not in ("w1", "b1"):
            continue
        core.require(tensor["encoding"] == "float32_le", "parent hidden tensor encoding")
        array = np.frombuffer(blob[tensor["offset"]:tensor["offset"] + tensor["bytes"]], dtype="<f4").copy()
        core.require(np.isfinite(array).all(), "nonfinite parent tensor")
        key = "first.weight" if tensor["name"] == "w1" else "first.bias"
        state[key] = torch.from_numpy(array.reshape(tuple(state[key].shape)))
        transferred.append(tensor["name"])
    core.require(sorted(transferred) == ["b1", "w1"], "missing parent hidden tensors")
    return state, {"metadata_sha256": core.sha(raw), "weights_sha256": core.sha(blob),
                   "transferred_tensors": ["w1", "b1"], "new_random_head_tensors": ["w2", "b2"],
                   "operation_labels_remapped_to_path_labels": False}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--dataset", type=Path, required=True)
    parser.add_argument("--parent", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--epochs", type=int, default=60)
    parser.add_argument("--seed", type=int, default=20261002)
    parser.add_argument("--feature-version", choices=("uniform", "positioned_intent_ngrams_v1"), default="uniform")
    parser.add_argument("--initialization", choices=("hidden_transfer", "random"), default="hidden_transfer")
    args = parser.parse_args()
    core.require(1 <= args.epochs <= 120 and torch.backends.mps.is_available(), "bounded epochs and MPS GPU required")
    core.require(not args.output.exists(), "output must be fresh")
    raw = args.dataset.read_bytes()
    manifest_raw = args.dataset.with_name("manifest.json").read_bytes()
    manifest = json.loads(manifest_raw, object_pairs_hook=core.unique_pairs)
    core.require(manifest["schema"] == "gooo/typed-path-curriculum/v1" and core.sha(raw) == manifest["dataset_sha256"], "dataset pin mismatch")
    records = [json.loads(line, object_pairs_hook=core.unique_pairs) for line in raw.splitlines()]
    fields = {"id", "instruction_id", "program_id", "template_id", "configuration_id", "configuration_index",
              "family", "language", "split", "view", "text", "label"}
    core.require(len(records) == 6240 and len({r["id"] for r in records}) == 6240 and len({r["text"] for r in records}) == 6240, "invalid total rows or duplicates")
    core.require(all(set(r) == fields and r["label"] in PATH_LABELS and 1 <= len(r["text"].encode()) <= 512
                     and r["language"] in ("en", "ko") and r["view"] in ("plain", "gooo", "prov")
                     and r["family"] in manifest["families"] for r in records), "invalid row shape or enum")
    rows = {split: [r for r in records if r["split"] == split] for split in ("train", "calibration", "test")}
    core.require({key: len(value) for key, value in rows.items()} == {"train": 4800, "calibration": 480, "test": 960} == manifest["rows"], "split counts")
    for key in ("instruction_id", "template_id", "configuration_id", "program_id"):
        for a, b in (("train", "calibration"), ("train", "test"), ("calibration", "test")):
            core.require(not {r[key] for r in rows[a]} & {r[key] for r in rows[b]}, "group leakage: " + key)
    for split, values in rows.items():
        lo, hi = manifest["configuration_ranges"][split]
        core.require(all(lo <= r["configuration_index"] <= hi for r in values), "configuration split mismatch")
        groups = {r["instruction_id"] for r in values}
        core.require(len(groups) == manifest["unique_instructions"][split] and all(sum(r["instruction_id"] == g for r in values) == 3 for g in groups), "view grouping mismatch")
    root = Path(__file__).resolve().parents[1]
    for path, expected in manifest["generator_source_sha256"].items():
        core.require(core.sha((root / path).read_bytes()) == expected, "generator source drift")
    feature_fn = positioned_features if args.feature_version != "uniform" else core.features
    xs = {key: np.stack([feature_fn(r["text"]) for r in value]) for key, value in rows.items()}
    ys = {key: np.array([PATH_LABELS.index(r["label"]) for r in value], dtype=np.int64) for key, value in rows.items()}
    initial, parent = transferred_state(args.parent, args.seed)
    if args.initialization == "random":
        torch.manual_seed(args.seed)
        initial = core.Model().state_dict()
        parent.update(transferred_tensors=[], initialized_from_parent=False)
    else:
        parent["initialized_from_parent"] = True
    args.output.mkdir(parents=True)
    sources = {path: core.sha((root / path).read_bytes()) for path in
               ("training/train_typed_path_v1.py", "training/train_compiler_prov_v3.py", "training/train_pilot_v2.py")}
    core.save_json(args.output / "preexecution.json", {
        "schema": "gooo/typed-path-training-preexecution/v1", "dataset_sha256": core.sha(raw), "manifest_sha256": core.sha(manifest_raw),
        "parent": parent, "training_sources_sha256": sources, "epochs_per_variant": args.epochs, "seed": args.seed,
        "device": "mps", "torch": torch.__version__, "numpy": np.__version__, "test_used_for_training_or_checkpoint_selection": False,
        "feature_version": args.feature_version, "initialization": args.initialization,
        "contract": "Separate structural ABI and new head. Positioned features change input-coordinate semantics; transferring the old hidden tensors is an initialization experiment, not an equivalent feature-layer reuse.",
        "development_test_disclosure": "The original uniform experiment's test results informed this representation change; reused 960 views are development evaluation, not an untouched final holdout",
        "split_counts": manifest["rows"], "unique_test_instructions": 320, "unique_test_program_configurations": 160})
    torch.set_num_threads(2)
    models, training = {}, {}
    models["fp32"], training["fp32"] = fit(initial, xs, ys, rows, args.epochs, args.seed, False)
    models["qat_ternary"], training["qat_ternary"] = fit(copy.deepcopy(models["fp32"].state_dict()), xs, ys, rows, args.epochs, args.seed + 1, True)
    models["ptq_ternary"] = models["fp32"]
    for value in training.values():
        value["repair_training_rows"] = 0
        value.pop("repair_exposures_per_epoch_per_row")
    results, parity = {}, []
    for variant in ("fp32", "ptq_ternary", "qat_ternary"):
        directory = args.output / "models" / variant
        core.export_model(models[variant], variant, directory, 1.0, 0.0)
        temperature, threshold = core.calibrate(core.exported_logits(directory, xs["calibration"]), ys["calibration"])
        metadata = json.loads((directory / "model.json").read_text())
        metadata.update(schema="gooo/tiny-path-decision-model/v1", temperature=temperature, confidence_threshold=threshold)
        if args.feature_version != "uniform":
            metadata["feature_version"] = args.feature_version
        core.save_json(directory / "model.json", metadata)
        results[variant] = {"scores": {key: core.scores(core.exported_logits(directory, value), ys[key], temperature) for key, value in xs.items()},
                            "temperature": temperature, "confidence_threshold": threshold, "weights_bytes": (directory / "weights.bin").stat().st_size,
                            "test_by_family": {}, "test_by_view": {}}
        logits = core.exported_logits(directory, xs["test"])
        probs = core.probabilities(logits, temperature)
        for field, dest in (("family", "test_by_family"), ("view", "test_by_view")):
            for key in sorted({r[field] for r in rows["test"]}):
                indices = [i for i, r in enumerate(rows["test"]) if r[field] == key]
                results[variant][dest][key] = core.scores(logits[indices], ys["test"][indices], temperature)
        for index in np.linspace(0, len(rows["test"]) - 1, 32, dtype=int):
            parity.append({"variant": variant, "text": rows["test"][index]["text"], "features": xs["test"][index].tolist(),
                           "logits": logits[index].tolist(), "probabilities": probs[index].tolist(), "selected_label": PATH_LABELS[int(probs[index].argmax())]})
    report = {"schema": "gooo/typed-path-training-report/v1", "status": "TRAINED_AND_EXPORTED", "preexecution_sha256": core.sha((args.output / "preexecution.json").read_bytes()),
              "training": training, "variants": results,
              "limitations": ["960 test views represent 320 instructions and 160 structural program configurations",
                              "Uniform-model test results informed representation development; repeated evaluation is not an untouched final holdout",
                              "Synthetic templates and numerical configurations are disjoint; arbitrary natural language remains unmeasured",
                              "PROV-O context is vocabulary conditioning, not an OWL reasoner",
                              "Structural selection does not generate new identifiers or arbitrary source fragments",
                              "Packed ternary weights are decoded to int8 arrays at load; no packed arithmetic speedup is established"]}
    core.save_json(args.output / "report.json", report)
    core.save_json(args.output / "go-parity.json", {"schema": "gooo/typed-path-parity/v1", "rows": parity})
    print(json.dumps({"status": report["status"], "test_correct_of_960": {key: value["scores"]["test"]["correct"] for key, value in results.items()}}))


if __name__ == "__main__":
    main()
