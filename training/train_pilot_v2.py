#!/usr/bin/env python3
"""Offline GPU training/export only. Deployed inference and bridges are Go."""

import argparse
import copy
import hashlib
import json
import math
import random
import resource
import struct
import sys
import time
from pathlib import Path

import numpy as np
import torch
from torch import nn

LABELS = ["add", "subtract", "multiply", "less_than", "less_equal", "equal", "and", "or"]
FEATURES, HIDDEN, LIMIT = 256, 48, 512
FROZEN_DATASET_SHA = "af0a637320a8256cd6ebcdb3486a6885f2766441c90f0cb154c11599f3c8ca3d"


def require(condition, message):
    if not condition:
        raise ValueError(message)


def unique_pairs(pairs):
    result = {}
    for key, value in pairs:
        require(key not in result, "duplicate JSON key")
        result[key] = value
    return result


def validate_frozen_dataset(path):
    root = Path(__file__).resolve().parents[1]
    raw = path.read_bytes()
    manifest_raw = path.with_name("manifest.json").read_bytes()
    manifest = json.loads(manifest_raw, object_pairs_hook=unique_pairs)
    require(sha(raw) == FROZEN_DATASET_SHA == manifest.get("dataset_sha256"), "dataset differs from frozen pilot")
    require(manifest.get("model_contract_sha256") == sha((root / "model-contract.json").read_bytes()), "model contract pin mismatch")
    require(manifest.get("generator_source_sha256") == sha((root / "cmd/dataset/main.go").read_bytes()), "generator source pin mismatch")
    lines = raw.decode("utf-8").splitlines(keepends=True)
    rows = [json.loads(line, object_pairs_hook=unique_pairs) for line in lines]
    fields = {"id", "template_id", "configuration_id", "language", "split", "text", "label"}
    require(all(isinstance(r, dict) and set(r) == fields for r in rows), "dataset row shape mismatch")
    require(all(isinstance(r["text"], str) and 1 <= len(r["text"].encode("utf-8")) <= LIMIT for r in rows), "input byte limit")
    require(all(r["label"] in LABELS and r["split"] in ("train", "calibration", "test") and r["language"] in ("en", "ko") for r in rows), "dataset enum mismatch")
    require(len({r["id"] for r in rows}) == len(rows) == 2048, "duplicate IDs or wrong row count")
    require(len({r["text"] for r in rows}) == len(rows), "duplicate texts")
    require(manifest.get("rows") == {"total": 2048, "train": 1536, "calibration": 256, "test": 256}, "manifest count mismatch")
    split_rows = {s: [r for r in rows if r["split"] == s] for s in ("train", "calibration", "test")}
    for split, expected in (("train", 1536), ("calibration", 256), ("test", 256)):
        require(len(split_rows[split]) == expected, "split count mismatch")
        split_raw = "".join(line for line, row in zip(lines, rows) if row["split"] == split).encode("utf-8")
        require(sha(split_raw) == manifest.get("split_sha256", {}).get(split), "split hash mismatch")
    for a, b in (("train", "calibration"), ("train", "test"), ("calibration", "test")):
        require(not {r["template_id"] for r in split_rows[a]} & {r["template_id"] for r in split_rows[b]}, "template split leakage")
    return raw, rows, {"dataset_manifest_sha256": sha(manifest_raw), "model_contract_sha256": manifest["model_contract_sha256"],
                       "generator_source_sha256": manifest["generator_source_sha256"]}


def sha(raw):
    return hashlib.sha256(raw).hexdigest()


def save_json(path, value):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, ensure_ascii=False, indent=2, sort_keys=True, allow_nan=False) + "\n")


