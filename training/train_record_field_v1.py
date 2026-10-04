"""Offline MPS optimization only; Go prepares sources/features and executes them."""
import argparse
import copy
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


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--prepared", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--source-revision", required=True)
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[1]
    head = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=root, text=True).strip()
    core.require(head == args.source_revision and not subprocess.check_output(["git", "status", "--porcelain"], cwd=root).strip(), "clean published trainer source required")
    core.require(torch.backends.mps.is_available() and not args.output.exists(), "fresh MPS output required")
    args.output.mkdir(parents=True)
    manifest_raw = (args.prepared / "manifest.json").read_bytes()
    manifest = json.loads(manifest_raw)
    core.require(manifest["complete"] and manifest["total"] == 768 and manifest["rows"] == {"train": 384, "calibration": 192, "test": 192} and manifest["feature_version"] == VERSION, "complete fixed source curriculum")
    for name, pin in manifest["files"].items():
        raw = (args.prepared / name).read_bytes()
        core.require(len(raw) == pin["bytes"] and core.sha(raw) == pin["sha256"], "Go preparation file differs")
    rows = [json.loads(line) for line in (args.prepared / "rows.jsonl").read_text().splitlines()]
    core.require(len(rows) == 768 and len({r["id"] for r in rows}) == 768, "complete unique row inventory")
    x = np.frombuffer((args.prepared / "features-fp32.bin").read_bytes(), dtype="<f4").copy().reshape(768, 768)
    y = np.array([r["observed_complete_mask"] for r in rows], dtype=np.int64)
    core.require(np.isfinite(x).all() and ((y >= 0) & (y < 8)).all(), "finite Go features and observed masks")
    for split, families in (("train", {0, 1, 2, 3}), ("calibration", {4, 5}), ("test", {6, 7})):
        core.require({r["source_family"] for r in rows if r["split"] == split} == families, "whole source family split")
    train = np.array([i for i, r in enumerate(rows) if r["split"] == "train"])
    cal = np.array([i for i, r in enumerate(rows) if r["split"] == "calibration"])
    core.FEATURES, core.HIDDEN, core.LIMIT, core.LABELS = 768, 24, 1600, [f"mask_{i}" for i in range(8)]
    blob = np.frombuffer((args.prepared / "initial-fp32.bin").read_bytes(), dtype="<f4").copy()
    initial, at = {}, 0
    for name, shape in (("first.weight", (24, 768)), ("first.bias", (24,)), ("last.weight", (8, 24)), ("last.bias", (8,))):
        count = int(np.prod(shape)); initial[name] = torch.from_numpy(blob[at:at+count].reshape(shape).copy()); at += count
    core.require(at == len(blob) == 18656, "fresh Go initializer extent")
    pre = {"schema": "gooo/record-field-training/v1", "source_revision": head, "prepared_manifest_sha256": core.sha(manifest_raw),
           "trainer_sha256": core.sha(Path(__file__).read_bytes()), "core_sha256": core.sha((root / "training/train_pilot_v2.py").read_bytes()),
           "device": "mps", "torch": torch.__version__, "epochs_per_stage": 40, "batch_rows": 64,
           "planned_optimizer_updates": 480, "learning_rate": .003, "weight_decay": .01, "shuffle_seed": 20261052,
           "initial_weights_sha256": manifest["files"]["initial-fp32.bin"]["sha256"], "gpu_utilization_measured": False,
           "scope": "Three field roles in eight source/expression families, paired KO/EN. Train rows alone fit weights; calibration chooses epoch/temperature; test labels are unused until Go evaluation."}
    core.save_json(args.output / "preexecution.json", pre)
    actual = 0
    reports = {}
    models = {}
    journal = (args.output / "updates.jsonl").open("x")
    try:
        for stage in ("fp32", "qat_ternary"):
            model = core.Model(qat=stage == "qat_ternary")
            model.load_state_dict(initial if stage == "fp32" else models["fp32"].state_dict())
            model.to("mps")
            optimizer = torch.optim.AdamW(model.parameters(), lr=.003, weight_decay=.01)
            train_x = torch.from_numpy(x[train]).to("mps"); train_y = torch.from_numpy(y[train]).to("mps")
            cal_x = torch.from_numpy(x[cal]).to("mps")
            rng = np.random.default_rng(20261052)
            best, selected, allocated, driver = None, (float("inf"), 0, 0.), 0, 0
            started, before = time.perf_counter(), resource.getrusage(resource.RUSAGE_SELF)
            for epoch in range(1, 41):
                order = rng.permutation(384)
                for begin in range(0, 384, 64):
                    ix = torch.from_numpy(order[begin:begin+64]).to("mps")
                    optimizer.zero_grad(set_to_none=True)
                    loss = nn.functional.cross_entropy(model(train_x[ix]), train_y[ix])
                    core.require(bool(torch.isfinite(loss).cpu()), "finite MPS objective")
                    loss.backward(); optimizer.step(); actual += 1
                    journal.write(json.dumps({"stage": stage, "epoch": epoch, "actual_updates": actual, "loss": float(loss.detach().cpu())}) + "\n"); journal.flush()
                with torch.no_grad():
                    logits = model(cal_x).cpu().numpy()
                candidate = min((core.scores(logits, y[cal], t)["nll"], epoch, t) for t in (.5, 1., 2., 4.))
                if candidate < selected:
                    selected = candidate; best = {k: v.detach().cpu().clone() for k, v in model.state_dict().items()}
                allocated = max(allocated, torch.mps.current_allocated_memory()); driver = max(driver, torch.mps.driver_allocated_memory())
                core.save_json(args.output / stage / f"epoch-{epoch:02d}.json", {"epoch": epoch, "calibration_nll": candidate[0], "temperature": candidate[2], "actual_updates": actual})
            torch.mps.synchronize()
            after = resource.getrusage(resource.RUSAGE_SELF)
            cpu = after.ru_utime + after.ru_stime - before.ru_utime - before.ru_stime
            reports[stage] = {"updates": 240, "wall_seconds": time.perf_counter()-started, "cpu_seconds": cpu,
                              "selected_epoch": selected[1], "temperature": selected[2], "calibration_nll": selected[0],
                              "sampled_mps_tensor_peak_bytes": allocated, "sampled_mps_driver_peak_bytes": driver}
            model.cpu().load_state_dict(best); models[stage] = model
            print(json.dumps({"completed_stage": stage, "actual_updates": actual, **reports[stage]}), flush=True)
        core.require(actual == 480, "fixed actual update budget")
        parity = []
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
            indices = [index for start in range(0, 768, 24) for index in (start, start+1)]
            values = core.exported_logits(directory, x[indices])
            for i, index in enumerate(indices):
                parity.append({"variant": variant, "row": index, "id": rows[index]["id"], "text": rows[index]["text"], "logits": values[i].tolist()})
        with (args.output / "go-parity.jsonl").open("x") as f:
            for record in parity: f.write(json.dumps(record, ensure_ascii=False, allow_nan=False) + "\n")
        core.save_json(args.output / "report.json", {"schema": "gooo/record-field-training-result/v1", "status": "TRAINED_AND_EXPORTED", "actual_optimizer_updates": actual, "training": reports,
                     "default_model_promoted": False, "scope": "Three new FP32/PTQ/QAT exports. Numerical, compiled finite and runtime costs are evaluated in Go separately."})
    except BaseException as error:
        core.save_json(args.output / "failure.json", {"actual_optimizer_updates": actual, "error_type": type(error).__name__, "status": "FAILED_PREFIX_RETAINED"})
        raise
    finally:
        journal.close()


if __name__ == "__main__":
    main()
