#!/usr/bin/env python3
"""Offline own MPS optimization/export; Go owns sources, targets and runtime."""
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
import semantic_features_v3 as v3
from train_typed_path_v1 import PATH_LABELS

DATASET = "2d1c9888a648d590e77253666b6e5ec848d375e47a3c99daf29dbde8bcbc7383"
MANIFEST = "f8feadcd45b64ee9f898c63914e1b7bf3b3fb2b537d9e35d1bd868abe89ec1e3"
DOCUMENTS = {
    "docs/joint-path-composition-preregistration-20261002.md": "03655ff443402b3e7d1b59fa780d4c8e68b750e1093b504a61de3fbf691d6a28",
    "docs/joint-path-composition-prefixture-amendment-20261002.md": "4f8b87715e3e23b6077f0dc80b2120d519b97135dff5c2c2ede1cd70583f33d1",
}


def load_pairs(directory):
    raw = (directory / "dataset.jsonl").read_bytes()
    manifest = (directory / "manifest.json").read_bytes()
    audit = json.loads((directory / "audit.json").read_text(), object_pairs_hook=core.unique_pairs)
    core.require(core.sha(raw) == DATASET and core.sha(manifest) == MANIFEST and
                 audit["status"] == "PASS" and audit["manifest_sha256"] == MANIFEST and
                 audit["decision_rows_audited"] == 4608 and audit["native_captures_audited"] == 2304,
                 "frozen native and independent oracle audit required")
    rows = [json.loads(line, object_pairs_hook=core.unique_pairs) for line in raw.splitlines()]
    core.require(len(rows) == len({r["id"] for r in rows}) == 4608, "distinct fixed decision rows required")
    grouped = {}
    for row in rows:
        c, lang, coordinate = row["configuration"], row["language"], row["coordinate"]
        core.require(0 <= c < 48 and lang in ("en", "ko") and coordinate in (0, 1), "fixed row bounds")
        core.require(row["split"] == ("train" if c < 32 else "calibration" if c < 40 else "development"), "group split")
        core.require(row["feature_version"] == v3.VERSION and
                     core.sha(row["input"]["text"].encode()) == row["input"]["input_sha256"].removeprefix("sha256:"), "full native input hash")
        key = lang, coordinate
        group = grouped.setdefault(row["program_contract_group"], {})
        core.require(key not in group, "duplicate function coordinate")
        group[key] = row
    pairs = []
    for group in grouped.values():
        core.require(set(group) == {(lang, coordinate) for lang in ("en", "ko") for coordinate in (0, 1)}, "two complete bilingual coordinates")
        pair = [[group[lang, i] for i in (0, 1)] for lang in ("en", "ko")]
        first = pair[0][0]
        for language in pair:
            texts = [r["input"]["text"] for r in language]
            combined = "gooo;joint2|" + "".join(str(len(t.encode())) + ":" + t for t in texts)
            for i, row in enumerate(language):
                core.require(row["joint_input"] == combined and row["joint_input_sha256"] == core.sha(combined.encode()), "complete ordered joint bytes")
                for field in ("program_contract_group", "family", "configuration", "desired_mask", "split", "source_sha256", "full_contract_target"):
                    core.require(row[field] == first[field], "joint target/source drift")
                core.require(row["finite_soft_targets"] == first["full_contract_target"]["coordinate_marginals"][i], "complete marginal target")
        pairs.append(pair)
    core.require(len(pairs) == 1152, "fixed bilingual function-pair count")
    return pairs


def configure(arm):
    core.FEATURES, core.HIDDEN = (256, 48) if arm == "independent" else (512, 24)
    core.LABELS = PATH_LABELS if arm == "independent" else [f"mask_{i}" for i in range(4)]
    core.LIMIT = 512 if arm == "independent" else 1088


def arrays(pairs, arm):
    x = np.asarray([[[v3.features(r["input"]["text"]) for r in language] for language in pair] for pair in pairs], dtype=np.float32)
    if arm == "joint":
        x = x.reshape(len(pairs), 2, 512) * np.float32(1 / math.sqrt(2))
        y = np.asarray([p[0][0]["full_contract_target"]["joint_mask_targets"] for p in pairs], dtype=np.float32)
        indices = np.broadcast_to(np.arange(4), y.shape).copy()
    else:
        y = np.asarray([[r["finite_soft_targets"] for r in p[0]] for p in pairs], dtype=np.float32)
        indices = np.asarray([[[PATH_LABELS.index(label) for label in r["eligible_labels"]] for r in p[0]] for p in pairs], dtype=np.int64)
    return x, indices, y


