"""Offline MPS optimization/export; Go owns the full-input bank and execution."""
import argparse
import copy
import gc
import hashlib
import json
from pathlib import Path
import subprocess

import numpy as np
import torch
import train_own_three_feedback_v1 as base
from train_shared_three_judgment_v1 import SharedLocal, shared_initial
from full_input_evidence_v1 import Evidence

core = base.core
PROTOCOL = "docs/full-input-judgment-preregistration-20261003.md"
ARMS = ("positioned-original", "positioned-varied", "bag-original", "bag-varied")
VERSIONS = {"positioned": "triple_semantic_context_v3_joint_v1",
            "bag": "triple_semantic_context_bag_v4_joint_v1"}
FILES = {"preexecution.json", "initial-fp32.bin", "order-control.json", "rows.jsonl", "rejections.jsonl",
         "positioned-f32le.bin", "bag-f32le.bin"}
INITIAL_SHA = "fa8af3f51b719cd11b99cd4ff7f89f64582695dc89ead5c2e09701c187a6267f"


def file_pin(path):
    core.require(path.is_file() and not path.is_symlink(), "regular pinned input")
    digest, size = hashlib.sha256(), 0
    with path.open("rb") as data:
        while chunk := data.read(1 << 20):
            digest.update(chunk)
            size += len(chunk)
            core.require(size <= 768 << 20, "bounded pinned input")
    return {"bytes": size, "sha256": digest.hexdigest()}


def load_inputs(directory, audit_path, revision, protocol_sha):
    manifest = base.read_json(directory / "manifest.json")
    audit = base.read_json(audit_path)
    count = manifest["prepared_rows"]
    core.require(manifest["schema"] == "gooo/full-input-training-bank/v1" and
                 manifest["status"] == "PREPARED_PENDING_INDEPENDENT_REPLAY" and
                 manifest["source_revision"] == revision and manifest["protocol_sha256"] == protocol_sha and
                 manifest["source_rows"] == 10739 and 10739 <= count <= 49599 and
                 manifest["feature_dim"] == 768 and manifest["parameters"] == 2072 and
                 manifest["groups"] == {"train": 1024, "calibration": 256, "development": 256} and
                 manifest["feature_versions"] == VERSIONS and
                 manifest["planned_optimizer_updates"] == 6400 and
                 manifest["optimizer_updates"] == manifest["model_predictions"] == 0 and
                 manifest["initial_state_sha256"] == INITIAL_SHA and set(manifest["files"]) == FILES,
                 "frozen full-input preparation contract")
    core.require(audit["schema"] == "gooo/full-input-training-bank-replay/v1" and
                 audit["status"] == "PASS" and audit["source_revision"] == revision and
                 audit["manifest_sha256"] == file_pin(directory / "manifest.json")["sha256"] and
                 audit["protocol_sha256"] == protocol_sha and audit["source_rows"] == 10739 and
                 audit["prepared_rows"] == count and audit["feature_values_recomputed"] == count*768*2 and
                 audit["model_predictions"] == audit["optimizer_updates"] == audit["native_executions"] == 0,
                 "complete independent Go bank replay required")
    for name in sorted(FILES):
        core.require(file_pin(directory / name) == manifest["files"][name], "Go feature bank digest")
    rows = []
    with (directory / "rows.jsonl").open("rb") as data:
        for line in data:
            core.require(len(line) <= 1 << 20, "bounded prepared row")
            rows.append(json.loads(line, object_pairs_hook=core.unique_pairs))
    core.require(len(rows) == count and all(r["feature_row_index"] == i for i, r in enumerate(rows)),
                 "complete ordered prepared bank")
    # These parity vectors never select a checkpoint, temperature or update.
    parity = []
    for language in ("en", "ko"):
        parity.extend([r for r in rows if r["split"] == "development" and r["language"] == language][:16])
        for form in ("original", "request-prefix", "please-prefix", "request-suffix", "please-suffix"):
            chosen = [r for r in rows if r["split"] == "train" and r["phase"] == "feedback" and
                      r["language"] == language and r["form"] == form][:2]
            core.require(len(chosen) == 2, "fixed feedback parity coverage")
            parity.extend(chosen)
    core.require(len(parity) == 52, "32 development + 20 feedback parity vectors")
    return manifest, rows, parity


def arm_data(rows, x, arm):
    varied = arm.endswith("-varied")
    # Original arms recover the frozen teacher-state weights before augmentation.
    selected = [dict(row) for row in rows if varied or row["form"] == "original"]
    features = x[[row["feature_row_index"] for row in selected]]
    if not varied:
        for row in selected:
            row["feedback_arm_row_weight"] = row["original_feedback_arm_row_weight"]
    return {split: base.arrays(selected, features, split, "set-feedback")
            for split in ("train", "calibration")}


