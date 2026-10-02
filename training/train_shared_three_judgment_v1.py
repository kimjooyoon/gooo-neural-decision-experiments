"""Offline MPS-only shared source/intent judge; existing Go tensor ABI export."""
import argparse
import copy
import json
from pathlib import Path
import subprocess
from types import SimpleNamespace

import torch
from torch import nn
import train_own_three_feedback_v1 as base

core = base.core
Dense = core.Model
PROTOCOL = "docs/shared-three-judgment-preregistration-20261002.md"
ARMS = ("dense", "shared-local")


class SharedLocal(nn.Module):
    """Three tied diagonal blocks; exactly the existing expanded Go computation."""
    def __init__(self, qat=False):
        super().__init__()
        self.local_first = nn.Linear(256, 8)
        self.local_last = nn.Linear(8, 2, bias=False)
        self.qat = qat

    @property
    def first(self):
        w, b = self.local_first.weight, self.local_first.bias
        return SimpleNamespace(weight=torch.block_diag(w, w, w), bias=b.repeat(3))

    @property
    def last(self):
        w = self.local_last.weight
        rows = [torch.cat([w[(mask >> part) & 1] for part in range(3)]) for mask in range(8)]
        return SimpleNamespace(weight=torch.stack(rows), bias=w.new_zeros(8))

    def forward(self, x):
        first, last = self.first, self.last
        w1 = core.ternary(first.weight) if self.qat else first.weight
        w2 = core.ternary(last.weight) if self.qat else last.weight
        return nn.functional.linear(torch.relu(nn.functional.linear(x, w1, first.bias)), w2, last.bias)

    def state_dict(self, *args, **kwargs):
        first, last = self.first, self.last
        return {"first.weight": first.weight.detach(), "first.bias": first.bias.detach(),
                "last.weight": last.weight.detach(), "last.bias": last.bias.detach()}

    def load_state_dict(self, state, *args, **kwargs):
        with torch.no_grad():
            self.local_first.weight.copy_(state["first.weight"][:8, :256])
            self.local_first.bias.copy_(state["first.bias"][:8])
            self.local_last.weight.copy_(torch.stack([state["last.weight"][0, :8], state["last.weight"][1, :8]]))
        rebuilt = self.state_dict()
        core.require(set(state) == set(rebuilt) and all(torch.equal(state[k].cpu(), v.cpu()) for k, v in rebuilt.items()),
                     "shared checkpoint must preserve exact expanded topology")