def features(text):
    encoded = text.encode("utf-8")
    if not 1 <= len(encoded) <= LIMIT:
        raise ValueError("input must contain 1..512 UTF-8 bytes; inputs are not truncated")
    raw = bytes(c + 32 if 65 <= c <= 90 else c for c in encoded)
    counts = np.zeros(FEATURES, dtype=np.float32)
    for n in (2, 3):
        for start in range(max(0, len(raw) - n + 1)):
            value = 2166136261
            for c in raw[start:start + n]:
                value = ((value ^ c) * 16777619) & 0xffffffff
            counts[value % FEATURES] += 1
    norm = math.sqrt(sum(float(c) ** 2 for c in counts))
    if norm:
        counts *= np.float32(1 / norm)
    return counts


def ternary(weight):
    scale = weight.detach().abs().mean().clamp_min(1e-8)
    codes = torch.round(weight / scale).clamp(-1, 1)
    quantized = codes * scale
    return weight + (quantized - weight).detach()


class Model(nn.Module):
    def __init__(self, qat=False):
        super().__init__()
        self.first = nn.Linear(FEATURES, HIDDEN)
        self.last = nn.Linear(HIDDEN, len(LABELS))
        self.qat = qat

    def forward(self, x):
        w1 = ternary(self.first.weight) if self.qat else self.first.weight
        w2 = ternary(self.last.weight) if self.qat else self.last.weight
        x = torch.relu(nn.functional.linear(x, w1, self.first.bias))
        return nn.functional.linear(x, w2, self.last.bias)


def probabilities(logits, temperature):
    z = logits.astype(np.float64) / temperature
    z -= z.max(axis=1, keepdims=True)
    p = np.exp(z)
    return p / p.sum(axis=1, keepdims=True)


def scores(logits, labels, temperature):
    p = probabilities(logits, temperature)
    choices = p.argmax(axis=1)
    correct = choices == labels
    onehot = np.eye(len(LABELS))[labels]
    confidence = p.max(axis=1)
    ece, bins = 0.0, []
    for i in range(10):
        mask = (confidence >= i / 10) & (confidence < (i + 1) / 10 if i < 9 else confidence <= 1)
        n = int(mask.sum())
        accuracy = float(correct[mask].mean()) if n else None
        mean = float(confidence[mask].mean()) if n else None
        if n:
            ece += n / len(labels) * abs(accuracy - mean)
        bins.append({"lower": i / 10, "upper": (i + 1) / 10, "count": n,
                     "accuracy": accuracy, "mean_confidence": mean})
    return {"planned": len(labels), "observed": len(labels), "correct": int(correct.sum()),
            "accuracy": float(correct.mean()), "nll": float(-np.log(p[np.arange(len(labels)), labels].clip(1e-12)).mean()),
            "brier_multiclass_sum": float(((p - onehot) ** 2).sum(axis=1).mean()),
            "ece_equal_width_10": ece, "calibration_bins": bins}


def calibrate(logits, labels):
    grid = np.geomspace(0.15, 5.0, 80)
    temperature = min(grid, key=lambda t: scores(logits, labels, float(t))["nll"])
    p = probabilities(logits, float(temperature))
    correct, confidence = p.argmax(axis=1) == labels, p.max(axis=1)
    candidates = []
    for threshold in np.linspace(0.125, 0.99, 90):
        accepted = confidence >= threshold
        count = int(accepted.sum())
        if count >= 32 and float(correct[accepted].mean()) >= 0.95:
            candidates.append((count, -float(threshold), float(threshold)))
    threshold = max(candidates)[2] if candidates else 1.0
    return float(temperature), threshold


