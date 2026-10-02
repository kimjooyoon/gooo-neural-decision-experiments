"""Offline MPS optimization/export only. Go supplies all features and weights."""
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
import train_pilot_v2 as core

ARMS = ("uniform-initial", "set-initial", "set-feedback")
TEMPERATURES = (.5, 1., 2., 4.)
PROTOCOL = "docs/own-three-choice-completeness-preregistration-20261002.md"
PROTOCOL_SHA = "5d840f7b3379339d684b85b7896a895538bd1a030395a096af3e52c5fc781cd3"
STATES_SHA = "38930441c45ec03336a66c670028208e525131f385699140d793da96cbf4028b"
AUDIT_SHA = "dc1bc90a5468b2f45a4e32501f0814a73ee2be7de56481c1edea3b7d414bfc6a"
INITIAL_SEED, SHUFFLE_SEED = 20261031, 20261032
RAW_CAP = 768 << 20


def read_json(path):
    return json.loads(path.read_text(), object_pairs_hook=core.unique_pairs)


def retained_bytes(root):
    total = 0
    for phase in (root / "runs").glob("own-three-*"):
        core.require(phase.is_dir() and not phase.is_symlink(), "regular phase directory")
        for path in phase.rglob("*"):
            core.require(not path.is_symlink(), "no raw evidence symlinks")
            if path.is_file():
                total += path.stat().st_size
    core.require(total <= RAW_CAP, "whole-study cap; retain prefix and stop")
    return total


def save(path, value, root):
    raw = (json.dumps(value, ensure_ascii=False, indent=2, sort_keys=True, allow_nan=False) + "\n").encode()
    core.require(len(raw) <= 1 << 20 and retained_bytes(root) + len(raw) <= RAW_CAP,
                 "bounded metadata and whole-study raw cap")
    core.require(not path.exists(), "immutable fresh optimizer output")
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("xb") as out:
        out.write(raw)


def save_jsonl(path, rows, root):
    core.require(not path.exists(), "fresh bounded parity journal")
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("xb") as out:
        for row in rows:
            raw = (json.dumps(row, ensure_ascii=False, sort_keys=True, allow_nan=False) + "\n").encode()
            core.require(len(raw) <= 1 << 20 and retained_bytes(root) + len(raw) <= RAW_CAP,
                         "bounded parity line and whole-study raw cap")
            out.write(raw)
            out.flush()


def record_update(path, value, root):
    raw = (json.dumps(value, sort_keys=True, allow_nan=False) + "\n").encode()
    core.require(len(raw) <= 1 << 20 and retained_bytes(root) + len(raw) <= RAW_CAP-(1 << 20),
                 "reserve a failure receipt and preserve actual update prefix")
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("xb" if value["actual_stage_updates"] == 1 else "ab") as out:
        out.write(raw)


def load_inputs(directory, audit_path):
    manifest_raw = (directory / "manifest.json").read_bytes()
    manifest, audit = read_json(directory / "manifest.json"), read_json(audit_path)
    core.require(manifest["schema"] == "gooo/own-three-training-inputs/v1" and
                 manifest["student_state_rows"] == 10739 and manifest["feature_dim"] == 768 and
                 manifest["label_count"] == 8 and manifest["planned_optimizer_steps"] == 4800 and
                 audit["status"] == "PASS" and audit["manifest_sha256"] == core.sha(manifest_raw) and
                 audit["student_states_sha256"] == STATES_SHA and audit["teacher_audit_sha256"] == AUDIT_SHA and
                 audit["feature_float32_values_recomputed"] == 10739*768 and
                 audit["model_predictions"] == audit["optimizer_updates"] == audit["native_calls"] == 0,
                 "independent Go replay of all source-bound features required")
    for name, pin in manifest["files"].items():
        raw = (directory / name).read_bytes()
        core.require(len(raw) == pin["bytes"] and core.sha(raw) == pin["sha256"], "exact Go optimizer input pin")
    rows = [json.loads(line, object_pairs_hook=core.unique_pairs)
            for line in (directory / "rows.jsonl").read_bytes().splitlines()]
    core.require(len(rows) == 10739 and all(r["feature_row_index"] == i for i, r in enumerate(rows)), "complete fixed feature row order")
    x = np.fromfile(directory / "features-f32le.bin", dtype="<f4").reshape(10739, 768)
    core.require(np.isfinite(x).all(), "finite Go features")
    return manifest, rows, x, read_json(directory / "parity-inputs.json")["rows"]


