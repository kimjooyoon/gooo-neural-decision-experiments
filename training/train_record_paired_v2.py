"""Offline MPS fitting/export for the preregistered paired intent pilot.

Go supplies all source-bound inputs, finite labels, inference and execution.
The preceding trainer and feature contracts remain independently reproducible.
"""
import argparse
import json
import resource
import subprocess
import time
from pathlib import Path

import numpy as np
import torch
from torch import nn
import train_pilot_v2 as core

VERSION = "triple_record_field_context_v1_joint_v1"
COUNTS = {"train": 3072, "calibration": 1536, "test_source": 1536,
          "test_wording_seen": 512, "test_wording_new": 512}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--prepared", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--source-revision", required=True)
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[1]
    head = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=root, text=True).strip()
    core.require(head == args.source_revision and not subprocess.check_output(
        ["git", "status", "--porcelain"], cwd=root).strip(), "clean published trainer required")
    core.require(torch.backends.mps.is_available() and not args.output.exists(), "fresh MPS output required")
    manifest_raw = (args.prepared / "manifest.json").read_bytes()
    manifest = json.loads(manifest_raw)
    core.require(manifest["schema"] == "gooo/record-field-paired-training-inputs/v2" and
                 manifest["complete"] and manifest["total"] == 7168 and
                 manifest["rows"] == COUNTS and manifest["feature_version"] == VERSION,
                 "complete fixed paired inventory required")
    core.require(set(manifest["files"]) == {"rows.jsonl", "features-fp32.bin", "initial-fp32.bin", "compiler-build.json"}, "four preparation pins")
    for name, pin in manifest["files"].items():
        raw = (args.prepared / name).read_bytes()
        core.require(len(raw) == pin["bytes"] and core.sha(raw) == pin["sha256"], "Go preparation differs")
    rows = [json.loads(line) for line in (args.prepared / "rows.jsonl").read_text().splitlines()]
    core.require(len(rows) == 7168 and len({r["id"] for r in rows}) == 7168, "unique paired views")
    core.require({s: sum(r["split"] == s for r in rows) for s in COUNTS} == COUNTS, "split inventory")
    x = np.frombuffer((args.prepared / "features-fp32.bin").read_bytes(), dtype="<f4").copy().reshape(7168, 768)
    y = np.array([r["observed_complete_mask"] for r in rows], dtype=np.int64)
    core.require(np.isfinite(x).all() and ((y >= 0) & (y < 8)).all(), "finite Go features/masks")
    for split, families, styles in (("train", {0, 1, 2, 3}, {0}), ("calibration", {4, 5}, {0}),
                                   ("test_source", {6, 7}, {0}), ("test_wording_seen", {0, 3}, {1, 2}),
                                   ("test_wording_new", {6, 7}, {1, 2})):
        part = [r for r in rows if r["split"] == split]
        core.require({r["source_family"] for r in part} == families and
                     {r["wording_variant"] for r in part} == styles and
                     all(sum(r["intent_goal"] == g for r in part) == len(part)//8 for g in range(8)),
                     "preregistered balanced goals/source/wording axes")
    train = np.array([i for i, r in enumerate(rows) if r["split"] == "train"])
    cal = np.array([i for i, r in enumerate(rows) if r["split"] == "calibration"])
    core.FEATURES, core.HIDDEN, core.LIMIT, core.LABELS = 768, 24, 1600, [f"mask_{i}" for i in range(8)]
    blob = np.frombuffer((args.prepared / "initial-fp32.bin").read_bytes(), dtype="<f4").copy()
    initial, at = {}, 0
    for name, shape in (("first.weight", (24, 768)), ("first.bias", (24,)), ("last.weight", (8, 24)), ("last.bias", (8,))):
        count = int(np.prod(shape))
        initial[name] = torch.from_numpy(blob[at:at+count].reshape(shape).copy())
        at += count
    core.require(at == len(blob) == 18656, "fresh initializer extent")
    args.output.mkdir(parents=True)
    core.save_json(args.output / "preexecution.json", {
        "schema": "gooo/record-paired-training/v2", "source_revision": head,
        "prepared_manifest_sha256": core.sha(manifest_raw), "trainer_sha256": core.sha(Path(__file__).read_bytes()),
        "core_sha256": core.sha((root / "training/train_pilot_v2.py").read_bytes()), "device": "mps",
        "torch": torch.__version__, "epochs_per_stage": 40, "batch_rows": 64,
        "planned_optimizer_updates": 3840, "learning_rate": .003, "weight_decay": .01, "shuffle_seed": 20261053,
        "initial_weights_sha256": manifest["files"]["initial-fp32.bin"]["sha256"], "gpu_utilization_measured": False,
        "scope": "Opposing and mixed requirements over eight authored source families. Only train fits weights, calibration selects epoch/temperature; three diagnostic axes are unused until separate Go evaluation."})
    actual, reports, models = 0, {}, {}
    journal = (args.output / "updates.jsonl").open("x")
    try:
        for stage in ("fp32", "qat_ternary"):
            model = core.Model(qat=stage == "qat_ternary")
            model.load_state_dict(initial if stage == "fp32" else models["fp32"].state_dict())
            model.to("mps")
            optimizer = torch.optim.AdamW(model.parameters(), lr=.003, weight_decay=.01)
            train_x = torch.from_numpy(x[train]).to("mps")
            train_y = torch.from_numpy(y[train]).to("mps")
            cal_x = torch.from_numpy(x[cal]).to("mps")
            rng = np.random.default_rng(20261053)
            best, selected, allocated, driver = None, (float("inf"), 0, 0.), 0, 0
            started, before = time.perf_counter(), resource.getrusage(resource.RUSAGE_SELF)
            for epoch in range(1, 41):
                order = rng.permutation(len(train))
                for begin in range(0, len(train), 64):
                    ix = torch.from_numpy(order[begin:begin+64]).to("mps")
                    optimizer.zero_grad(set_to_none=True)
                    loss = nn.functional.cross_entropy(model(train_x[ix]), train_y[ix])
                    core.require(bool(torch.isfinite(loss).cpu()), "finite MPS objective")
                    loss.backward()
                    optimizer.step()
                    actual += 1
                    journal.write(json.dumps({"stage": stage, "epoch": epoch, "actual_updates": actual,
                                              "loss": float(loss.detach().cpu())}) + "\n")
                    journal.flush()
                with torch.no_grad():
                    logits = model(cal_x).cpu().numpy()
                candidate = min((core.scores(logits, y[cal], t)["nll"], epoch, t) for t in (.5, 1., 2., 4.))
                if candidate < selected:
                    selected = candidate
                    best = {k: v.detach().cpu().clone() for k, v in model.state_dict().items()}
                allocated = max(allocated, torch.mps.current_allocated_memory())
                driver = max(driver, torch.mps.driver_allocated_memory())
                core.save_json(args.output / stage / f"epoch-{epoch:02d}.json", {
                    "epoch": epoch, "calibration_nll": candidate[0], "temperature": candidate[2], "actual_updates": actual})
            torch.mps.synchronize()
            after = resource.getrusage(resource.RUSAGE_SELF)
            reports[stage] = {"updates": 1920, "wall_seconds": time.perf_counter()-started,
                              "cpu_seconds": after.ru_utime+after.ru_stime-before.ru_utime-before.ru_stime,
                              "selected_epoch": selected[1], "temperature": selected[2], "calibration_nll": selected[0],
                              "sampled_mps_tensor_peak_bytes": allocated, "sampled_mps_driver_peak_bytes": driver}
            model.cpu().load_state_dict(best)
            models[stage] = model
            print(json.dumps({"completed_stage": stage, "actual_updates": actual, **reports[stage]}), flush=True)
        core.require(actual == 3840, "fixed actual update budget")
        parity = []
        # Four KO/EN adjacent pairs per evaluation axis, goals0/1/6/7 when present.
        # Add train/calibration views: fixed balanced numerical sample, no quality tuning.
        indices = []
        for split in COUNTS:
            for goal in (0, 1, 6, 7):
                for lang in ("ko", "en"):
                    indices.append(next(i for i, r in enumerate(rows) if r["split"] == split and
                                        r["intent_goal"] == goal and r["language"] == lang))
        core.require(len(set(indices)) == 40, "fixed numerical sample")
        for variant in ("fp32", "ptq_ternary", "qat_ternary"):
            model = models["qat_ternary"] if variant == "qat_ternary" else models["fp32"]
            directory = args.output / "models" / variant
            core.export_model(model, variant, directory, 1., 1.)
            logits = core.exported_logits(directory, x[cal])
            temperature = min((core.scores(logits, y[cal], t)["nll"], t) for t in (.5, 1., 2., 4.))[1]
            metadata = json.loads((directory / "model.json").read_text())
            metadata.update(schema="gooo/tiny-three-choice-path-model/v1", feature_version=VERSION,
                            arithmetic_version="float32_separate_v1", temperature=temperature)
            del metadata["confidence_threshold"]
            core.save_json(directory / "model.json", metadata)
            values = core.exported_logits(directory, x[indices])
            for i, index in enumerate(indices):
                parity.append({"variant": variant, "row": index, "id": rows[index]["id"],
                               "text": rows[index]["text"], "logits": values[i].tolist()})
        with (args.output / "go-parity.jsonl").open("x") as f:
            for record in parity:
                f.write(json.dumps(record, ensure_ascii=False, allow_nan=False) + "\n")
        core.save_json(args.output / "report.json", {"schema": "gooo/record-paired-training-result/v2",
                     "status": "TRAINED_AND_EXPORTED", "actual_optimizer_updates": actual, "training": reports,
                     "default_model_promoted": False, "scope": "All variants retained. Go independently audits numerical, source/goal ordering and compiled current-input completeness."})
    except BaseException as error:
        core.save_json(args.output / "failure.json", {"actual_optimizer_updates": actual,
                       "error_type": type(error).__name__, "status": "FAILED_PREFIX_RETAINED"})
        raise
    finally:
        journal.close()


if __name__ == "__main__":
    main()