def export_model(model, variant, directory, temperature, threshold):
    directory.mkdir(parents=True, exist_ok=False)
    blob, layout = bytearray(), []
    for name, tensor, rows, cols in [
        ("w1", model.first.weight, HIDDEN, FEATURES),
        ("b1", model.first.bias, 1, HIDDEN),
        ("w2", model.last.weight, len(LABELS), HIDDEN),
        ("b2", model.last.bias, 1, len(LABELS)),
    ]:
        array = tensor.detach().cpu().numpy().astype(np.float32).reshape(-1)
        offset, scale = len(blob), 1.0
        if variant != "fp32" and name.startswith("w"):
            scale = max(float(np.abs(array).mean()), 1e-8)
            codes = np.rint(array / scale).clip(-1, 1).astype(np.int8)
            packed = bytearray()
            for start in range(0, len(codes), 5):
                group = [int(x) + 1 for x in codes[start:start + 5]]
                group += [1] * (5 - len(group))
                packed.append(sum(code * (3 ** index) for index, code in enumerate(group)))
            blob.extend(packed)
            encoding = "ternary_base3_5"
        else:
            blob.extend(array.astype("<f4").tobytes())
            encoding = "float32_le"
        layout.append({"name": name, "rows": rows, "cols": cols, "count": len(array),
                       "encoding": encoding, "offset": offset, "bytes": len(blob) - offset, "scale": scale})
    (directory / "weights.bin").write_bytes(blob)
    config = {"schema": "gooo/tiny-ir-decision-model/v1", "variant": variant,
              "feature_dim": FEATURES, "hidden_dim": HIDDEN, "max_bytes": LIMIT,
              "labels": LABELS, "temperature": temperature, "confidence_threshold": threshold,
              "weights_file": "weights.bin", "weights_sha256": sha(blob), "tensors": layout}
    save_json(directory / "model.json", config)
    total_parameters = sum(x["count"] for x in layout)
    return {"weights_bytes": len(blob), "model_json_bytes": (directory / "model.json").stat().st_size,
            "total_parameters": total_parameters, "matrix_parameters": HIDDEN * FEATURES + len(LABELS) * HIDDEN,
            "weight_blob_bits_per_total_parameter": len(blob) * 8 / total_parameters,
            "note": "Packed artifact size is separate from resident execution memory; biases/scales/metadata are not 1.58-bit."}


def exported_logits(directory, x):
    config = json.loads((directory / "model.json").read_text())
    blob = (directory / "weights.bin").read_bytes()
    tensors = {}
    for t in config["tensors"]:
        raw = blob[t["offset"]:t["offset"] + t["bytes"]]
        if t["encoding"] == "float32_le":
            values = np.frombuffer(raw, dtype="<f4").copy()
        else:
            decoded = []
            for code in raw:
                for _ in range(5):
                    decoded.append(code % 3 - 1)
                    code //= 3
            values = np.array(decoded[:t["count"]], dtype=np.float32) * np.float32(t["scale"])
        tensors[t["name"]] = values.reshape(t["rows"], t["cols"])
    h = np.maximum(x @ tensors["w1"].T + tensors["b1"], 0)
    return h @ tensors["w2"].T + tensors["b2"]


