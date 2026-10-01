#!/usr/bin/env python3
"""Own random-init matched context study. Offline GPU training/export only."""
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
from split_features_v2 import VERSION, features

core.LABELS = PATH_LABELS


def pair_arrays(pairs, field):
    return (np.asarray([[features(r[field]) for r in p] for p in pairs], dtype=np.float32),
            np.asarray([[PATH_LABELS.index(v) for v in p[0]["eligible_labels"]] for p in pairs], dtype=np.int64),
            np.asarray([p[0]["finite_soft_targets"] for p in pairs], dtype=np.float32))


def finite_nll(logits, data, temperature):
    _, indices, targets = data
    eligible = logits[np.arange(len(logits))[:, None], indices] / temperature
    eligible -= eligible.max(axis=1, keepdims=True)
    e = np.exp(eligible.astype(np.float64))
    p = e / e.sum(axis=1, keepdims=True)
    return float(-(targets * np.log(p.clip(1e-12))).sum(axis=1).mean())


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--source-revision", required=True)
    parser.add_argument("--curriculum", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[1]
    head = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=root, text=True).strip()
    core.require(head == args.source_revision and not subprocess.check_output(["git", "status", "--porcelain"], cwd=root).strip(), "clean exact source required")
    core.require(torch.backends.mps.is_available() and not args.output.exists(), "MPS and fresh output required")
    raw = (args.curriculum / "dataset.jsonl").read_bytes()
    manifest_raw = (args.curriculum / "manifest.json").read_bytes()
    manifest = json.loads(manifest_raw, object_pairs_hook=core.unique_pairs)
    core.require(core.sha(raw) == manifest["dataset_sha256"] and manifest["actual_native_export_calls"] == 2080, "canonical curriculum pin")
    rows = [json.loads(v, object_pairs_hook=core.unique_pairs) for v in raw.splitlines()]
    core.require(len(rows) == 2080 and len({r["id"] for r in rows}) == 2080, "2080 distinct views required")
    pairs = [rows[i:i+2] for i in range(0,len(rows),2)]
    for a,b in pairs:
        core.require(a["pair_id"] == b["pair_id"] and [a["language"],b["language"]] == ["en","ko"], "adjacent bilingual pairs required")
        for field in ("program_id","template_pair_id","split","eligible_labels","finite_soft_targets","finite_cases","intention_label","best_finite_labels"):
            core.require(a[field] == b[field], "paired source/target mismatch")
        for r in (a,b):
            core.require(core.sha(r["text"].encode()) == r["input_sha256"] and core.sha(r["caller_context_text"].encode()) == r["caller_context_sha256"], "full input hashes")
            core.require(r["text"].rsplit("intent: ",1)[-1] == r["caller_context_text"].rsplit("intent: ",1)[-1], "same natural suffix")
            core.require([float(v in r["best_finite_labels"])/len(r["best_finite_labels"]) for v in r["eligible_labels"]] == r["finite_soft_targets"], "finite target mass")
    grouped = {s:[p for p in pairs if p[0]["split"] == s] for s in ("train","calibration","test")}
    core.require({s:len(v) for s,v in grouped.items()} == {"train":800,"calibration":80,"test":160}, "fixed split counts")
    for field in ("program_id","template_pair_id"):
        for a,b in (("train","calibration"),("train","test"),("calibration","test")):
            core.require(not {p[0][field] for p in grouped[a]} & {p[0][field] for p in grouped[b]}, "group leakage")
    sources = {f:core.sha((root/f).read_bytes()) for f in (
        "training/train_compiler_context_v1.py","training/train_bilingual_judgment_v1.py","training/train_pilot_v2.py",
        "training/split_features_v2.py","docs/compiler-context-training-preregistration.md",
        "tools/native-feedback-study/compiler_curriculum.go")}
    torch.set_num_threads(2)
    torch.manual_seed(20261012)
    initial = {k:v.detach().clone() for k,v in core.Model().state_dict().items()}
    initial_blob = b"".join(v.numpy().astype("<f4").tobytes() for v in initial.values())
    args.output.mkdir(parents=True)
    core.save_json(args.output/"preexecution.json",{"schema":"gooo/compiler-context-training-preexecution/v1",
        "source_revision":head,"sources_sha256":sources,"curriculum_manifest_sha256":core.sha(manifest_raw),
        "dataset_sha256":core.sha(raw),"seed":20261012,"initial_state_sha256":core.sha(initial_blob),
        "initialization":"identical own random state; no Laya or inherited pretrained weights",
        "device":"mps","torch":torch.__version__,"numpy":np.__version__,"maximum_optimizer_steps":560,
        "feature_version":VERSION,"context_schema":"gooo/compiler-typed-path-context/v2","new_independent_intentions":0})
    reports,exports = {},{}
    for arm,field in (("caller_context","caller_context_text"),("compiler_context","text")):
        data = {s:pair_arrays(v,field) for s,v in grouped.items()}
        fp,fp_report = paired.fit(copy.deepcopy(initial),data,20,20261012,0.0,False)
        qat,qat_report = paired.fit(copy.deepcopy(fp.state_dict()),data,20,20261013,0.0,True)
        reports[arm],exports[arm],parity = {"fp32":fp_report,"qat_ternary":qat_report},{},[]
        for variant,model in (("fp32",fp),("ptq_ternary",fp),("qat_ternary",qat)):
            directory = args.output/arm/"models"/variant
            core.export_model(model,variant,directory,1.0,1.0)
            x,ix,y = data["calibration"]
            cx,ci,cy = x.reshape(-1,256),np.repeat(ix,2,axis=0),np.repeat(y,2,axis=0)
            logits = core.exported_logits(directory,cx)
            temperature = min((0.5,1.0,2.0,4.0),key=lambda t:finite_nll(logits,(cx,ci,cy),t))
            meta = json.loads((directory/"model.json").read_text())
            meta.update(schema="gooo/tiny-path-decision-model/v1",feature_version=VERSION,temperature=temperature)
            core.save_json(directory/"model.json",meta)
            exports[arm][variant] = {"metadata_sha256":core.sha((directory/"model.json").read_bytes()),
                "weights_sha256":core.sha((directory/"weights.bin").read_bytes()),"packed_weights_bytes":(directory/"weights.bin").stat().st_size}
            test_rows = [r for p in grouped["test"] for r in p]
            x = np.stack([features(r[field]) for r in test_rows])
            logits = core.exported_logits(directory,x)
            probability = core.probabilities(logits,temperature)
            for i in np.linspace(0,len(test_rows)-1,32,dtype=int):
                parity.append({"variant":variant,"text":test_rows[i][field],"features":x[i].tolist(),
                    "logits":logits[i].tolist(),"probabilities":probability[i].tolist(),"selected_label":PATH_LABELS[int(probability[i].argmax())]})
        core.save_json(args.output/arm/"go-parity.json",{"schema":"gooo/typed-path-parity/v1","rows":parity})
    steps = sum(v["optimizer_steps"] for r in reports.values() for v in r.values())
    core.require(steps == 560,"fixed optimizer budget")
    core.save_json(args.output/"report.json",{"schema":"gooo/compiler-context-training-report/v1","status":"TRAINED_AND_EXPORTED",
        "preexecution_sha256":core.sha((args.output/"preexecution.json").read_bytes()),"optimizer_steps":steps,
        "training":reports,"exports":exports,"scope":"same random initialization and finite bilingual targets; calibration-only checkpoint/temperature; reused development cohort, no default model promotion or inherited weights"})
    print(json.dumps({"status":"TRAINED_AND_EXPORTED","optimizer_steps":steps}))


if __name__ == "__main__":
    main()
