---
language:
- en
- ko
license: mit
tags:
- gooo
- metaprogramming
- prov-o
- tiny-model
- ternary
- experimental
---

# Gooo typed path tiny v1

An independently trained bilingual model for ranking **compiler-owned Gooo
structural paths**. The workflow is natural-language judgment → typed path
ranking → finite tests → code assembly → recorded feedback. First proposal
accuracy, finite completion, remaining candidates, failed cases and cost are
reported separately.

Three training comparisons each include FP32, post-training ternary and
ternary-aware variants. No Laya weights are used. Two comparisons transfer the
hidden layer of our own public operator model; `positioned-random` is independently
initialized. Runtime, data generation, assembly and evaluation are Go. Python and
PyTorch are used only for offline MPS training.

## Closed contract and resources

Five edits: local reference, assignment target, operand order, `if` branch-body
layout and declared assignment execution order. Identifiers, body contents and
alternatives are compiler-owned; the model returns a fixed label. Every offered
alternative and the combined body are type/scope checked. The structural ABI is
`gooo/tiny-path-decision-model/v1`, separate from the existing operation ABI.

256 features → 48 hidden units → eight labels: 12,728 parameters. FP32 weights
occupy 50,912 bytes; ternary files 2,759 bytes. Five trits per byte use 1.6 bits
per matrix weight plus float biases/scales. Resident tensor arrays are 50,912
bytes (FP32) or 12,896 bytes (ternary), plus eight bytes of matrix scales. The
shared workspace is 1,248 bytes. Packed size is not process RAM or evidence of
packed-arithmetic speedup.

`uniform-transfer` uses uniform hashed byte n-grams. Positioned models use
`positioned_intent_ngrams_v1`: four coarse intent positions and lower context
weight. Uniform features collide for some reordered assignment sentences.
Positioned features separate that pair, but did **not** improve global label
results here. Transferring hidden weights across this changed feature layout is
an initialization experiment.

## Training observations

Public synthetic curriculum: 6,240 views (4,800 training, 480 calibration, 960
development evaluation). Templates and numeric configurations are partitioned;
plain, fallback Gooo body and PROV-O vocabulary views stay grouped. Evaluation
views represent 320 instructions and 160 program configurations. These 960 views
informed representation development and are **reused development evaluation**,
not an untouched final holdout.

| Comparison | FP32 global label /960 | PTQ /960 | QAT /960 |
|---|---:|---:|---:|
| Uniform transfer | 609 | 576 | 643 |
| Positioned transfer | 482 | 399 | 566 |
| Positioned random | 594 | 599 | 605 |

All nine bundles retain global confidence threshold 1.0. Conservative `Choose`
abstains on every recorded development view and uses the fallback. This is not
successful learned body generation. Go/Python parity is separately verified on
96 saved predictions per comparison. The preserved first MPS failure occurred
before any optimizer step: empty repair indices promoted a batch index to
float64. Corrected training uses integral indices.

Positioned-random 60-epoch FP32/QAT loops took 1.363/1.212 seconds. Sampled MPS
allocated peak was 5,717,248 bytes; Python lifetime peak RSS 478,281,728/482,066,432
bytes. These are short training observations, not Go inference requirements.

## Reserved bilingual TDD experiment

After freezing checkpoints, reserved configurations 64..95 and new English/Korean
templates produced 640 instructions, 320 program configurations and 1,920 views.
The fixed positioned-random model ranks two eligible labels. Search evaluates at
most two candidates using four authored cases; eight separate boundary inputs
are checked **after** selection. Expectations use independent arithmetic.

| Arm | Initial typed label /1920 | Evaluated candidates | Final unseen-input cases | Median prediction | Median typed search |
|---|---:|---:|---:|---:|---:|
| Deterministic order | 960 | 2,880 | 15,360/15,360 | No calls | 69.584 µs |
| FP32 ranking | 1,372 | 2,468 | 15,360/15,360 | 8.500 µs | 76.834 µs |
| PTQ ranking | 1,233 | 2,607 | 15,360/15,360 | 9.958 µs | 78.958 µs |
| QAT ranking | 1,300 | 2,540 | 15,360/15,360 | 10.041 µs | 79.333 µs |

FP32 saves 412 candidate evaluations (14.31%), while median search time increases.
This is useful prioritization on a bounded workload, not wall-time speedup.
Initial FP32 English/Korean choices are 671/960 and 701/960. All 1,920 global
confidence abstentions per model remain recorded. TDD acceptance is a separate
explicit mode. Conditional pair probabilities rank paths; they are **not**
calibrated probabilities of functional completeness. No checkpoint is selected
using this probe.