def train(x, y, cal_x, cal_y, seed, epochs, batch, device, qat):
    random.seed(seed)
    np.random.seed(seed)
    torch.manual_seed(seed)
    model = Model(qat=qat).to(device)
    optimizer = torch.optim.AdamW(model.parameters(), lr=0.003, weight_decay=0.01)
    tx, ty = torch.from_numpy(x).to(device), torch.from_numpy(y).to(device)
    cx = torch.from_numpy(cal_x).to(device)
    rng = np.random.default_rng(seed)
    history, best, best_loss = [], None, float("inf")
    start = time.perf_counter()
    max_allocated, max_driver = 0, 0
    for epoch in range(epochs):
        order = rng.permutation(len(y))
        losses = []
        model.train()
        for begin in range(0, len(order), batch):
            index = torch.from_numpy(order[begin:begin + batch]).to(device)
            optimizer.zero_grad(set_to_none=True)
            logits = model(tx[index])
            loss = nn.functional.cross_entropy(logits, ty[index])
            loss.backward()
            optimizer.step()
            losses.append(float(loss.detach().cpu()))
        model.eval()
        with torch.no_grad():
            cal = model(cx).cpu().numpy()
        value = scores(cal, cal_y, 1.0)["nll"]
        if value < best_loss:
            best_loss = value
            best = {key: tensor.detach().cpu().clone() for key, tensor in model.state_dict().items()}
        if device == "mps":
            max_allocated = max(max_allocated, torch.mps.current_allocated_memory())
            max_driver = max(max_driver, torch.mps.driver_allocated_memory())
        history.append({"epoch": epoch + 1, "training_loss": sum(losses) / len(losses), "calibration_nll": value})
    if device == "mps":
        torch.mps.synchronize()
    duration = time.perf_counter() - start
    model.cpu()
    model.load_state_dict(best)
    return model, {"seed": seed, "epochs": epochs, "batch_size": batch, "steps": epochs * math.ceil(len(y) / batch),
                   "device": device, "training_loop_wall_seconds": duration, "mps_allocated_peak_sampled_bytes": max_allocated,
                   "mps_driver_peak_sampled_bytes": max_driver, "process_peak_rss_bytes": resource.getrusage(resource.RUSAGE_SELF).ru_maxrss,
                   "selection": "lowest calibration NLL; no test access during training", "history": history,
                   "timing_note": "Whole loop includes training, calibration forward, CPU transfers and scoring; not pure GPU kernel time",
                   "memory_note": "MPS end-of-epoch samples; process RSS is cumulative across variants, not isolated variant peak"}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--dataset", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--epochs", type=int, default=120)
    parser.add_argument("--batch", type=int, default=256)
    parser.add_argument("--seed", type=int, default=20360930)
    parser.add_argument("--validate-only", action="store_true")
    args = parser.parse_args()
    data_raw, rows, dataset_pins = validate_frozen_dataset(args.dataset)
    require(1 <= args.epochs <= 200 and 1 <= args.batch <= 512, "pilot budget outside bounds")
    if args.validate_only:
        print(json.dumps({"status": "FROZEN_DATASET_VERIFIED", **dataset_pins, "optimization_level": sys.flags.optimize,
                          "model_training_calls": 0}))
        return
    require(torch.backends.mps.is_available(), "This pilot explicitly requires available MPS GPU")
    args.output.mkdir(parents=True, exist_ok=False)
    source = Path(__file__).read_bytes()
    (args.output / "training-source.py").write_bytes(source)
    require(all(row["label"] in LABELS and row["split"] in ("train", "calibration", "test") for row in rows), "invalid split/label")
    split_rows = {split: [r for r in rows if r["split"] == split] for split in ("train", "calibration", "test")}
    prompt_sets = {s: {r["text"] for r in rs} for s, rs in split_rows.items()}
    groups = {s: {r["template_id"] for r in rs} for s, rs in split_rows.items()}
    for a, b in (("train", "calibration"), ("train", "test"), ("calibration", "test")):
        require(not prompt_sets[a] & prompt_sets[b] and not groups[a] & groups[b], "split leakage")
    xs = {s: np.stack([features(r["text"]) for r in rs]) for s, rs in split_rows.items()}
    ys = {s: np.array([LABELS.index(r["label"]) for r in rs], dtype=np.int64) for s, rs in split_rows.items()}
    pre = {"schema": "gooo/tiny-ir-decision-training-preexecution/v1", "training_source_sha256": sha(source),
           **dataset_pins, "python_optimization_level": sys.flags.optimize,
           "dataset_sha256": sha(data_raw), "torch_version": torch.__version__, "numpy_version": np.__version__,
           "seed": args.seed, "epochs": args.epochs, "batch_size": args.batch,
           "hardware": {"chip": "Apple M4", "gpu_cores": 10, "unified_memory_gib": 16},
           "split_counts": {s: len(rs) for s, rs in split_rows.items()}, "data_provenance": "Public synthetic Go-generated templates only",
           "gpu_determinism": "seeded; bit-identical cross-device training is not asserted",
           "variants": ["fp32", "ptq_ternary", "qat_ternary"], "test_used_for_training_or_calibration": False}
    save_json(args.output / "preexecution.json", pre)
    torch.set_num_threads(2)
    models, training = {}, {}
    for variant, qat in (("fp32", False), ("qat_ternary", True)):
        models[variant], training[variant] = train(xs["train"], ys["train"], xs["calibration"], ys["calibration"],
                                                   args.seed, args.epochs, args.batch, "mps", qat)
    models["ptq_ternary"] = models["fp32"]
    results, parity = {}, []
    for variant in ("fp32", "ptq_ternary", "qat_ternary"):
        directory = args.output / "models" / variant
        export_model(models[variant], variant, directory, 1.0, 0.0)
        calibration_logits = exported_logits(directory, xs["calibration"])
        temperature, threshold = calibrate(calibration_logits, ys["calibration"])
        config = json.loads((directory / "model.json").read_text())
        config.update(temperature=temperature, confidence_threshold=threshold)
        save_json(directory / "model.json", config)
        variant_scores = {s: scores(exported_logits(directory, xs[s]), ys[s], temperature) for s in xs}
        tp = probabilities(exported_logits(directory, xs["test"]), temperature)
        accepted = tp.max(axis=1) >= threshold
        correct = tp.argmax(axis=1) == ys["test"]
        matrix_count = HIDDEN * FEATURES + len(LABELS) * HIDDEN
        all_count = matrix_count + HIDDEN + len(LABELS)
        results[variant] = {"scores": variant_scores, "temperature": temperature, "confidence_threshold": threshold,
                            "test_selective": {"planned": len(accepted), "accepted": int(accepted.sum()),
                                               "correct": int((correct & accepted).sum()),
                                               "accuracy": float(correct[accepted].mean()) if accepted.any() else None},
                            "weights_bytes": (directory / "weights.bin").stat().st_size,
                            "model_json_bytes": (directory / "model.json").stat().st_size,
                            "total_parameters": all_count, "matrix_parameters": matrix_count,
                            "weight_blob_bits_per_total_parameter": (directory / "weights.bin").stat().st_size * 8 / all_count,
                            "storage_note": "Matrix codes use 1.6 physical bits/weight; 1.585 is theoretical alphabet entropy. Biases, scales, metadata and runtime memory are separate."}
        logits = exported_logits(directory, xs["test"])
        for index in range(min(32, len(split_rows["test"]))):
            row = split_rows["test"][index]
            parity.append({"variant": variant, "text": row["text"], "features": xs["test"][index].tolist(),
                           "logits": logits[index].tolist(), "probabilities": tp[index].tolist(),
                           "selected_label": LABELS[int(tp[index].argmax())]})
    report = {"schema": "gooo/tiny-ir-decision-training-report/v1", "status": "TRAINED_AND_EXPORTED",
              "preexecution_sha256": sha((args.output / "preexecution.json").read_bytes()), "training": training,
              "variants": results, "limitations": ["Closed eight-operation synthetic task; not arbitrary natural language compilation",
                                                       "One seed and template-group test; no unseen operation family evaluation",
                                                       "QAT is ternary-weight MLP with float activations, not a reproduced BitNet Transformer",
                                                       "Python/exported scores pending independent Go inference parity and compiler-route replay",
                                                       "Sampled memory includes tooling and is not packed-artifact size"]}
    save_json(args.output / "report.json", report)
    save_json(args.output / "go-parity.json", {"schema": "gooo/tiny-ir-decision-parity/v1", "rows": parity})
    print(json.dumps({"status": report["status"], "training_loop_seconds": {k: v["training_loop_wall_seconds"] for k, v in training.items()},
                      "test_accuracy": {k: v["scores"]["test"]["accuracy"] for k, v in results.items()}}))


if __name__ == "__main__":
    main()
