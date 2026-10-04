"""Offline MPS optimization/export; Go owns source, arrays, inference and execution."""
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

VERSION = "triple_record_field_context_v1_shared_v1"
COUNTS = {"train": 3072, "calibration": 1536, "test_source": 1536,
          "test_wording_seen": 512, "test_wording_new": 512}

class Shared(nn.Module):
    def __init__(self, qat=False):
        super().__init__()
        self.first = nn.Linear(256, 8)
        self.last = nn.Linear(8, 2, bias=False)
        self.qat = qat

    def forward(self, x):
        w1 = core.ternary(self.first.weight) if self.qat else self.first.weight
        w2 = core.ternary(self.last.weight) if self.qat else self.last.weight
        h = torch.relu(nn.functional.linear(x.reshape(-1, 3, 256), w1, self.first.bias))
        local = nn.functional.linear(h, w2)
        return torch.stack([sum(local[:, part, (mask >> part) & 1] for part in range(3))
                            for mask in range(8)], dim=1)

def export(model, variant, directory):
    directory.mkdir(parents=True, exist_ok=False)
    blob, layout = bytearray(), []
    for name, tensor, rows, cols in (("w1", model.first.weight, 8, 256),
                                    ("b1", model.first.bias, 1, 8), ("w2", model.last.weight, 2, 8)):
        array = tensor.detach().cpu().numpy().astype(np.float32).reshape(-1)
        offset, scale = len(blob), 1.
        if variant != "fp32" and name.startswith("w"):
            scale = float(np.float32(max(float(np.abs(array).mean()), 1e-8)))
            codes = np.rint(array / np.float32(scale)).clip(-1, 1).astype(np.int8)
            for begin in range(0, len(codes), 5):
                digits = [int(c)+1 for c in codes[begin:begin+5]]
                digits += [1] * (5-len(digits))
                blob.append(sum(d * 3**i for i, d in enumerate(digits)))
            encoding = "ternary_base3_5"
        else:
            blob.extend(array.astype("<f4").tobytes())
            encoding = "float32_le"
        layout.append(dict(name=name, rows=rows, cols=cols, count=len(array), encoding=encoding,
                           offset=offset, bytes=len(blob)-offset, scale=scale))
    core.require(len(blob) == (8288 if variant == "fp32" else 446), "compact extent")
    (directory / "weights.bin").write_bytes(blob)
    core.save_json(directory / "model.json", dict(schema="gooo/tiny-shared-three-choice-path-model/v1",
                   feature_version=VERSION, arithmetic_version="float32_separate_v1", variant=variant,
                   feature_dim=768, hidden_dim=8, max_bytes=1600, labels=[f"mask_{i}" for i in range(8)],
                   temperature=1., weights_file="weights.bin", weights_sha256=core.sha(blob), tensors=layout))

def exact_logits(directory, x):
    metadata = json.loads((directory / "model.json").read_text())
    blob = (directory / "weights.bin").read_bytes()
    tensors, scales = {}, {}
    for t in metadata["tensors"]:
        raw = blob[t["offset"]:t["offset"]+t["bytes"]]
        if t["encoding"] == "float32_le":
            array = np.frombuffer(raw, dtype="<f4").copy()
        else:
            values = []
            for byte in raw:
                for _ in range(5):
                    values.append(byte % 3-1)
                    byte //= 3
            array = np.array(values[:t["count"]], dtype=np.float32)
        tensors[t["name"]] = array.reshape(t["rows"], t["cols"])
        scales[t["name"]] = np.float32(t["scale"])
    chunks = x.reshape(-1, 3, 256)
    hidden = np.empty((len(x), 3, 8), dtype=np.float32)
    for node in range(8):
        acc = np.zeros((len(x), 3), dtype=np.float32)
        for i in range(256):
            acc = np.float32(acc + np.float32(chunks[:, :, i] * tensors["w1"][node, i]))
        if metadata["variant"] != "fp32":
            acc = np.float32(acc * scales["w1"])
        hidden[:, :, node] = np.maximum(np.float32(acc + tensors["b1"][0, node]), 0)
    logits = np.empty((len(x), 8), dtype=np.float32)
    for mask in range(8):
        acc = np.zeros(len(x), dtype=np.float32)
        for part in range(3):
            for node in range(8):
                acc = np.float32(acc + np.float32(hidden[:, part, node] * tensors["w2"][(mask >> part) & 1, node]))
        if metadata["variant"] != "fp32":
            acc = np.float32(acc * scales["w2"])
        logits[:, mask] = np.float32(acc + np.float32(0))
    return logits