def state_from_blob(path):
    array = np.fromfile(path, dtype="<f4")
    core.require(array.shape == (18656,) and np.isfinite(array).all(), "fresh Go FP32 initializer")
    at, result = 0, {}
    for key, shape in (("first.weight", (24, 768)), ("first.bias", (24,)),
                       ("last.weight", (8, 24)), ("last.bias", (8,))):
        n = math.prod(shape)
        result[key] = torch.from_numpy(array[at:at+n].copy().reshape(shape))
        at += n
    return result


def state_sha(state):
    return core.sha(b"".join(state[key].detach().cpu().numpy().astype("<f4").tobytes()
                            for key in ("first.weight", "first.bias", "last.weight", "last.bias")))


def arrays(rows, x, split, arm):
    field = "feedback_arm_row_weight" if arm == "set-feedback" else "initial_arm_row_weight"
    selected = [i for i, r in enumerate(rows) if r["split"] == split and r[field] > 0]
    groups, indices = {}, []
    weights = np.asarray([rows[i][field] for i in selected], dtype=np.float32)
    target = np.asarray([rows[i]["finite_soft_targets"] for i in selected], dtype=np.float32)
    core.require(target.shape == (len(selected), 8) and np.isfinite(target).all() and
                 (target >= 0).all() and np.max(np.abs(target.sum(axis=1)-1)) < 1e-6, "full finite passing-mask targets")
    for local, original in enumerate(selected):
        groups.setdefault(rows[original]["program_contract_group"], []).append(local)
        core.require(split == "train" or rows[original]["phase"] == "initial", "no holdout feedback optimization")
    for group in sorted(groups):
        ix = np.asarray(groups[group], dtype=np.int64)
        core.require(abs(float(weights[ix].sum())-1) < 1e-6, "equal function group weight")
        indices.append(ix)
    expected = 1024 if split == "train" else 256
    core.require(len(indices) == expected, "fixed disjoint train/calibration groups")
    return x[selected], target, weights, indices


def row_loss(logits, target, uniform):
    if uniform:
        return -(target * torch.log_softmax(logits, dim=-1)).sum(dim=-1)
    return torch.logsumexp(logits, dim=-1) - torch.logsumexp(logits.masked_fill(target <= 0, -torch.inf), dim=-1)


def passing_nll(logits, targets, temperature):
    # Stable float64 CPU logsumexp is calibration arithmetic, not optimization.
    z = logits.astype(np.float64) / temperature
    z -= z.max(axis=1, keepdims=True)
    all_log = np.log(np.exp(z).sum(axis=1))
    passing = np.where(targets > 0, z, -np.inf)
    maximum = passing.max(axis=1)
    pass_log = maximum + np.log(np.exp(passing - maximum[:, None]).sum(axis=1))
    return all_log-pass_log


