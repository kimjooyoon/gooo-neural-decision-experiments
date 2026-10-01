#!/usr/bin/env python3
"""Preregistered fixed-size v1/v2 channel comparison, offline MPS only."""
import argparse
import copy
import json
import subprocess
from pathlib import Path

import numpy as np
import torch
import train_pilot_v2 as core
import train_bilingual_judgment_v1 as paired
import train_feedback_path_v1 as feedback
from train_typed_path_v1 import PATH_LABELS, positioned_features
from split_features_v2 import VERSION, features as split_features

core.LABELS = PATH_LABELS


def arrays(pairs, rows, encoder):
    return (np.asarray([[encoder(rows[i]["text"]) for i in p["row_ids"]] for p in pairs], dtype=np.float32),
            np.asarray([[PATH_LABELS.index(v) for v in p["eligible_labels"]] for p in pairs], dtype=np.int64),
            np.asarray([p["finite_soft_targets"] for p in pairs], dtype=np.float32))


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--source-revision", required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[1]
    head = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=root, text=True).strip()
    core.require(head == args.source_revision and not subprocess.check_output(["git", "status", "--porcelain"], cwd=root).strip(), "clean committed source required")
    core.require(torch.backends.mps.is_available() and not args.output.exists(), "MPS and fresh output required")
    raw = (root / "data/feedback-path-v1/dataset.jsonl").read_bytes()
    pair_raw = (root / "data/bilingual-gooo-pairs-v1/pairs.jsonl").read_bytes()
    core.require(core.sha(raw) == "570d1d73bfea32662b869e5cfbf77e6d04df838b703191b076be58c1d2180dbf"
                 and core.sha(pair_raw) == "25e090b1c4d842b34cf7a1d3a994b7b6e4be70cb59389fe33420e5d8750574ce", "fixed existing data pins")
    rows = {r["id"]: r for r in (json.loads(v, object_pairs_hook=core.unique_pairs) for v in raw.splitlines())}
    pairs = [json.loads(v, object_pairs_hook=core.unique_pairs) for v in pair_raw.splitlines()]
    grouped = {s: [p for p in pairs if p["split"] == s] for s in ("train", "calibration", "test")}
    core.require(len(rows) == 6240 and {s: len(v) for s, v in grouped.items()} == {"train":800,"calibration":80,"test":160}, "fixed denominators")
    for p in pairs:
        a, b = [rows[i] for i in p["row_ids"]]
        core.require(a["split"] == b["split"] == p["split"] and [a["language"], b["language"]] == ["en","ko"]
                     and a["intention_label"] == b["intention_label"] and a["finite_cases"] == b["finite_cases"]
                     and a["eligible_labels"] == b["eligible_labels"] == p["eligible_labels"]
                     and a["best_finite_labels"] == b["best_finite_labels"]
                     and [core.sha(r["text"].encode()) for r in (a,b)] == p["input_sha256"], "source pair binding")
        core.require([float(label in a["best_finite_labels"])/len(a["best_finite_labels"]) for label in p["eligible_labels"]]
                     == p["finite_soft_targets"], "finite soft target binding")
    for field in ("program_id", "template_pair_id"):
        for a,b in (("train","calibration"),("train","test"),("calibration","test")):
            core.require(not {p[field] for p in grouped[a]} & {p[field] for p in grouped[b]}, "group leakage")
    sources = {name:core.sha((root/name).read_bytes()) for name in (
        "training/train_split_context_v2.py","training/split_features_v2.py","training/train_bilingual_judgment_v1.py",
        "training/train_feedback_path_v1.py","training/train_typed_path_v1.py","training/train_pilot_v2.py",
        "internal/decision/model.go","internal/decision/split_features.go","docs/split-context-judgment-preregistration.md")}
    args.output.mkdir(parents=True)
    torch.set_num_threads(2)
    torch.manual_seed(20261010)
    initial = {k:v.detach().clone() for k,v in core.Model().state_dict().items()}
    initial_blob = b"".join(v.numpy().astype("<f4").tobytes() for v in initial.values())
    core.save_json(args.output/"preexecution.json", {"schema":"gooo/split-context-training-preexecution/v1",
        "source_revision":head,"sources_sha256":sources,"dataset_sha256":core.sha(raw),"pairs_sha256":core.sha(pair_raw),
        "initialization":"identical random FP32 state across feature versions; no v1 weights reinterpreted",
        "initial_state_sha256":core.sha(initial_blob),"seed":20261010,"epochs_per_trainable_variant":20,
        "maximum_optimizer_steps":560,"device":"mps","torch":torch.__version__,"numpy":np.__version__,
        "feature_versions":{"v1":"positioned_intent_ngrams_v1","v2":VERSION},"native_deployed":False,
        "scope":"Reused synthetic Gooo bilingual pairs; no new intentions or untouched benchmark. Finite soft targets, zero consistency penalty. New ABI requires later explicit SDK/native adoption."})
    reports, exports = {}, {}
    for arm, encoder in (("v1",positioned_features),("v2",split_features)):
        data = {s:arrays(v,rows,encoder) for s,v in grouped.items()}
        fp, fp_report = paired.fit(copy.deepcopy(initial),data,20,20261010,0.0,False)
        qat, qat_report = paired.fit(copy.deepcopy(fp.state_dict()),data,20,20261011,0.0,True)
        reports[arm], exports[arm], parity = {"fp32":fp_report,"qat_ternary":qat_report}, {}, []
        for variant, model in (("fp32",fp),("ptq_ternary",fp),("qat_ternary",qat)):
            directory = args.output/arm/"models"/variant
            core.export_model(model,variant,directory,1.0,1.0)
            cal_rows = [rows[i] for p in grouped["calibration"] for i in p["row_ids"]]
            cal_data = feedback.arrays(cal_rows)
            cal_data = (np.stack([encoder(r["text"]) for r in cal_rows]),cal_data[1],cal_data[2])
            logits = core.exported_logits(directory,cal_data[0])
            temperature = min((0.5,1.0,2.0,4.0),key=lambda t:feedback.score(logits,cal_data,cal_rows,t)["paired_soft_target_nll"])
            meta = json.loads((directory/"model.json").read_text())
            meta.update(schema="gooo/tiny-path-decision-model/v1",feature_version=VERSION if arm=="v2" else "positioned_intent_ngrams_v1",temperature=temperature)
            core.save_json(directory/"model.json",meta)
            exports[arm][variant] = {"metadata_sha256":core.sha((directory/"model.json").read_bytes()),
                "weights_sha256":core.sha((directory/"weights.bin").read_bytes()),"packed_weights_bytes":(directory/"weights.bin").stat().st_size}
            test_rows = [rows[i] for p in grouped["test"] for i in p["row_ids"]]
            x = np.stack([encoder(r["text"]) for r in test_rows])
            logits, probability = core.exported_logits(directory,x), None
            probability = core.probabilities(logits,temperature)
            for i in np.linspace(0,len(test_rows)-1,32,dtype=int):
                parity.append({"variant":variant,"text":test_rows[i]["text"],"features":x[i].tolist(),
                    "logits":logits[i].tolist(),"probabilities":probability[i].tolist(),"selected_label":PATH_LABELS[int(probability[i].argmax())]})
        core.save_json(args.output/arm/"go-parity.json",{"schema":"gooo/typed-path-parity/v1","rows":parity})
    steps = sum(v["optimizer_steps"] for r in reports.values() for v in r.values())
    core.require(steps==560,"bounded optimizer count")
    core.save_json(args.output/"report.json",{"schema":"gooo/split-context-training-report/v1","status":"TRAINED_AND_EXPORTED",
        "preexecution_sha256":core.sha((args.output/"preexecution.json").read_bytes()),"optimizer_steps":steps,"training":reports,"exports":exports,
        "scope":"Identical random initialization, same finite pairs and optimizer configuration; checkpoint/calibration-only selection. No old v1 weight reinterpretation, native deployment, online optimizer or new independent intentions."})
    print(json.dumps({"status":"TRAINED_AND_EXPORTED","optimizer_steps":steps}))


if __name__ == "__main__":
    main()
