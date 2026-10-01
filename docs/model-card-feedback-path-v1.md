---
license: mit
language:
  - ko
  - en
tags:
  - gooo
  - metaprogramming
  - prov-o
  - finite-tdd
  - ternary
  - experimental
---

# Gooo feedback path tiny v1

Experimental Korean/English judgment of compiler-owned Gooo construction paths.
The model prioritizes bounded choices for local references, assignment targets,
branch layout and root statement order. Gooo assembles typed bodies, tests them,
retains partial results and supplies observed failures for the next judgment.
First-shot exactness is not the development objective.

These are our own 12,728-parameter models, fine tuned from our positioned
structural model. Upstream Laya motivated the small decision-model experiments;
these checkpoints contain no Laya weights and do not call Laya or an external
inference service. Runtime, orchestration, arrays and publication verification
are Go. Python/PyTorch was used only for offline MPS training.

## Evidence and scope

Synthetic data has 6,240 views of 2,080 existing original instruction groups:
4,800 train, 480 calibration, 960 development test. Original groups/templates,
configurations, programs and intention hashes retain their previous disjoint
splits. The test split has already been used for development; it is not an
untouched holdout. Three views per instruction use complete finite cases, a
sparse case, or a deliberately contradictory duplicate case. All tied best
finite options retain equal target mass; the original intention label is kept
separately. There are 496 ambiguous views overall, 76 in development test.
Caller CI PASS/FAIL/UNKNOWN context is synthetic and is not verified authority.
No additional independent intention family was created by these repeated views.

FP32 and quantization-aware training each executed 380 optimizer steps on MPS,
760 total. The checkpoint and temperature were selected using calibration
paired soft-target NLL. PTQ derives from the new FP32 checkpoint.

| Arm | Finite best set selected / 960 | Finite cases passed / 5,120 |
| --- | ---: | ---: |
| Parent FP32 | 814 | 4,072 |
| Feedback FP32 | 754 | 3,748 |
| Parent PTQ | 808 | 4,038 |
| Feedback PTQ | 786 | 3,895 |
| Parent QAT | 817 | 4,076 |
| Feedback QAT | 756 | 3,742 |
| Deterministic fallback | 518 | 2,514 |

**This iteration regressed against its parent on these measurements.** The
negative result is retained. Finite test agreement does not prove full natural
language meaning or unseen behavior. Existing compiler defaults are not replaced
on the basis of these numbers. All models retain global abstention at threshold
1.0; explicit compiler-owned pair ranking can still explore bounded TDD paths.
The probabilities are ranking hints and are not calibrated semantic confidence.

Go made 5,760 actual development predictions plus 96 numerical parity predictions;
maximum Go/Python logit/probability difference was 0.00000166893 (tolerance 0.0001).
The disconnected control made zero predictions.

The published `native/` feature-source dogfood separately made six native calls,
108 predictions (72 feedback), and 384 candidate attempts. All six policy/language
views retained 6/7 partial completeness for an intentionally inconsistent
seven-case contract. One exact generated Go program was deduplicated and actually
executed on seven original inputs plus nine existing disjoint inputs; all 16
values agreed with the authored original expectations. This is one compound
intention, not six tasks or a new language benchmark. Go audit reinterprets the
captures and verifies progress, model, intention and feedback receipt digests
with zero new predictions or executions.

## Size and measured cost

| Variant | Weight file bytes | Resident tensor bytes |
| --- | ---: | ---: |
| FP32 | 50,912 | 50,912 |
| PTQ ternary | 2,759 | 12,896 |
| QAT ternary | 2,759 | 12,896 |

Ternary matrix storage packs five base-three values per byte: 1.6 disk bits per
matrix weight; 1.585 is its theoretical information bound. Go expands matrices
to int8, with eight scale bytes and a 1,248-byte per-worker workspace. File size
and model arrays are not total process RAM or a packed-compute acceleration claim.

The sampled training loops took 0.970133 seconds FP32 and 0.406094 seconds QAT,
excluding feature preparation/export/startup. Sampled allocated MPS peak was
5,810,944 bytes, sampled driver allocation 53,166,080 bytes, and Python process
lifetime RSS reached 523,583,488 bytes. Host GPU utilization was not sampled.
Published feature-source native process wall times were 10.051–54.775 ms; the
first 54.775 ms observation remains included. Child peak RSS was
21,577,728–22,216,704 bytes. Process CPU time/wall was measured; host CPU usage
increase was not measured. These are six fixed-order observations without repeats.

## Use with Gooo

Download one variant's `models/<variant>/{model.json,weights.bin}`. Use Gooo
compiler source `4dced73b26dde567cb7129f3e4ba5733850196d0` (SDK
`v0.2.4-experimental`) or a compatible later version, with Go 1.27.1:

```sh
gooo body-codegen --json --path-plan plan.json \
  --path-model models/qat_ternary/model.json \
  --path-step-attempts 8 --path-feedback-rounds 2 \
  --activity ConditionalAssign fixture.gooo
```

`plan.json` declares typed alternatives and finite expected cases. Inspect the
functional completeness and failed cases independently of the command's
verification PASS. Omit `--path-model` and feedback flags for deterministic,
disconnected construction. Optional feedback input overflow records a zero-call
context decline and preserves the remaining frontier; it does not truncate the
original intention. CI can audit and replay; local generation needs no CI roundtrip.

## Provenance and contents

Training source: `439da7bce8e801da94720696fbf16165a3bd099e`.
Native dogfood runner: `1cfeb4753fdcb62aa2b9cfad3002c1c0172b5328`.
Frozen feature compiler: `6a3e45d7f185c74f5136ab81b560a14a1366f367`.
The repository includes models, complete synthetic data, training/parity/audit
records and allowlisted native captures, with a SHA-256 publication manifest.
No credentials, private repository text, environment dump or local host paths
are intentionally included. The fixed allowlist is mechanically checked in Go.

[Source, evaluation tools and main-deployment evidence](https://github.com/kimjooyoon/gooo-neural-decision-experiments).
[Native compiler main promotion](https://github.com/kimjooyoon/meta-ontology-go/pull/1121).
