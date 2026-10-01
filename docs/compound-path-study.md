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
