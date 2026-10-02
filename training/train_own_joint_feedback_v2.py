"""Offline MPS optimizer/export only; all collection, execution and audits are Go."""
import argparse
import copy
import json
import math
from pathlib import Path
import resource
import subprocess
import time

import numpy as np
import torch
import semantic_features_v3 as v3
import train_pilot_v2 as core

ARMS = ("uniform-initial", "set-initial", "set-feedback")
VARIANTS = ("fp32", "ptq_ternary", "qat_ternary")
PROTOCOL = "docs/own-joint-completeness-feedback-preregistration-20261002.md"
PROTOCOL_SHA = "aad8b228c7c647690abbccd628473f7757517ba09d689a8d46e0e8c180ba6a28"
STATES_SHA = "2fb6ecc9e2085a311456d2fa6c158b3cd8ee0b6c481ed7395df8acf8c28f4aa2"
COLLECTION_SHA = "1a3d4495eb9c1ec2cd389586abb4606b4530ad923e94f45a2926a0a13afca986"
DATASET_SHA = "2d1c9888a648d590e77253666b6e5ec848d375e47a3c99daf29dbde8bcbc7383"
INITIAL_SEED, SHUFFLE_SEED = 20261027, 20261028


def features(text):
    raw, prefix = text.encode("utf-8"), b"gooo;joint2|"
    core.require(len(raw) <= 1088 and raw.startswith(prefix), "bounded complete joint input")
    at, parts = len(prefix), []
    for _ in range(2):
        colon = raw.find(b":", at)
        core.require(colon > at and colon-at <= 3, "canonical joint byte framing")
        digits = raw[at:colon]
        core.require(digits.isdigit() and digits[:1] != b"0", "canonical joint byte length")
        length = int(digits)
        core.require(1 <= length <= 512 and colon+1+length <= len(raw), "full part within source-v3 bound")
        part = raw[colon+1:colon+1+length].decode("utf-8")
        parts.append(v3.features(part))
        at = colon+1+length
    core.require(at == len(raw), "no trailing joint bytes")
    return np.concatenate(parts) * np.float32(1/math.sqrt(2))


def load_states(directory):
    raw = (directory / "states.jsonl").read_bytes()
    report_raw = (directory / "report.json").read_bytes()
    audit = json.loads((directory / "independent-audit.json").read_text(), object_pairs_hook=core.unique_pairs)
    core.require(core.sha(raw) == STATES_SHA and core.sha(report_raw) == COLLECTION_SHA and
                 audit["status"] == "PASS" and audit["collection_report_sha256"] == COLLECTION_SHA and
                 audit["dataset_sha256"] == DATASET_SHA and audit["actual_teacher"]["actual_sdk_sessions"] == 6144 and
                 audit["unique_train_continuation_states"] == 4159, "actual independently audited teacher data required")
    rows = [json.loads(line, object_pairs_hook=core.unique_pairs) for line in raw.splitlines()]
    core.require(len(rows) == len({r["id"] for r in rows}) == 6463, "frozen distinct state rows")
    initial, continuation = {}, {}
    for r in rows:
        expected = "train" if r["configuration"] < 32 else "calibration" if r["configuration"] < 40 else "development"
        core.require(r["split"] == expected and r["language"] in ("en", "ko") and
                     core.sha(r["complete_joint_input"].encode()) == r["input_sha256"], "source group, language and complete input")
        y = np.asarray(r["finite_soft_targets"], dtype=np.float32)
        core.require(y.shape == (4,) and np.isfinite(y).all() and (y >= 0).all() and float(y.sum()) == 1, "finite passing targets")
        key = r["program_contract_group"], r["language"]
        if r["phase"] == "initial":
            core.require(key not in initial and not r["actual_feedback_origins"], "distinct original source input")
            initial[key] = r
        else:
            core.require(r["phase"] == "feedback" and expected == "train" and r["actual_feedback_origins"], "training-only actual continuation")
            continuation.setdefault(key, []).append(r)
    core.require(len(initial) == 2304, "all bilingual initial views")
    for key, states in continuation.items():
        core.require(key in initial and all(s["finite_soft_targets"] == initial[key]["finite_soft_targets"] for s in states), "continuation preserves full contract")
    return rows, initial, continuation


def arrays(initial, continuation, split, arm):
    groups = sorted({group for (group, _), row in initial.items() if row["split"] == split})
    x, target, weights, indices = [], [], [], []
    for group in groups:
        members = []
        for language in ("en", "ko"):
            row = initial[group, language]
            feedback = continuation.get((group, language), []) if arm == "set-feedback" and split == "train" else []
            samples = [(row, .25 if feedback else .5)] + [(s, .25/len(feedback)) for s in feedback]
            for state, weight in samples:
                members.append(len(x))
                x.append(features(state["complete_joint_input"]))
                target.append(state["finite_soft_targets"])
                weights.append(weight)
        core.require(abs(sum(weights[i] for i in members)-1) < 1e-6, "equal function-group weighting")
        indices.append(np.asarray(members, dtype=np.int64))
    return (np.asarray(x, dtype=np.float32), np.asarray(target, dtype=np.float32),
            np.asarray(weights, dtype=np.float32), indices)