def fit(initial_state, data, arm, qat, output, root):
    torch.manual_seed(INITIAL_SEED)
    model = core.Model(qat=qat)
    model.load_state_dict(initial_state)
    input_sha = state_sha(model.state_dict())
    model.to("mps")
    optimizer = torch.optim.AdamW(model.parameters(), lr=.001, weight_decay=.01)
    train = [torch.from_numpy(a).to("mps") for a in data["train"][:3]]
    cal_x = torch.from_numpy(data["calibration"][0]).to("mps")
    cal_y, cal_w = data["calibration"][1:3]
    group_rows = data["train"][3]
    rng = np.random.default_rng(SHUFFLE_SEED)
    best, selected = None, (math.inf, 0, 0.)
    history, updates, allocated, driver = [], 0, 0, 0
    started, before = time.perf_counter(), resource.getrusage(resource.RUSAGE_SELF)
    for epoch in range(100):
        order, losses = rng.permutation(len(group_rows)), []
        for begin in range(0, len(order), 128):
            groups = order[begin:begin+128]
            ix = torch.from_numpy(np.concatenate([group_rows[int(i)] for i in groups])).to("mps")
            optimizer.zero_grad(set_to_none=True)
            loss = (row_loss(model(train[0][ix]), train[1][ix], arm == "uniform-initial") * train[2][ix]).sum() / len(groups)
            core.require(bool(torch.isfinite(loss).cpu()), "finite MPS objective")
            loss.backward()
            optimizer.step()
            updates += 1
            losses.append(float(loss.detach().cpu()))
            record_update(output / ("qat" if qat else "fp32") / "optimizer-updates.jsonl",
                          {"arm": arm, "stage": "qat" if qat else "fp32", "epoch": epoch+1,
                           "batch": begin//128+1, "function_groups": len(groups),
                           "actual_stage_updates": updates, "objective": losses[-1]}, root)
        with torch.no_grad():
            logits = model(cal_x).cpu().numpy()
        candidates = [(float((passing_nll(logits, cal_y, t)*cal_w).sum())/256, epoch+1, t)
                      for t in TEMPERATURES]
        choice = min(candidates)
        if choice < selected:
            selected = choice
            best = {k: v.detach().cpu().clone() for k, v in model.state_dict().items()}
        allocated = max(allocated, torch.mps.current_allocated_memory())
        driver = max(driver, torch.mps.driver_allocated_memory())
        record = {"epoch": epoch+1, "actual_stage_updates": updates, "train_objective": float(np.mean(losses)),
                  "calibration_passing_set_nll_by_temperature": {str(t): v for v, _, t in candidates}}
        history.append(record)
        # Each completed epoch survives cancellation/failure; never retry a prefix.
        save(output / ("qat" if qat else "fp32") / f"epoch-{epoch+1:03d}.json", record, root)
        if (epoch+1) % 20 == 0:
            print(json.dumps({"arm": arm, "stage": "qat" if qat else "fp32", "epoch": epoch+1,
                              "actual_stage_updates": updates}), flush=True)
    torch.mps.synchronize()
    wall, after = time.perf_counter()-started, resource.getrusage(resource.RUSAGE_SELF)
    cpu = after.ru_utime+after.ru_stime-before.ru_utime-before.ru_stime
    core.require(updates == 800 and best is not None, "fixed actual per-stage update budget")
    model.cpu().load_state_dict(best)
    return model, {"optimizer_steps": updates, "epochs": 100, "training_function_groups": 1024,
                   "training_weighted_state_rows": len(train[0]), "initial_state_sha256": input_sha,
                   "selected_checkpoint_sha256": state_sha(best), "selected_epoch": selected[1],
                   "selected_temperature": selected[2], "selected_calibration_passing_set_nll": selected[0],
                   "selection": "minimum (calibration passing-set NLL, epoch, temperature) across all 100 epochs and four frozen temperatures",
                   "loop_wall_seconds": wall, "process_cpu_seconds_during_loop": cpu,
                   "process_cpu_percent_one_core_normalization": 100*cpu/wall,
                   "process_lifetime_peak_rss_bytes": after.ru_maxrss,
                   "mps_allocated_peak_sampled_bytes": allocated, "mps_driver_peak_sampled_bytes": driver,
                   "history": history}


