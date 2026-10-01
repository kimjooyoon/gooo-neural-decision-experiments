# Retained native Gooo construction pilot

## Implemented boundary

The optional `gooo-body-worker` shares one immutable, explicitly loaded structural
model. It constructs a fresh source-bound typed plan and private SDK session for
each Korean/English request, then emits actual native Go with type/replay and
finite-case receipts. Disconnected requests use the same deterministic native
construction flow. No input source files, weights or intent are updated by a call.

This is a bounded metaprogramming decision experiment: the small model selects
among declared reference, assignment-target, operand-order, branch and scheduling
coordinates. Local TDD failures can guide remaining choices. It does not generate
arbitrary free-form source or establish broad natural-language understanding.
First-shot accuracy is not acceptance. Partial completeness and iteration cost
are recorded separately from complete compiler lowering of the chosen body.

The new worker is implemented in native feature
`faf932f85c6acc293ce3839a563665661cb53665` with SDK 0.2.7. Its implementation and
lifecycle tests are public in [PR 1126](https://github.com/kimjooyoon/meta-ontology-go/pull/1126).
Measurement used clean Go 1.27.1 binaries and runner
`be3ccb633446ef4379af964007f8d25766d6726b`. Main deployment is a separate fact;
these captures remain feature evidence even after the code merges.

## Actual construction and independent replay

The [preregistered protocol](retained-native-preregistration.md) reused six bilingual
contradictory views (three structural templates), with a forward and reverse pass.
Five frozen arms each performed 12 fresh-process, 12 sequential-retained and 12
parallel-4 constructions. Translations, repeats and modes are not new independent
experiments. Four arms used own tiny models; one was disconnected.

- 180 verified native constructions; 70 native processes.
- 720 actual predictions: 288 initial and 432 feedback predictions.
- 10 invalid-source rejections, each before prediction, followed by clean requests.
- 120 fresh/retained pairs preserved candidate sequence, selected Go and actuals.
- Finite result: 1,260 / 1,440 repeated cases, **87.5%**; contradictory cases remain.
- Three distinct actual Go executions, 42 function evaluations including int64
  extremes, reconciled with independently authored integer-state oracles.
- Source/document/suite/model identity, selected native function AST, progress and
  original failure/CI chains replay offline without predictions or subprocesses.

Original CI context is **UNKNOWN**, bound to the measured compiler revision and
unauthenticated. Subsequent CI status is recorded separately and never replaces it.
Raw JSON/NDJSON and timing/resource sidecars were saved before inspection.

All five arms, including the disconnected control, reached the same 87.5% finite
completeness. This pilot shows no functional gain from the model: the full four-path
budget already finds the finite best body, and one case is deliberately contradictory.
Model value under a smaller construction budget, unfamiliar expressions or more
interacting coordinates requires a separate experiment. The observed worker cost
changes should not be presented as improved language judgment.

## Observed costs

Each cell uses the same fixed 12 valid requests per model/mode.

| Own model | Fresh response p50 ms | Retained sequential p50 ms | Fresh CPU ms / request | Retained sequential CPU ms / request | Fresh RSS p50 MiB | Retained process peak MiB |
|---|---:|---:|---:|---:|---:|---:|
| Parent FP32 | 5.917 | 1.211 | 5.311 | 1.778 | 17.219 | 20.766 |
| Feedback FP32 | 5.876 | 1.174 | 5.143 | 1.706 | 17.297 | 20.203 |
| PTQ | 5.985 | 1.219 | 5.257 | 1.715 | 17.016 | 20.875 |
| QAT | 6.140 | 1.281 | 5.372 | 1.626 | 17.102 | 21.188 |

Fresh response includes process startup/model load. Retained sequential response
excludes one startup/load, which the setup record measures separately. Whole
retained CPU includes startup and one rejected request, divided by 12 valid
requests; fresh CPU aggregates 12 processes. Fresh RSS is a median across processes;
retained RSS is one process's peak. These boundaries preclude a simple causal
speedup claim, but make the observed amortization and memory increase inspectable.

For four parallel workers, own-model native-body p50 grew from sequential
0.665–0.730 ms to **1.837–2.116 ms**. Enqueue-to-result p50 was **5.671–6.238 ms**,
including queueing. Whole-process throughput was about **826–980 valid requests/s**,
versus sequential **576–615/s**. CPU per valid request increased to
**3.012–3.433 ms**, and process peak RSS to **21.875–22.266 MiB**. One-core child CPU
was about 284–295%, meaning the process used multiple cores; this is not a measured
change in host CPU utilization. The default remains one worker.

The first disconnected fresh binary invocation had **466.889 ms** wall time. The
first retained binary invocation's constructor-record arrival was **422.647 ms**,
while recorded disconnected constructor work was only 0.000625 ms. Both startup
outliers remain in full-process totals. Their cause was not instrumented. Do not
compare the cold disconnected totals to later model arms as an inference effect.

Decoded tensors are 50,912 bytes for FP32 and 12,896 bytes for ternary models,
separate from compiler/transport/process memory. Ternary storage is 1.6 bits per
weight (1.58 theoretical); compute uses decoded int8 arrays. No new training,
optimizer steps, GPU work or upstream Laya HTTP calls occurred. Controller CPU,
memory and serialization costs are outside the child resource measurements.
The recorded worker peak covers only 13 requests in this small workload. It is
not an upper bound for maximal source/plan/case receipts or a long running stream.

## Next bounded use

Use the retained sequential worker for repeated local Gooo body assembly where
setup can be amortized. Use parallel mode when batch throughput matters and its
additional memory/CPU is acceptable. Extend authored source/IR/finite contracts
to new structures before training additional judgment coordinates. Keep intent
ambiguity, failure traces and partial progress visible; do not turn partial case
scores or CI context into source authority. New tuning remains a separate measured
curriculum/GPU phase. Existing checkpoint regressions and earlier evidence remain
published, with weights/card unchanged.
