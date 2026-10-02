# Own-model Gooo generation, native execution and reverse observations

The [immutable public HF evidence](https://huggingface.co/asketeddy/gooo-three-choice-feedback-tiny-v1/tree/24238ca67048b36bc305731799985271bb752dd1/research/native-runtime-20261002)
captures compiler `7a71f9871ae4b6cecf1f9e5e1170961a5e47ced8` from
[compiler PR 1144](https://github.com/kimjooyoon/meta-ontology-go/pull/1144).
The reusable Go runner is [native-runtime-observe](../cmd/native-runtime-observe/main.go).
It performs each generation followed immediately by its native runtime
observation; it does not wait for a batch of all generations or call the model
again after emission.

## Actual use and finite results

We used the retained own FP32 model on all eight composition families, Korean
and English, configuration 20 / goal 4, alongside disconnected deterministic
search. These are already observed frozen source inputs. No optimizer update or
model-weight change occurred in this experiment.

- 32 real Gooo generations: 16 own-model and 16 disconnected.
- 64 local model predictions during selection/feedback; zero in disconnected mode.
- 64 compiled-program executions; 1,536 ordered output observations.
- 768/768 supplied expectations matched, including 256 inputs absent from the
  current selection suites. There are 24 expectations per generated program.
- Runtime/replay stages make zero new model or external provider calls.

Each runtime suite keeps its original 16 selection cases and adds
`-257, -127, -31, -7, 7, 31, 127, 257`. Expected values come from the existing
independent ordinary-Go oracle, without inspecting the selected generated body.
This disjointness is only relative to the current selection suite. It is not
model-training holdout independence, unseen-source generalization or evidence of
improved first-choice accuracy. Earlier bilingual disagreement and continuation
results remain intact.

## What the compiler now measures

`gooo body-execute` checks original source, complete typed document, chosen paths
and exact regenerated Go before compiling it with local Go 1.27.1. It executes
the result twice and composes the Gooo-declared shared completeness schema.
Each observation binds tool bytes, executable bytes, generated/selected/original
source, stable activity identity, ordered inputs/expected/actual values and
process resource measurements.

An independent consumer uses the actual compiler `completeness.Decode`. All 32
runtime receipts validate, preserve the parent's original bytes and every
unrelated parent dimension, and bind the actual runtime observation digest.
Execution and reverse-observation axes now pass. The first unresolved dimension
moves from `execution_boundary` in the immutable generation receipts to
`permission_boundary` in the new runtime receipts. Unknown permissions are an
unobserved fact, not a human approval requirement. No aggregate completeness
percentage or full-domain claim is introduced.

The compiler's regressions separately exercise 3/3, 2/3 and observed 0/3 scores,
unexecuted cases, source forgery, parent mutation, selection overlap, failed
builds, cancellation, timeouts and oversized child output. A promoted
`bytes.Buffer.ReadFrom` bypass of the initial output bound was found by a real
subprocess regression and fixed. Initial CI also caught the old root-help
expected string missing `body-execute`; `ad0ebe0b` changes only that expectation,
after which the complete local CLI suite passes. Captured measurements retain
their original implementation SHA rather than being relabeled as that follow-up.

## Observed process costs

| Median measurement | Own model | Disconnected |
| --- | ---: | ---: |
| Code generation | 35.396 ms | 30.541 ms |
| Codegen process CPU, one core = 100% | 84.286% | 83.241% |
| Runtime observation pipeline | 687.674 ms | 584.486 ms |
| Go build child | 364.317 ms | 339.994 ms |
| One compiled-program process | 94.924 ms | 93.890 ms |
| Compiled-program peak RSS | 4.219 MiB | 4.227 MiB |

Codegen p95 is 581.543 / 34.711 ms; the first model process is cold. Even-sized
medians average the two middle samples; p95 uses nearest rank. There are 16
requests per mode, in fixed order, with two native runs per request. Compiled
process timings include process startup; they are not per-function latency.
Compiled-program RSS is not model-process or whole-pipeline memory. CPU reflects
process user/system time divided by elapsed time, not whole-host utilization.
These observations do not establish causal slowdown, speedup or parallel scaling.

The largest directly observed cost is now the build/process boundary. A future
performance study can compare repeated observation with source/tool-bound
executable reuse, while retaining per-request evidence and explicit cache
provenance. The current producer always performs its declared native build.

## Publication and scope

The [independent consumption report](../publication/native-runtime-consumption-20261002.json)
contains exact counters and resource distributions. The
[anonymous transport verification](../publication/native-runtime-hf-verification-20261002.json)
checks all eight public files against prepared local bytes and all 229 archive
members by size, SHA and CRC. Publication uses an explicit file list and checks
UTF-8, private home paths and credential patterns. Model weights and private
workspace files are excluded from this appendix.

This advances the typed-body runtime/reverse portion of compiler issue #1023.
Natural-language capability discovery, broader runtime/workflow integration,
before/after domain deltas and regression accounting remain open. This capture
does not constitute complete language or model development.