def loss(logits, indices, targets, arm):
    if arm == "independent":
        eligible = logits.gather(3, indices[:, None, :, :].expand(-1, 2, -1, -1))
    else:
        eligible = logits
    return -(targets[:, None] * torch.log_softmax(eligible, dim=-1)).sum(dim=-1).mean()


def fit(initial, data, arm, qat):
    torch.manual_seed(20261025)
    model = core.Model(qat=qat).to("mps")
    model.load_state_dict(initial)
    optimizer = torch.optim.AdamW(model.parameters(), lr=.001, weight_decay=.01)
    train = [torch.from_numpy(v).to("mps") for v in data["train"]]
    calibration = [torch.from_numpy(v).to("mps") for v in data["calibration"]]
    rng = np.random.default_rng(20261026)
    history, best, best_loss, allocated, driver, steps = [], None, float("inf"), 0, 0, 0
    started = time.perf_counter()
    def forward(dataset, ix=None):
        x, indices, target = dataset if ix is None else [v[ix] for v in dataset]
        shape = (-1, 2, 2, 8) if arm == "independent" else (-1, 2, 4)
        logits = model(x.reshape(-1, core.FEATURES)).reshape(shape)
        return loss(logits, indices, target, arm)
    for epoch in range(20):
        order = rng.permutation(len(train[0]))
        losses = []
        for begin in range(0, len(order), 128):
            ix = torch.from_numpy(order[begin:begin + 128]).to("mps")
            optimizer.zero_grad(set_to_none=True)
            value = forward(train, ix)
            value.backward()
            optimizer.step()
            losses.append(float(value.detach().cpu()))
            steps += 1
        with torch.no_grad():
            cal_nll = float(forward(calibration).cpu())
        if cal_nll < best_loss:
            best_loss, best = cal_nll, {k: v.detach().cpu().clone() for k, v in model.state_dict().items()}
        allocated = max(allocated, torch.mps.current_allocated_memory())
        driver = max(driver, torch.mps.driver_allocated_memory())
        history.append({"epoch": epoch + 1, "train_finite_nll": float(np.mean(losses)), "calibration_finite_nll": cal_nll})
    torch.mps.synchronize()
    duration = time.perf_counter() - started
    model.cpu().load_state_dict(best)
    core.require(steps == 120, "frozen function-batch optimizer budget")
    return model, {"epochs": 20, "optimizer_steps": steps, "training_bilingual_function_pairs": 768,
                   "training_function_views": 1536, "loop_wall_seconds": duration,
                   "mps_allocated_peak_sampled_bytes": allocated, "mps_driver_peak_sampled_bytes": driver,
                   "process_lifetime_peak_rss_bytes": resource.getrusage(resource.RUSAGE_SELF).ru_maxrss,
                   "selection": "calibration finite NLL; development excluded", "history": history}