5,760 local predictions, zero external calls and zero native compiler calls.
Whole audit: 1.23 s wall, 0.95 s CPU, 26,345,472-byte lifetime peak RSS. CPU/wall
is approximately 77.24% of one core, not host utilization increase. Per-arm search
timings exclude holdout evaluation and serialization. All per-view receipts are
in the GitHub repository.

## Go usage

Use [gooo-neural-decision-experiments](https://github.com/kimjooyoon/gooo-neural-decision-experiments).

```text
go run ./cmd/gooo-path-compose --plan plan.json
go run ./cmd/gooo-path-compose --plan plan.json --model positioned-random/models/fp32/model.json
go run ./cmd/gooo-path-compose --plan plan.json --model positioned-random/models/fp32/model.json --tests tests.json --max-attempts 2
```

Tests use `gooo/typed-path-finite-tests/v1` and integer input/expected cases.
Without a model, assembly and TDD use deterministic declared order with no
inference/network calls. With TDD, one prediction per decision ranks candidates
before finite tests; there is no second model call after generation. Explicit
sampling binds plan/model/seed hashes. Deadlines, budgets, rejected types and
partial completion are recorded.

This is a Go structural assembly stage **before** native Gooo codegen. The native
compiler's `--tiny-model` still uses the operation ABI. Arbitrary body synthesis,
native in-process structural inference, OWL reasoning and production online
learning are not established. Finite synthetic cases do not prove general
natural-language intent completion. PROV-O views condition vocabulary; provenance
can support later learning without becoming an acceptance authority.

Publication copies a fixed synthetic-only allowlist and verifies digests.
Credentials, host paths, environments, private source and upstream large weights
are excluded. GitHub stores source, failed attempts and full raw evidence; this
HF bundle stores weights, public data and compact reports.

## Subsequent native compiler dogfood review

The reviewed publication retains exactly the same nine weight files. A separate
Go driver replays saved offline/FP32 TDD selections through the actual native
compiler at source `68361e64def5457f0d0e6de972570a9885cceb96`, using Go 1.27.1.
It checks compiler type checking, deterministic replay and compiler source ID,
then executes the **exact emitted Go bytes** in isolated packages.

Twenty native generations (five families × two languages × offline/FP32) pass
240/240 independent arithmetic cases. These executions repeat **five unique
functions**; the result is not twenty independent code synthesis tasks. Both
arms select the same final functions after finite TDD. Native replay makes zero
additional model predictions and zero external calls. Model ranking already
happened in the preceding 5,760-prediction probe; native execution does not call
the model a second time. Structural inference is still a Go stage before native
codegen, not the native compiler's operation provider.

The first replay incorrectly expanded the runner's short commit into a full
revision. Its raw metadata is preserved and excluded from primary source-bound
evidence. The driver now checks the declared full revision against its clean
tracked checkout before execution and captures its running binary digest. A
fresh run at `643ca6ead45cef35d85175864aa3b16556346bca` provides the primary
native review. Across both local attempts there were 40 native calls and 480
executed finite cases, with no new model calls. Only the correctly bound fresh
run supplies the 20-call/240-case primary review; the earlier correction record
is included here, and full captures are retained in GitHub.

This validates an end-to-end compiler path for the measured closed structures.
General intent completeness, arbitrary code generation and runtime speedup are
still unmeasured. Future work can use failure/type/provenance receipts to improve
bilingual ranking and reduce setup/candidate cost, while keeping explicit finite
completion and deterministic disconnection.

## Fresh native structural inference (2026-10-01 KST)

The subsequent direct integration uses native compiler source
`d21ce275ec0832e38ad4962ad03f4546c1b46e9f` and public Go SDK
`v0.2.0-experimental` at `e67cb5934d892fbc81be18038be2d624f1a6b5e0`.
The SDK's CI passed before tagging. Native implementation and fixtures are
public in [PR 1112](https://github.com/kimjooyoon/meta-ontology-go/pull/1112).
This supersedes the preceding review's limitation to replaying saved selections:
native `body-codegen --path-plan [--path-model]` now loads a structural bundle
and performs one in-process prediction per decision before finite tests.

The compiler first binds the declared fallback to the authoritative Gooo body
with its existing canonical equivalence witness. It validates all individually
offered options and combined scope/types, ranks finite alternatives, retains
partial results, replaces only an in-memory computes literal, and emits marked
Go with stable semantic identity, native type checking and deterministic replay.
It calls no external provider, even if Laya environment settings are configured.
Disconnection retains deterministic finite search from the declared fallback.

The fresh probe at runner source `b4d4ebe08335ab96427e2add3b012126d4687a3f`
uses one compound body with local reference, branch assignment/layout, and
declaration order: 8 combinations, 2 scope-invalid. Two languages × four arms
(offline/FP32/PTQ/QAT) × attempt budgets 4/8 × full/partial finite contracts give
**32 native calls and 72 fresh native model predictions**. These are views of
one compound experiment, not 32 distinct synthesis ideas. No weights are updated;
the nine published bundles retain their prior bytes.

All 32 exact native emitted Go functions independently execute 10 integer inputs
absent from the search cases, including int64 overflow edges. The 32 Go tests
pass **selected arena/native semantic parity**, while the separate arithmetic
intent observations pass **226/320 cases**. All budget-8 arms pass 160/160 of
those arithmetic observations. At budget 4, offline misses all 40; the models
pass all 60 English cases but only 6/60 Korean cases. Thus a small search budget
can still miss the intended assembly, even with identical initial selected
labels, because ranking weights change subsequent exploration. No claim of
unseen natural-language accuracy or full-domain proof is made.

The deliberately wrong partial contract requests 999 for input 3. With budget 8
the correct arithmetic function still yields a recorded **2/3 = 66.67%** on
that declared suite. Native lowering remains complete. Neither the incorrect
expectation nor the partial fraction is silently repaired or presented as 100%.

| Arm | Median prediction | Median native stages | Median lifetime native RSS |
| --- | ---: | ---: | ---: |
| Offline | no prediction | 1.571 ms | 18,350,080 bytes |
| FP32 | 8.375 us | 1.681 ms | 18,554,880 bytes |
| PTQ ternary | 9.646 us | 1.623 ms | 18,415,616 bytes |
| QAT ternary | 9.833 us | 1.658 ms | 18,415,616 bytes |

Process CPU-time/wall ratios have arm medians 81.49–82.89% of one core. These
are per-child measurements, **not host CPU utilization increases**. The native
stage durations exclude process startup; per-cell process wall time is retained
separately. This single fixed-order local probe has no matched repetitions and
does not establish a speedup. Inference uses no GPU. The packed ternary weights
remain 2,759 bytes; process RSS includes Go runtime, parsing, typing, arenas,
search and decoded tensors and is not the packed-weight size.

Initial fixture preparation rejected unsupported activity-ID syntax and then
caught a manually miscomputed arithmetic expectation before this source-bound
probe. The corrected explicit contract is `5*input+2`, minus 3 for negatives,
plus 3 otherwise. Native full local tests also report failures in unrelated
macOS namespace replacement, symlink diagnostics and source-splitter assertions;
no full local-suite pass is claimed. The new focused race tests passed. Actual
Linux CI results remain independently inspectable on the public PR.

The earlier ad-hoc mixed-language FP32 check made one additional native call
and three predictions; its 2.744 ms stage observation is excluded from the
32-cell primary probe. Publication contains only synthetic fixtures, public
model metadata/weights and selected evidence. No Laya weights, environment,
credentials, private repositories or user paths are included.

## Native integration is on main

[PR 1113](https://github.com/kimjooyoon/meta-ontology-go/pull/1113) merged
the direct structural model integration to native compiler main at
`9b8b900e32384b5397fb3d08dc545e22cc7530da` on 2026-09-30 22:51:48 UTC.
The protected promotion retained the exact dev tree with the previous main
parent. The six canonical checks and actual PR-bound promotion authorization
passed in [CI 36785977424](https://github.com/kimjooyoon/meta-ontology-go/actions/runs/36785977424).
The downloaded proof/append-only receipt independently passed the Go verifier
before the ordinary squash merge; no protection or required-review change was
made. Required approving reviews remain zero and Guardian is absent.

A clean main build with Go 1.27.1 then loaded the published FP32 structural
model directly, made three fresh local predictions, passed the fixture's three
explicit cases, type checking and replay, and recorded the actual main source
SHA. It made zero external calls. This main smoke is separate from the primary
32-cell probe and is not an additional independent arithmetic holdout study.
The main and dev trees were checked equal after merging. This records deployed
source availability; it does not claim a newly packaged native release binary.
Post-main push CI was queued when this note was written; the successful
authorization above comes from the actual pre-merge PR run.

Use native `body-codegen --path-plan [--path-model]`, documented in the
[Gooo compiler body-codegen guide](https://github.com/kimjooyoon/meta-ontology-go/blob/9b8b900e32384b5397fb3d08dc545e22cc7530da/docs/language/body-codegen.md).
For example, use `positioned-random/models/fp32/model.json` from this public
repository with the compound plan/fixture supplied by the compiler. Omitting
the model retains deterministic finite search. These are closed bilingual
structural choice models; they do not freely generate arbitrary program text.
All nine published weight bundles remain unchanged.