def export(model, variant, directory, calibration, parity, x, temperature, version):
    pin, records = base.export(model, variant, directory, calibration, parity, x, temperature)
    # The frozen exporter computes the identical expanded tensors; only the
    # explicit feature identity differs. Re-pin final metadata after setting it.
    metadata = base.read_json(directory / "model.json")
    metadata["feature_version"] = version
    core.save_json(directory / "model.json", metadata)
    pin["metadata_sha256"] = file_pin(directory / "model.json")["sha256"]
    pin["feature_version"] = version
    for row, original in zip(records, parity, strict=True):
        row["feature_version"] = version
        row["form"] = original["form"]
        row["input_sha256"] = original["input_sha256"]
    return pin, records


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--prepared", type=Path, required=True)
    parser.add_argument("--preparation-audit", type=Path, required=True)
    parser.add_argument("--storage-preflight", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--source-revision", required=True)
    parser.add_argument("--protocol-sha256", required=True)
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[1]
    head = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=root, text=True).strip()
    core.require(head == args.source_revision and
                 not subprocess.check_output(["git", "status", "--porcelain"], cwd=root).strip(),
                 "clean exact published optimizer source")
    protocol_sha = file_pin(root / PROTOCOL)["sha256"]
    core.require(protocol_sha == args.protocol_sha256, "published protocol pin")
    core.require(torch.backends.mps.is_available(), "actual local MPS required")
    core.FEATURES, core.HIDDEN, core.LIMIT, core.LABELS = 768, 24, 1600, [f"mask_{i}" for i in range(8)]
    manifest, rows, parity_inputs = load_inputs(args.prepared, args.preparation_audit, head, protocol_sha)
    initial = base.state_from_blob(args.prepared / "initial-fp32.bin")
    core.require(base.state_sha(initial) == INITIAL_SHA, "byte-exact fresh Go initializer")
    initial = shared_initial(initial)
    evidence = Evidence(root, args.output, base.read_json(args.storage_preflight), head)
    sources = {name: file_pin(root / "training" / name)["sha256"] for name in
               ("train_full_input_judgment_v1.py", "full_input_evidence_v1.py",
                "train_shared_three_judgment_v1.py", "shared_evidence_v1.py",
                "train_own_three_feedback_v1.py", "train_pilot_v2.py")}
    evidence.save(args.output / "preexecution.json", {
        "schema": "gooo/full-input-judgment-training/v1", "source_revision": head,
        "protocol_sha256": protocol_sha, "sources_sha256": sources,
        "prepared_manifest_sha256": file_pin(args.prepared / "manifest.json")["sha256"],
        "preparation_audit_sha256": file_pin(args.preparation_audit)["sha256"],
        "storage_preflight_sha256": file_pin(args.storage_preflight)["sha256"],
        "arms": ARMS, "feature_versions": VERSIONS, "trainable_parameters_per_arm": 2072,
        "initial_state_sha256": base.state_sha(initial), "go_initializer_sha256": INITIAL_SHA,
        "initial_seed": base.INITIAL_SEED, "shuffle_seed": base.SHUFFLE_SEED,
        "planned_optimizer_updates": 6400, "optimizer": "AdamW", "learning_rate": .001,
        "weight_decay": .01, "temperatures": base.TEMPERATURES, "parity_vectors_per_export": 52,
        "device": "mps", "torch": torch.__version__, "numpy": np.__version__,
        "prior_retained_bytes": evidence.legacy, "remaining_study_bytes": evidence.cap,
        "new_study_cap_bytes": 768 << 20, "whole_own_three_cap_bytes": 3 << 30,
        "gpu_utilization_measured": False, "host_cpu_causal_delta_measured": False,
        "default_model_promoted": False,
        "scope": "Full-input interventions over the previously observed source/teacher cohort; unchanged original calibration."}, root)
    original_model = core.Model
    reports, exports, updates = {}, {}, 0
    try:
        core.Model = SharedLocal
        for arm in ARMS:
            representation = arm.split("-", 1)[0]
            x = np.memmap(args.prepared / (representation + "-f32le.bin"), dtype="<f4",
                          mode="r", shape=(manifest["prepared_rows"], 768))
            core.require(np.isfinite(x).all(), "finite prepared features")
            data = arm_data(rows, x, arm)
            with evidence.writers():
                fp, fp_report = base.fit(copy.deepcopy(initial), data, arm, False, args.output / arm, root)
                updates += fp_report["optimizer_steps"]
                qat, qat_report = base.fit(copy.deepcopy(fp.state_dict()), data, arm, True, args.output / arm, root)
                updates += qat_report["optimizer_steps"]
            reports[arm], exports[arm], parity = {"fp32": fp_report, "qat_ternary": qat_report}, {}, []
            for variant, model, temperature in (("fp32", fp, fp_report["selected_temperature"]),
                                                ("ptq_ternary", fp, None),
                                                ("qat_ternary", qat, qat_report["selected_temperature"])):
                evidence.reserve(74624 + (64 << 10))
                pin, records = export(model, variant, args.output / arm / "models" / variant,
                                      data["calibration"], parity_inputs, x, temperature, VERSIONS[representation])
                evidence.reconcile()
                exports[arm][variant] = pin
                parity.extend(records)
            evidence.save_jsonl(args.output / arm / "go-parity.jsonl", parity, root)
            evidence.save(args.output / arm / "training-report.json", reports[arm], root)
            evidence.reconcile()
            del data, x, fp, qat, model
            gc.collect()
            torch.mps.empty_cache()
            print(json.dumps({"completed_arm": arm, "actual_optimizer_updates": updates}), flush=True)
        core.require(updates == evidence.actual_updates == 6400, "all preregistered optimizer updates")
        evidence.save(args.output / "report.json", {
            "schema": "gooo/full-input-judgment-training-result/v1", "status": "TRAINED_AND_EXPORTED",
            "optimizer_updates": updates, "preexecution_sha256": file_pin(args.output / "preexecution.json")["sha256"],
            "training": reports, "exports": exports, "default_model_promoted": False,
            "scope": "Twelve fresh exports; Go parity, behavior, resource and native observations remain required."}, root)
    except BaseException as error:
        evidence.reconcile()
        evidence.save(args.output / "failure.json", {
            "status": "FAILED_PREFIX_RETAINED", "error_type": type(error).__name__,
            "actual_optimizer_updates": evidence.actual_updates, "completed_stage_optimizer_updates": updates,
            "scope": "Per-update and epoch journals retained. No automatic retry."}, root)
        raise
    finally:
        core.Model = original_model
        evidence.reconcile()


if __name__ == "__main__":
    main()