def calibration_nll(directory, data, arm, temperature):
    x, indices, targets = data
    logits = core.exported_logits(directory, x.reshape(-1, core.FEATURES))
    if arm == "independent":
        logits = logits.reshape(-1, 2, 2, 8)
        eligible = np.take_along_axis(logits, np.broadcast_to(indices[:, None], (len(indices), 2, 2, 2)), axis=3)
    else:
        eligible = logits.reshape(-1, 2, 4)
    z = eligible.astype(np.float64) / temperature
    z -= z.max(axis=-1, keepdims=True)
    p = np.exp(z)
    p /= p.sum(axis=-1, keepdims=True)
    return float(-(targets[:, None] * np.log(p.clip(1e-12))).sum(axis=-1).mean())


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--source-revision", required=True)
    parser.add_argument("--curriculum", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[1]
    head = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=root, text=True).strip()
    core.require(head == args.source_revision and not subprocess.check_output(["git", "status", "--porcelain"], cwd=root).strip(), "clean exact trainer source")
    core.require(torch.backends.mps.is_available() and not args.output.exists(), "MPS and fresh output required")
    for name, pin in DOCUMENTS.items():
        core.require(core.sha((root / name).read_bytes()) == pin, "frozen protocol/amendment")
    pairs = load_pairs(args.curriculum)
    groups = {s: [p for p in pairs if p[0][0]["split"] == s] for s in ("train", "calibration", "development")}
    core.require({k: len(v) for k, v in groups.items()} == {"train": 768, "calibration": 192, "development": 192}, "function split counts")
    torch.set_num_threads(2)
    initial = {}
    for arm in ("independent", "joint"):
        configure(arm)
        torch.manual_seed(20261025)
        initial[arm] = {k: v.detach().clone() for k, v in core.Model().state_dict().items()}
    initial_pins = {arm: core.sha(b"".join(v.numpy().astype("<f4").tobytes() for v in state.values())) for arm, state in initial.items()}
    args.output.mkdir(parents=True)
    sources = {"training/" + f: core.sha((root / "training" / f).read_bytes()) for f in
               ("train_joint_composition_v1.py", "semantic_features_v3.py", "train_pilot_v2.py", "train_typed_path_v1.py")}
    core.save_json(args.output / "preexecution.json", {"schema": "gooo/joint-composition-training-preexecution/v1",
        "source_revision": head, "sources_sha256": sources, "dataset_sha256": DATASET, "curriculum_manifest_sha256": MANIFEST,
        "frozen_documents_sha256": DOCUMENTS, "initial_state_sha256": initial_pins, "initial_seed": 20261025,
        "shuffle_seed": 20261026, "device": "mps", "torch": torch.__version__, "numpy": np.__version__, "maximum_optimizer_steps": 480,
        "initialization": "architecture-specific own random states; no pretrained/Laya/earlier own weights; QAT starts its fresh FP checkpoint",
        "gpu_utilization_measured": False, "consistency_weight": 0.0})
    reports, exports = {}, {}
    for arm in ("independent", "joint"):
        configure(arm)
        data = {s: arrays(p, arm) for s, p in groups.items()}
        fp, fp_report = fit(copy.deepcopy(initial[arm]), data, arm, False)
        qat, qat_report = fit(copy.deepcopy(fp.state_dict()), data, arm, True)
        reports[arm], exports[arm], parity = {"fp32": fp_report, "qat_ternary": qat_report}, {}, []
        for variant, model in (("fp32", fp), ("ptq_ternary", fp), ("qat_ternary", qat)):
            directory = args.output / arm / "models" / variant
            core.export_model(model, variant, directory, 1, 1)
            temperature = min((.5, 1., 2., 4.), key=lambda t: calibration_nll(directory, data["calibration"], arm, t))
            meta = json.loads((directory / "model.json").read_text())
            meta.update(schema="gooo/tiny-path-decision-model/v1" if arm == "independent" else "gooo/tiny-joint-path-model/v1",
                        feature_version=v3.VERSION if arm == "independent" else "paired_semantic_context_v3_joint_v1", temperature=temperature)
            if arm == "joint":
                meta.pop("confidence_threshold")
            core.save_json(directory / "model.json", meta)
            exports[arm][variant] = {"metadata_sha256": core.sha((directory / "model.json").read_bytes()),
                                    "weights_sha256": core.sha((directory / "weights.bin").read_bytes()),
                                    "packed_weights_bytes": (directory / "weights.bin").stat().st_size}
            rows = [r for p in groups["development"] for language in p for r in (language if arm == "independent" else [language[0]])]
            indices = np.linspace(0, len(rows)-1, 32, dtype=int)
            texts = [rows[i]["input"]["text"] if arm == "independent" else rows[i]["joint_input"] for i in indices]
            if arm == "independent":
                features = np.stack([v3.features(t) for t in texts])
            else:
                feature_rows = []
                for i in indices:
                    r = rows[i]
                    pair = next(p for p in groups["development"] if p[0][0]["program_contract_group"] == r["program_contract_group"])
                    language = pair[0] if r["language"] == "en" else pair[1]
                    feature_rows.append(np.concatenate([v3.features(v["input"]["text"]) for v in language]) * np.float32(1/math.sqrt(2)))
                features = np.stack(feature_rows)
            logits = core.exported_logits(directory, features)
            probabilities = core.probabilities(logits, temperature)
            for j, i in enumerate(indices):
                parity.append({"variant": variant, "row_id": rows[i]["id"], "text": texts[j], "features": features[j].tolist(),
                               "logits": logits[j].tolist(), "probabilities": probabilities[j].tolist(),
                               "selected_label": core.LABELS[int(probabilities[j].argmax())]})
        core.save_json(args.output / arm / "go-parity.json", {"schema": "gooo/joint-composition-parity/v1", "rows": parity})
    steps = sum(r["optimizer_steps"] for a in reports.values() for r in a.values())
    core.require(steps == 480, "frozen total optimizer budget")
    core.save_json(args.output / "report.json", {"schema": "gooo/joint-composition-training-report/v1", "status": "TRAINED_AND_EXPORTED",
        "optimizer_steps": steps, "preexecution_sha256": core.sha((args.output / "preexecution.json").read_bytes()),
        "training": reports, "exports": exports, "default_model_promoted": False,
        "scope": "New six nonlinear finite compositions, 768 bilingual training function pairs per arm, 20+20 epochs, architecture-specific random initialization; six exports preserved; Go assembly and native dogfood remain separate evidence."})
    print(json.dumps({"status": "TRAINED_AND_EXPORTED", "optimizer_steps": steps, "model_exports": 6}))


if __name__ == "__main__":
    main()
