# Interacting Gooo body construction

This study uses the deployed native compiler with SDK v0.2.5. Korean and English
intentions select two existing binary structural decisions, producing four
whole-body paths. Models remain optional. The disconnected control evaluates
paths in its deterministic declared order. First-choice accuracy is an
observation; finite completion, remaining paths, failed cases and cost are the
development targets.

## Declared design

The synthetic configuration is 101 with `A=4`, `B=14`, `C=10`. Three newly
authored body templates expose interacting decisions:

| Template | First decision | Second decision | Body behavior |
|---|---|---|---|
| `reference_assignment` | Read first/second local | Assign first/second local | Read a local, add C, assign the update, return the difference |
| `operand_branch` | Input−B / B−input | Keep/swap if branches | Compute subtraction, apply multiplication or addition under input<A |
| `reference_schedule` | Read first/second local | First/second assignment first | Update first by C, update second from a selected local, return the difference |

Each template has four intended masks, two languages and three contracts:
seven sufficient selected inputs, one deliberately ambiguous selected input,
or the seven inputs plus a contradictory duplicate. The contradiction makes
7/8 the highest declared finite completion. The resulting 72 views contain
three fallback source bodies, twelve structural intention groups and 24
bilingual program instructions. They are repeated experimental views, not 72
independent ideas or an untouched natural-language benchmark.

The coefficient C was chosen as B−A before model execution so every sparse
contract has an integer ambiguity witness. `reference_schedule` masks 1 and 3
have commuting updates: both read second, whose update is independent of first.
These two structural paths are functionally equivalent on all int64 inputs by
the declared arithmetic transitions. Their structural intention labels remain
separate. Functional completion can therefore reach 100% with a different
execution order from the intention label.

## Measurement protocol

The matrix contains a disconnected control, the existing parent FP32 model and
three feedback-trained experimental checkpoints. Each model runs once with
initial ranking and once with observed-feedback ranking. That is nine arms and
648 actual native calls. A three-call disconnected pilot checks native source
binding before the matrix. Candidate steps are one attempt each, the total
budget is four and the feedback-round budget is three. When only one complete
path remains, SDK v0.2.5 emits a hashed zero-prediction receipt.

Source, model identity, failures, finite inputs and original intentions are
fixed. Caller CI status describes the source the caller claims was checked;
it grants no semantic authority. Each feedback receipt must link to its prior
progress, original model and observed first failed case. Every emitted selected
program is deduplicated by exact Go source, executed on the union of selected
and separate inputs, and compared to independent arithmetic state transitions.
Repeated policy observations are counted separately from actual Go processes
and function evaluations.

The source runner is committed before execution. Raw native output is retained
before interpretation. Each child has a bounded timeout/output and process-group
cancellation. Whole-process wall time, CPU time and peak RSS are descriptive
observations in a fixed arm order. No causal speedup or host CPU utilization
increase follows from this design. No training or upstream Laya HTTP call is
part of this study.

The authored cohort and independent typed/oracle regressions are available in
[compound-path-v1](../studies/compound-path-v1/) and
[compoundstudy](../internal/compoundstudy/).

## Observed main results

Committed runner `0127ed66e5fae44d134e07593bec683246285979` used clean native
main `ef63060ed1aebd9d92a2fe4cf24ce9c5929b8726` / SDK v0.2.5. The pilot performed
three disconnected native calls and three actual Go processes with 42 function
evaluations. The full matrix performed **648 native calls, 1,590 local model
predictions (438 after failures), 1,405 candidate attempts, twelve actual Go
processes and 192 actual function evaluations**. Repeated finite and separate
observations are 3,240/3,456 and 4,810/4,932 respectively; they reuse those Go
executions. All children completed within the declared timeout.

The [matrix](../runs/compound-path-main-20261001/report.json) and
[zero-inference audit](../runs/compound-path-main-20261001/audit.json) preserve
every raw capture, progress link and failed case. Across all 288 paired
ranking/feedback views, final Go, finite pass count and separate-input pass
count match. **Sixteen candidate sequences differ**. Twenty-nine individual
feedback judgments differ from their initial proposals. These two counts
describe different observations.

| Model | Candidate attempts, initial → feedback | Model predictions, initial → feedback | Final structural intention matches / 72 | Separate pass observations / 548 |
|---|---:|---:|---:|---:|
| Disconnected | 198 | 0 | 60 | 498 |
| Parent FP32 | 146 → 147 | 144 → 246 | 70 | 530 |
| New FP32 | 152 → 152 | 144 → 256 | 72 | 548 |
| New PTQ | 152 → 152 | 144 → 256 | 67 | 530 |
| New QAT | 154 → 152 | 144 → 256 | 72 | 548 |

Each arm reaches the declared finite best: 360/384, or 93.75% overall. Complete
and sparse contracts reach 100%; contradictory contracts reach their 7/8
ceiling. Structural matches are **after finite TDD**, not first-choice model
accuracy. The QAT feedback arm saves two attempts while making 112 extra
predictions. The parent feedback arm makes one extra attempt and 102 extra
predictions. Those costs remain visible alongside the favorable separate-input
results of new FP32/QAT on this small configuration. Earlier regressions on
[other development views](feedback-path-training-results.md) remain relevant;
this result does not justify a general checkpoint promotion.

### A measured remaining ranking cost

SDK v0.2.5 records 96 zero-prediction receipts when one complete path remains. The audit
also identifies **97 of 438 actual feedback predictions** for decision
coordinates that have the same value in every unattempted whole-body path.
Such probabilities cannot distinguish those remaining candidates by that
coordinate. This is a cost opportunity for a later active-coordinate ranking
experiment. The current study has not skipped those predictions or measured
the resulting behavior; proposal scheduling and numerical ties must be checked.

### Projection and resource scope

The initial AST audit rejected the SDK's `int64(4)` against the native
compiler's `4`. The repaired audit normalizes only in-range integer literal
wrappers, then compares the function signature and body. It does not erase
variable conversions, arbitrary calls, changed constants or extra declarations.
The repair made no new native/model/Go calls. All 648 selected native functions
now reconcile with the selected typed bodies, and every policy observation is
checked against reused actual-Go values and independent arithmetic transitions.

Across nine arms, whole compiler process wall medians are 5.592–5.789 ms,
CPU-time medians 4.839–5.054 ms, median peak RSS 17,211,392–17,514,496 bytes and
median CPU/wall ratios 86.66–87.37% of one core. The full records retain outliers.
This fixed-order local measurement does not show a causal wall-speedup or host
CPU utilization increase. No GPU training or upstream Laya call occurred.

## Public source and model evidence

The implementation and recorded experiments at
`c42c9e4c1061d4f47d3c99d63e5347249b397f80` passed all four research CI jobs in
[run 36821564541](https://github.com/kimjooyoon/gooo-neural-decision-experiments/actions/runs/36821564541).
The [source/publication receipt](../publication/compound-path-source-publication-20261001.json)
keeps that implementation check separate from the original measured runner,
actual main source and public byte verification.

The [immutable Hugging Face appendix](https://huggingface.co/asketeddy/gooo-feedback-path-tiny-v1/tree/c902d14550eda5de08ff7a5bbd8507e60f02c31f/research/compound-path-main-20261001)
contains nine verified payload files and a deterministic 1,729,483-byte archive
covering 674 raw allowlisted records. Ten anonymous requests verified the new
appendix. At the same revision, 26 anonymous requests reverified the 25-file
original model/core bundle and eight requests reverified the seven-file previous
main appendix. Credentials were absent from those requests. No new checkpoint
weights or original model-card edits were published in this iteration.