def row_loss(logits, target, uniform):
    if uniform:
        return -(target * torch.log_softmax(logits, dim=-1)).sum(dim=-1)
    return torch.logsumexp(logits, dim=-1) - torch.logsumexp(logits.masked_fill(target <= 0, -torch.inf), dim=-1)


def fit(initial_state, data, arm, qat):
    torch.manual_seed(INITIAL_SEED)
    model = core.Model(qat=qat).to("mps")
    model.load_state_dict(initial_state)
    optimizer = torch.optim.AdamW(model.parameters(), lr=.001, weight_decay=.01)
    train = [torch.from_numpy(a).to("mps") for a in data["train"][:3]]
    cal = [torch.from_numpy(a).to("mps") for a in data["calibration"][:3]]
    group_rows = data["train"][3]
    rng = np.random.default_rng(SHUFFLE_SEED)
    best, best_loss, selected_epoch, history = None, math.inf, 0, []
    allocated, driver, updates = 0, 0, 0
    usage_start, started = resource.getrusage(resource.RUSAGE_SELF), time.perf_counter()
    for epoch in range(100):
        order, losses = rng.permutation(len(group_rows)), []
        for begin in range(0, len(order), 128):
            groups = order[begin:begin+128]
            ix = torch.from_numpy(np.concatenate([group_rows[int(i)] for i in groups])).to("mps")
            optimizer.zero_grad(set_to_none=True)
            value = (row_loss(model(train[0][ix]), train[1][ix], arm == "uniform-initial") * train[2][ix]).sum() / len(groups)
            core.require(bool(torch.isfinite(value).cpu()), "finite training objective")
            value.backward()
            optimizer.step()
            updates += 1
            losses.append(float(value.detach().cpu()))
        with torch.no_grad():
            logits = model(cal[0])
            set_nll = float((row_loss(logits, cal[1], False)*cal[2]).sum().cpu()) / len(data["calibration"][3])
            uniform_ce = float((row_loss(logits, cal[1], True)*cal[2]).sum().cpu()) / len(data["calibration"][3])
        if set_nll < best_loss:
            best_loss, selected_epoch = set_nll, epoch+1
            best = {k: v.detach().cpu().clone() for k, v in model.state_dict().items()}
        allocated = max(allocated, torch.mps.current_allocated_memory())
        driver = max(driver, torch.mps.driver_allocated_memory())
        history.append({"epoch":epoch+1, "train_objective":float(np.mean(losses)),
                        "calibration_passing_set_nll":set_nll, "calibration_uniform_target_cross_entropy":uniform_ce})
    torch.mps.synchronize()
    duration, usage = time.perf_counter()-started, resource.getrusage(resource.RUSAGE_SELF)
    cpu_seconds = usage.ru_utime+usage.ru_stime-usage_start.ru_utime-usage_start.ru_stime
    model.cpu().load_state_dict(best)
    core.require(updates == 600, "fixed actual per-stage optimizer budget")
    return model, {"optimizer_steps":updates, "epochs":100, "training_function_groups":768,
        "training_weighted_state_rows":len(train[0]), "selected_epoch":selected_epoch,
        "loop_wall_seconds":duration, "process_cpu_seconds_during_loop":cpu_seconds,
        "process_cpu_percent_one_core_normalization":100*cpu_seconds/duration,
        "process_lifetime_peak_rss_bytes":usage.ru_maxrss, "mps_allocated_peak_sampled_bytes":allocated,
        "mps_driver_peak_sampled_bytes":driver, "selection":"calibration passing-set NLL; no development inputs", "history":history}