def shared_initial(dense):
    model = SharedLocal()
    with torch.no_grad():
        model.local_first.weight.copy_(dense["first.weight"][:8, :256])
        model.local_first.bias.copy_(dense["first.bias"][:8])
        model.local_last.weight.copy_(dense["last.weight"][:2, :8])
    core.require(sum(p.numel() for p in model.parameters()) == 2072, "fixed shared parameter count")
    return {k: v.clone() for k, v in model.state_dict().items()}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--prepared", type=Path, required=True)
    parser.add_argument("--preparation-audit", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--source-revision", required=True)
    parser.add_argument("--protocol-sha256", required=True)
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[1]
    head = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=root, text=True).strip()
    core.require(head == args.source_revision and not subprocess.check_output(["git", "status", "--porcelain"], cwd=root).strip(),
                 "clean exact published optimizer source")
    protocol_sha = core.sha((root / PROTOCOL).read_bytes())
    core.require(protocol_sha == args.protocol_sha256, "published protocol hash")
    core.require(torch.backends.mps.is_available() and not args.output.exists(), "fresh local MPS output")
    core.require(args.output.resolve().parent == root / "runs" and args.output.name.startswith("own-three-"), "bounded study phase")
    core.FEATURES, core.HIDDEN, core.LIMIT, core.LABELS = 768, 24, 1600, [f"mask_{i}" for i in range(8)]
    manifest, rows, x, parity_inputs = base.load_inputs(args.prepared, args.preparation_audit)
    initial = base.state_from_blob(args.prepared / "initial-fp32.bin")
    core.require(base.state_sha(initial) == manifest["files"]["initial-fp32.bin"]["sha256"], "fresh Go initializer")
    shared = shared_initial(initial)
    data = {split: base.arrays(rows, x, split, "set-feedback") for split in ("train", "calibration")}
    sources = {name: core.sha((root / "training" / name).read_bytes()) for name in
               ("train_shared_three_judgment_v1.py", "train_own_three_feedback_v1.py", "train_pilot_v2.py")}
    prior_bytes = base.retained_bytes(root)
    base.save(args.output / "preexecution.json", {
        "schema": "gooo/shared-three-judgment-training/v1", "source_revision": head,
        "protocol_sha256": protocol_sha, "sources_sha256": sources, "arms": ARMS,
        "prepared_manifest_sha256": core.sha((args.prepared / "manifest.json").read_bytes()),
        "preparation_audit_sha256": core.sha(args.preparation_audit.read_bytes()),
        "initial_state_sha256": {"dense": base.state_sha(initial), "shared-local": base.state_sha(shared)},
        "trainable_parameters": {"dense": 18656, "shared-local": 2072},
        "expanded_parameters": 18656, "planned_optimizer_steps": 3200, "device": "mps",
        "torch": torch.__version__, "prior_raw_bytes": prior_bytes, "new_raw_cap_bytes": 64 << 20,
        "gpu_utilization_measured": False, "default_model_promoted": False,
        "scope": "Existing frozen source/teacher feature bank; no new intentions or untouched holdout; own fresh Go initialization."}, root)
    reports, exports, updates = {}, {}, 0
    try:
        for arm in ARMS:
            core.Model = Dense if arm == "dense" else SharedLocal
            seed = initial if arm == "dense" else shared
            fp, fp_report = base.fit(copy.deepcopy(seed), data, arm, False, args.output / arm, root)
            updates += fp_report["optimizer_steps"]
            qat, qat_report = base.fit(copy.deepcopy(fp.state_dict()), data, arm, True, args.output / arm, root)
            updates += qat_report["optimizer_steps"]
            reports[arm], exports[arm], parity = {"fp32": fp_report, "qat_ternary": qat_report}, {}, []
            for variant, model, temperature in (("fp32", fp, fp_report["selected_temperature"]),
                                                ("ptq_ternary", fp, None),
                                                ("qat_ternary", qat, qat_report["selected_temperature"])):
                pin, records = base.export(model, variant, args.output / arm / "models" / variant,
                                           data["calibration"], parity_inputs, x, temperature)
                exports[arm][variant] = pin
                parity.extend(records)
            base.save_jsonl(args.output / arm / "go-parity.jsonl", parity, root)
            base.save(args.output / arm / "training-report.json", reports[arm], root)
            core.require(base.retained_bytes(root)-prior_bytes <= 64 << 20, "new evidence cap; preserve prefix")
            print(json.dumps({"completed_arm": arm, "actual_optimizer_updates": updates}), flush=True)
        core.require(updates == 3200, "all fixed optimizer steps")
        base.save(args.output / "report.json", {"schema": "gooo/shared-three-judgment-training-result/v1",
                  "status": "TRAINED_AND_EXPORTED", "optimizer_steps": updates,
                  "preexecution_sha256": core.sha((args.output / "preexecution.json").read_bytes()),
                  "training": reports, "exports": exports, "default_model_promoted": False,
                  "scope": "Six fresh FP32/PTQ/QAT exports; Go numerical/topology/behavior/native audits remain required."}, root)
    except BaseException as error:
        base.save(args.output / "failure.json", {"status": "FAILED_PREFIX_RETAINED", "error_type": type(error).__name__,
                  "completed_stage_optimizer_updates": updates, "scope": "Actual per-update/epoch journals retained; no restart."}, root)
        raise
    finally:
        core.Model = Dense


if __name__ == "__main__":
    main()
