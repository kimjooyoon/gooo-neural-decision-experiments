# Native Gooo feedback judgment

The Go compiler now has an explicit optional path from partial finite-test
failures back to its original local structural model. The intended use is bounded
Korean/English judgment and continued body construction. The current models are
frozen and were not trained on these feedback contexts.

## Recorded integration study

- Native feature source: `7430d23583223d59def01d4cff772dbab7e8bdab`.
- Runner source: `c6943ec61f1163c4b36618716c12b3d0df135973`.
- Go 1.27.1; SDK `v0.2.3-experimental`; the three existing positioned-random
  FP32, PTQ ternary and QAT ternary bundles.
- One compound intention in Korean and English, complete and inconsistent
  seven-case contracts, 12 baseline/feedback model pairs, four deterministic
  controls. These 28 policy views are not 28 independent tasks.
- Batch size eight, 64 total candidates, at most two feedback rounds.
- Caller CI context comes from the previously verified main push at
  `307159f041644a3aa56dfd325c345f5325aec902`. It is recorded context without
  current-study, semantic-edit or merge authority.

All 12 model pairs emitted identical Go and had identical finite completeness.
Complete contracts reached 7/7; inconsistent contracts retained 6/7 while exhausting
the 64 candidates. The 28 views recorded 182/196 repeated finite passes. Actual
local predictions were **246**, including **102 feedback predictions** over 17
rounds; 1,080 candidate attempts were independently reinterpreted from captures.
No attempted path was repeated inside a session.

| Native process observation, median over 12 model views per mode | Rank once | Feedback |
| --- | ---: | ---: |
| Wall time, including process startup and JSON output | 8.815750 ms | 9.020834 ms |
| User + system CPU time | 8.650 ms | 8.857 ms |
| Child maximum resident memory | 20,865,024 bytes | 20,971,520 bytes |
| Summed observed model prediction time per view | 59.666 μs | 174.939 μs |

This is one baseline-first pass with no randomized ordering or repetitions.
The measurements do not establish a causal slowdown or general performance
advantage. Maximum RSS covers the whole native process and is not model-only RAM.
CPU time is process consumption; host utilization or a system-wide increase was
not sampled. Reconsideration incurred more inference work without measured
aggregate completeness gain in this fixture.

Two distinct emitted Go projections were compiled and executed separately, each
on the seven selection inputs and nine existing disjoint inputs. This means two
actual Go subprocesses, 32 actual function evaluations and 18 actual disjoint-input
evaluations, with nine distinct disjoint inputs. Reusing those executions verifies
252/252 repeated separate-input policy observations; it does not create 252
independent examples. Native AST, typed arena and actual Go values agree.

The rejected first study adapter omitted the outer plan schema and was rejected
before model loading. Its one failed native call and zero model predictions are
preserved separately and excluded from these primary totals. No new training,
GPU work, upstream Laya calls or external model calls occurred in this study.

## Development use

Use the planner to define Gooo types, stable IDs, finite choices and tests; use
the tiny model to rank bounded body references, assignment targets, branch layout
and scheduling choices. Test the selected body, retain partial completeness and
consider another unattempted path when the observed failure is useful context.
The model can abstain; finite source checks still decide what may be emitted.

The mechanism is suited to narrow Korean/English requests whose alternatives
are already representable in Gooo. The next useful experiment is a separate,
source-pinned feedback curriculum with counterexamples and partial completion
targets, measured on held-out intentions and paths. It should preserve the
original intent, report residual unconstructed behavior, and compare inference
cost against deterministic enumeration. This result is not evidence that the
current model can express arbitrary programs or reason over unrestricted IR.

## Evidence and reproduction

The full captures and executions are in
`runs/native-feedback-integration-fixed-20261001`; `preexecution.json` pins the
source, model and caller-context digests. `audit.json` records independently
reinterpreted candidate outcomes, progress/feedback hashes and actual Go parity.
CI reproduces the audit without new predictions, native calls or Go executions.

```sh
go run ./tools/native-feedback-study --mode audit \
  --native-root studies/native-feedback-v1 \
  --out runs/native-feedback-integration-fixed-20261001 \
  --audit-output /tmp/native-feedback-audit.json
```

This study measures the clean feature source. Merged-main checks and smoke
observations must be published separately; they cannot retroactively replace
the feature revision or measurements in this report. Existing HF weights and
earlier frozen studies remain unchanged when this evidence appendix is added.