def export(model, variant, directory, calibration, parity_rows):
    core.export_model(model, variant, directory, 1., 1.)
    x, targets = calibration[:2]
    logits = core.exported_logits(directory, x)
    def loss_at(temperature):
        p = core.probabilities(logits, temperature)
        return float(-np.log((p*(targets > 0)).sum(axis=-1).clip(1e-12)).mean())
    temperature = min((.5, 1., 2., 4.), key=loss_at)
    metadata = json.loads((directory / "model.json").read_text())
    metadata.update(schema="gooo/tiny-joint-path-model/v1", feature_version="paired_semantic_context_v3_joint_v1", temperature=temperature)
    metadata.pop("confidence_threshold")
    core.save_json(directory / "model.json", metadata)
    vectors = np.stack([features(r["complete_joint_input"]) for r in parity_rows])
    logits = core.exported_logits(directory, vectors)
    p = core.probabilities(logits, temperature)
    parity = [{"variant":variant, "state_id":r["id"], "phase":r["phase"], "text":r["complete_joint_input"],
        "features":vectors[i].tolist(), "logits":logits[i].tolist(), "probabilities":p[i].tolist(),
        "selected_mask":int(p[i].argmax())} for i,r in enumerate(parity_rows)]
    pin = {"metadata_sha256":core.sha((directory/"model.json").read_bytes()),
           "weights_sha256":core.sha((directory/"weights.bin").read_bytes()), "packed_weights_bytes":(directory/"weights.bin").stat().st_size}
    return pin, parity


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--curriculum", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--source-revision", required=True)
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[1]
    head = subprocess.check_output(["git","rev-parse","HEAD"],cwd=root,text=True).strip()
    core.require(head == args.source_revision and not subprocess.check_output(["git","status","--porcelain"],cwd=root).strip(), "clean exact optimizer source")
    core.require(torch.backends.mps.is_available() and not args.output.exists(), "local MPS and fresh output")
    core.require(core.sha((root/PROTOCOL).read_bytes()) == PROTOCOL_SHA, "frozen v2 protocol")
    core.FEATURES, core.HIDDEN, core.LIMIT, core.LABELS = 512, 24, 1088, [f"mask_{i}" for i in range(4)]
    rows, initial, continuation = load_states(args.curriculum)
    torch.manual_seed(INITIAL_SEED)
    initial_state = copy.deepcopy(core.Model().state_dict())
    initial_sha = core.sha(b"".join(v.numpy().astype("<f4").tobytes() for v in initial_state.values()))
    sources = {name:core.sha((root/"training"/name).read_bytes()) for name in
               ("train_own_joint_feedback_v2.py", "semantic_features_v3.py", "train_pilot_v2.py")}
    pre = {"schema":"gooo/own-joint-feedback-training-preexecution/v2", "source_revision":head,
        "sources_sha256":sources, "protocol_sha256":PROTOCOL_SHA, "source_dataset_sha256":DATASET_SHA,
        "student_states_sha256":STATES_SHA, "collection_report_sha256":COLLECTION_SHA, "initial_state_sha256":initial_sha,
        "initial_seed":INITIAL_SEED, "shuffle_seed":SHUFFLE_SEED, "arms":ARMS, "device":"mps",
        "torch":torch.__version__, "numpy":np.__version__, "planned_optimizer_steps":3600,
        "initialization":"One new own random state shared by matched students; no pretrained/Laya/v1 student weights inherited. QAT starts its student's own FP32 checkpoint. The old own teacher supplies observed failures only.",
        "gpu_utilization_measured":False, "host_cpu_causal_delta_measured":False, "default_model_promoted":False}
    core.save_json(args.output/"preexecution.json",pre)
    development = [r for r in rows if r["split"] == "development"]
    feedback_rows = [r for r in rows if r["phase"] == "feedback"]
    parity_rows = [development[i] for i in np.linspace(0,len(development)-1,32,dtype=int)] + [feedback_rows[i] for i in np.linspace(0,len(feedback_rows)-1,16,dtype=int)]
    reports, exports = {}, {}
    for arm in ARMS:
        data = {s:arrays(initial,continuation,s,arm) for s in ("train","calibration")}
        core.require(len(data["train"][3]) == 768 and len(data["calibration"][3]) == 192, "fixed disjoint function groups")
        fp, fp_report = fit(copy.deepcopy(initial_state),data,arm,False)
        qat, qat_report = fit(copy.deepcopy(fp.state_dict()),data,arm,True)
        reports[arm], exports[arm], parity = {"fp32":fp_report, "qat_ternary":qat_report}, {}, []
        for variant, model in (("fp32",fp),("ptq_ternary",fp),("qat_ternary",qat)):
            pin, records = export(model,variant,args.output/arm/"models"/variant,data["calibration"],parity_rows)
            exports[arm][variant] = pin
            parity.extend(records)
        core.save_json(args.output/arm/"go-parity.json",{"schema":"gooo/own-joint-feedback-parity/v2","rows":parity})
        print(json.dumps({"completed_arm":arm,"actual_optimizer_updates":fp_report["optimizer_steps"]+qat_report["optimizer_steps"]}),flush=True)
    steps = sum(stage["optimizer_steps"] for arm in reports.values() for stage in arm.values())
    core.require(steps == 3600, "fixed total actual optimizer steps")
    core.save_json(args.output/"report.json",{"schema":"gooo/own-joint-feedback-training-report/v2",
        "status":"TRAINED_AND_EXPORTED", "optimizer_steps":steps, "preexecution_sha256":core.sha((args.output/"preexecution.json").read_bytes()),
        "training":reports, "exports":exports, "default_model_promoted":False,
        "scope":"Matched initial objectives and a separate observed-failure curriculum arm. Same source-v3 ABI and fresh shared own random initialization. Nine exports preserved; student Go completeness and native compiled execution remain separate stages."})
    print(json.dumps({"status":"TRAINED_AND_EXPORTED","optimizer_steps":steps,"model_exports":9}),flush=True)


if __name__ == "__main__":
    main()