def main():
    parser = argparse.ArgumentParser()
    for name in ("prepared", "initializer", "output"):
        parser.add_argument("--"+name, type=Path, required=True)
    parser.add_argument("--source-revision", required=True)
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[1]
    head = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=root, text=True).strip()
    core.require(head == args.source_revision and not subprocess.check_output(["git", "status", "--porcelain"], cwd=root).strip(), "clean published shared trainer required")
    core.require(torch.backends.mps.is_available() and not args.output.exists(), "fresh MPS output")
    manifest_raw = (args.initializer / "manifest.json").read_bytes()
    manifest = json.loads(manifest_raw)
    core.require(manifest["schema"] == "gooo/record-shared-field-initializer/v1" and manifest["rows"] == 7168 and manifest["parameters"] == 2072 and manifest["initial_seed"] == 20261055 and manifest["shared_feature_version"] == VERSION, "registered Go initialization")
    for name, pin in manifest["prepared_pins"].items():
        raw = (args.prepared / name).read_bytes()
        core.require(core.sha(raw) == pin["sha256"] and len(raw) == pin["bytes"], "frozen Go bank pin")
    rows = [json.loads(line) for line in (args.prepared / "rows.jsonl").read_text().splitlines()]
    core.require(len(rows) == 7168 and {s: sum(r["split"] == s for r in rows) for s in COUNTS} == COUNTS, "fixed paired inventory")
    x = np.frombuffer((args.prepared / "features-fp32.bin").read_bytes(), dtype="<f4").copy().reshape(7168, 768)
    y = np.array([r["observed_complete_mask"] for r in rows], dtype=np.int64)
    raw = (args.initializer / "initial-fp32.bin").read_bytes()
    core.require(len(raw) == 8288 and core.sha(raw) == manifest["initial_sha256"], "fresh 2072 Go parameters")
    blob = np.frombuffer(raw, dtype="<f4").copy()
    initial = {"first.weight": torch.from_numpy(blob[:2048].reshape(8, 256)),
               "first.bias": torch.from_numpy(blob[2048:2056]), "last.weight": torch.from_numpy(blob[2056:].reshape(2, 8))}
    train = np.array([i for i, r in enumerate(rows) if r["split"] == "train"])
    cal = np.array([i for i, r in enumerate(rows) if r["split"] == "calibration"])
    args.output.mkdir(parents=True)
    core.save_json(args.output / "preexecution.json", dict(schema="gooo/record-shared-field-training/v1", source_revision=head,
        initializer_manifest_sha256=core.sha(manifest_raw), protocol_sha256=core.sha((root / "docs/record-shared-field-design-20261005.md").read_bytes()),
        trainer_sha256=core.sha(Path(__file__).read_bytes()), core_sha256=core.sha((root / "training/train_pilot_v2.py").read_bytes()),
        device="mps", torch=torch.__version__, epochs_per_stage=40, planned_optimizer_updates=3840,
        batch_rows=64, shuffle_seed=20261053, learning_rate=.003, weight_decay=.01, gpu_utilization_measured=False))
    actual, reports, models = 0, {}, {}
    journal = (args.output / "updates.jsonl").open("x")
    try:
        for stage in ("fp32", "qat_ternary"):
            model = Shared(qat=stage == "qat_ternary")
            model.load_state_dict(initial if stage == "fp32" else models["fp32"].state_dict())
            model.to("mps")
            optimizer = torch.optim.AdamW(model.parameters(), lr=.003, weight_decay=.01)
            tx, ty, cx = torch.from_numpy(x[train]).to("mps"), torch.from_numpy(y[train]).to("mps"), torch.from_numpy(x[cal]).to("mps")
            rng = np.random.default_rng(20261053)
            best, selected, allocated, driver = None, (float("inf"), 0, 0.), 0, 0
            started, before = time.perf_counter(), resource.getrusage(resource.RUSAGE_SELF)
            for epoch in range(1, 41):
                order = rng.permutation(len(train))
                for begin in range(0, len(train), 64):
                    ix = torch.from_numpy(order[begin:begin+64]).to("mps")
                    optimizer.zero_grad(set_to_none=True)
                    loss = nn.functional.cross_entropy(model(tx[ix]), ty[ix])
                    core.require(bool(torch.isfinite(loss).cpu()), "finite shared objective")
                    loss.backward(); optimizer.step(); actual += 1
                    journal.write(json.dumps(dict(stage=stage, epoch=epoch, actual_updates=actual, loss=float(loss.detach().cpu())))+"\n"); journal.flush()
                with torch.no_grad(): logits = model(cx).cpu().numpy()
                candidate = min((core.scores(logits, y[cal], t)["nll"], epoch, t) for t in (.5, 1., 2., 4.))
                if candidate < selected:
                    selected = candidate
                    best = {k: v.detach().cpu().clone() for k, v in model.state_dict().items()}
                allocated = max(allocated, torch.mps.current_allocated_memory())
                driver = max(driver, torch.mps.driver_allocated_memory())
                core.save_json(args.output / stage / f"epoch-{epoch:02d}.json", dict(epoch=epoch, calibration_nll=candidate[0], temperature=candidate[2], actual_updates=actual))
            torch.mps.synchronize()
            after = resource.getrusage(resource.RUSAGE_SELF)
            reports[stage] = dict(updates=1920, wall_seconds=time.perf_counter()-started,
                cpu_seconds=after.ru_utime+after.ru_stime-before.ru_utime-before.ru_stime, selected_epoch=selected[1], temperature=selected[2], calibration_nll=selected[0],
                sampled_mps_tensor_peak_bytes=allocated, sampled_mps_driver_peak_bytes=driver)
            model.cpu().load_state_dict(best); models[stage] = model
            print(json.dumps(dict(completed_stage=stage, actual_updates=actual, **reports[stage])), flush=True)
        core.require(actual == 3840, "fixed shared fitting budget")
        indices = [next(i for i, r in enumerate(rows) if r["split"] == split and r["intent_goal"] == goal and r["language"] == lang)
                   for split in COUNTS for goal in (0, 1, 6, 7) for lang in ("ko", "en")]
        parity = []
        for variant in ("fp32", "ptq_ternary", "qat_ternary"):
            model = models["qat_ternary"] if variant == "qat_ternary" else models["fp32"]
            directory = args.output / "models" / variant
            export(model, variant, directory)
            logits = exact_logits(directory, x[cal])
            temperature = min((core.scores(logits, y[cal], t)["nll"], t) for t in (.5, 1., 2., 4.))[1]
            meta = json.loads((directory / "model.json").read_text()); meta["temperature"] = temperature
            core.save_json(directory / "model.json", meta)
            values = exact_logits(directory, x[indices])
            parity.extend(dict(variant=variant, id=rows[ix]["id"], text=rows[ix]["text"], logits=values[j].tolist()) for j, ix in enumerate(indices))
        with (args.output / "go-parity.jsonl").open("x") as out:
            for ref in parity: out.write(json.dumps(ref, ensure_ascii=False, allow_nan=False)+"\n")
        core.save_json(args.output / "report.json", dict(schema="gooo/record-shared-field-training-result/v1", status="TRAINED_AND_EXPORTED", actual_optimizer_updates=actual, training=reports, default_model_promoted=False))
    except BaseException as error:
        core.save_json(args.output / "failure.json", dict(actual_optimizer_updates=actual, error_type=type(error).__name__, status="FAILED_PREFIX_RETAINED")); raise
    finally: journal.close()

if __name__ == "__main__": main()