def export(model, variant, directory, calibration, parity_inputs, x, selected_temperature):
    core.export_model(model, variant, directory, 1., 1.)
    cal_logits = core.exported_logits(directory, calibration[0])
    if variant == "ptq_ternary":
        temperature = min(TEMPERATURES, key=lambda t: (float((passing_nll(cal_logits, calibration[1], t)*calibration[2]).sum())/256, t))
    else:
        temperature = selected_temperature
    metadata = read_json(directory / "model.json")
    metadata.update(schema="gooo/tiny-three-choice-path-model/v1",
                    feature_version="triple_semantic_context_v3_joint_v1", temperature=temperature)
    metadata.pop("confidence_threshold")
    core.save_json(directory / "model.json", metadata)
    vectors = x[[r["feature_row_index"] for r in parity_inputs]]
    logits = core.exported_logits(directory, vectors)
    probabilities = core.probabilities(logits, temperature)
    parity = [{"variant": variant, "state_id": r["state_id"], "phase": r["phase"], "text": r["text"],
               "features": vectors[i].tolist(), "logits": logits[i].tolist(), "probabilities": probabilities[i].tolist(),
               "selected_mask": int(probabilities[i].argmax())} for i, r in enumerate(parity_inputs)]
    pin = {"metadata_sha256": core.sha((directory / "model.json").read_bytes()),
           "weights_sha256": core.sha((directory / "weights.bin").read_bytes()),
           "packed_weights_bytes": (directory / "weights.bin").stat().st_size,
           "temperature": temperature, "calibration_export_passing_set_nll": float((passing_nll(cal_logits, calibration[1], temperature)*calibration[2]).sum())/256}
    core.require(pin["packed_weights_bytes"] == (74624 if variant == "fp32" else 3854), "exact separate three-choice weight ABI")
    return pin, parity


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--prepared", type=Path, required=True)
    parser.add_argument("--preparation-audit", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--source-revision", required=True)
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[1]
    head = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=root, text=True).strip()
    core.require(head == args.source_revision and not subprocess.check_output(["git", "status", "--porcelain"], cwd=root).strip(), "clean exact offline optimizer source")
    core.require(torch.backends.mps.is_available() and not args.output.exists(), "actual local MPS and fresh output")
    core.require(args.output.resolve().parent == root / "runs" and args.output.name.startswith("own-three-"), "capped own-three raw phase")
    core.require(core.sha((root / PROTOCOL).read_bytes()) == PROTOCOL_SHA, "immutable three-choice protocol")
    core.FEATURES, core.HIDDEN, core.LIMIT, core.LABELS = 768, 24, 1600, [f"mask_{i}" for i in range(8)]
    manifest, rows, x, parity_inputs = load_inputs(args.prepared, args.preparation_audit)
    initial_state = state_from_blob(args.prepared / "initial-fp32.bin")
    initial_sha = state_sha(initial_state)
    core.require(initial_sha == manifest["files"]["initial-fp32.bin"]["sha256"], "Go common initialization preserved exactly")
    pre = {"schema": "gooo/own-three-choice-training-preexecution/v1", "source_revision": head,
           "sources_sha256": {name: core.sha((root / "training" / name).read_bytes())
                              for name in ("train_own_three_feedback_v1.py", "train_pilot_v2.py")},
           "protocol_sha256": PROTOCOL_SHA, "student_states_sha256": STATES_SHA, "teacher_audit_sha256": AUDIT_SHA,
           "prepared_manifest_sha256": core.sha((args.prepared / "manifest.json").read_bytes()),
           "preparation_audit_sha256": core.sha(args.preparation_audit.read_bytes()),
           "initial_state_sha256": initial_sha, "initial_seed": INITIAL_SEED, "shuffle_seed": SHUFFLE_SEED,
           "arms": ARMS, "device": "mps", "torch": torch.__version__, "numpy": np.__version__,
           "planned_optimizer_steps": 4800, "optimizer": "AdamW", "learning_rate": .001, "weight_decay": .01,
           "temperatures": TEMPERATURES, "prior_retained_raw_bytes": retained_bytes(root), "whole_study_raw_cap_bytes": RAW_CAP,
           "initialization": "Byte-exact new common Go initializer. Teacher observations only; no prior/pretrained/Laya student weights. QAT starts its own jointly calibration-selected FP checkpoint.",
           "gpu_utilization_measured": False, "host_cpu_causal_delta_measured": False, "default_model_promoted": False}
    save(args.output / "preexecution.json", pre, root)
    reports, exports = {}, {}
    total_updates = 0
    try:
        for arm in ARMS:
            data = {s: arrays(rows, x, s, arm) for s in ("train", "calibration")}
            fp, fp_report = fit(copy.deepcopy(initial_state), data, arm, False, args.output / arm, root)
            total_updates += fp_report["optimizer_steps"]
            qat, qat_report = fit(copy.deepcopy(fp.state_dict()), data, arm, True, args.output / arm, root)
            total_updates += qat_report["optimizer_steps"]
            reports[arm], exports[arm], parity = {"fp32": fp_report, "qat_ternary": qat_report}, {}, []
            for variant, model, temperature in (("fp32", fp, fp_report["selected_temperature"]),
                                                ("ptq_ternary", fp, None),
                                                ("qat_ternary", qat, qat_report["selected_temperature"])):
                core.require(retained_bytes(root) + 74624 + (64 << 10) <= RAW_CAP-(1 << 20),
                             "reserve complete model and failure metadata before export")
                pin, records = export(model, variant, args.output / arm / "models" / variant,
                                      data["calibration"], parity_inputs, x, temperature)
                exports[arm][variant] = pin
                parity.extend(records)
            save_jsonl(args.output / arm / "go-parity.jsonl", parity, root)
            save(args.output / arm / "training-report.json", reports[arm], root)
            retained_bytes(root)
            print(json.dumps({"completed_arm": arm, "actual_optimizer_updates": total_updates}), flush=True)
        core.require(total_updates == 4800, "fixed total actual optimizer budget")
        save(args.output / "report.json", {"schema": "gooo/own-three-choice-training-report/v1", "status": "TRAINED_AND_EXPORTED",
                                           "optimizer_steps": total_updates, "preexecution_sha256": core.sha((args.output / "preexecution.json").read_bytes()),
                                           "training": reports, "exports": exports, "default_model_promoted": False,
                                           "scope": "All nine matched fresh three-choice FP/PTQ/QAT exports retained. Offline MPS optimization only; Go numerical parity, SDK continuation and native compilation remain separate uncompleted stages."}, root)
    except BaseException as error:
        save(args.output / "failure.json", {"status": "FAILED_PREFIX_RETAINED", "error_type": type(error).__name__,
                                            "completed_stage_optimizer_updates": total_updates,
                                            "scope": "Completed epoch receipts retain actual per-stage update counts. No restart, trimming, arm replacement or promotion."}, root)
        raise
    print(json.dumps({"status": "TRAINED_AND_EXPORTED", "optimizer_steps": total_updates, "model_exports": 9}), flush=True)


if __name__ == "__main__":
    main()
