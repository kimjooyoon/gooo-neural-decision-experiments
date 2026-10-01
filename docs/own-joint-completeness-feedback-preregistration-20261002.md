# Own joint completeness and observed-feedback experiment v2

This document is frozen before new continuation collection or training. The
existing source corpus is already observed; this is not a blind new-data claim.

## Reason for the experiment

The v1 source representation has eight joint input groups with unequal uniform
finite target distributions. Independent Go reconstruction finds a common valid
full mask in every one of them, and no such variation in development. The 100
unequal independent-coordinate target groups also have nonempty common marginal
support, which does not itself prove full-function compatibility. These are
target-distribution differences, not established impossible intent judgments.
The frozen typed encoder remains unchanged. Adding source literals solely to
force agreement with arbitrary tie probabilities is not an established remedy.

v1 optimizes cross entropy against a uniform distribution over equivalent
passing masks. Executable completeness instead requires selecting any passing
mask. This study separates that objective change from learning actual feedback.

## Matched fresh own model arms

1. `uniform-initial`: uniform passing-mask cross entropy on initial inputs.
2. `set-initial`: negative log total probability of any passing mask, on the
   identical initial inputs.
3. `set-feedback`: the same passing-set loss with initial and observed failure
   inputs. Within each function/language, weight initial input 0.5 and the
   equally weighted unique continuation inputs 0.5 when continuations exist;
   otherwise weight the initial input 1.0. Functions do not receive more weight
   merely because a teacher took more attempts.

All arms use the existing 512/24/4 joint ABI, 12,412 parameters, source-v3 typed
facts, full natural intention, and the existing four legal masks. All start
from the same freshly initialized own state, seed 20261027. No pretrained,
Laya, v1 or earlier own weights initialize these students. QAT starts each
student's own calibration-selected FP32 checkpoint. Use shuffle seed 20261028,
AdamW learning rate 0.001 and weight decay 0.01, batches of 128 bilingual
function groups, 100 FP32 and 100 QAT epochs: 600 updates per stage, 3,600 actual
optimizer updates in total. PTQ performs zero updates. Export FP32, PTQ ternary
and QAT ternary for all three arms, retaining all nine exports and regressions.

Offline optimization/export may use the existing local MPS Python environment.
Collection, runtime, tests, selection, audits and packaging use Go. Record loop
time, process peak RSS and sampled MPS allocated/driver bytes. Do not invent GPU
utilization or a causal host CPU increase. Packed ternary bytes are separate
from decoded runtime tensors, workspace and process RAM.

## Data and teacher provenance

Use exactly the immutable v1 source curriculum SHA256
`2d1c9888a648d590e77253666b6e5ec848d375e47a3c99daf29dbde8bcbc7383`.
Configurations 0–31 train, 32–39 calibrate and 40–47 develop. Preserve all 1,152
contract groups, 2,304 bilingual function views, original typed source binding,
16 ordered int64 cases, retained duplicates and equivalent passing targets.
Neither language, configuration, goal, seed nor replay is a new authored intent.

Only the 768 training bilingual function groups collect teacher continuations.
Use the already public v1 joint FP32 teacher, metadata SHA256
`bbad1ae3bd5fbee3e782be40ea75c656be7f33f554ea64a79f034dc74669d931`, weights
`e1576dd7e7f206e60d1523eff3bb0af2b33c969c7021a502133d980de4985cf6`.
This teacher ranks inputs; it supplies no student initialization weights.
Execute four retained SDK sessions per function/language using seeds
`own-feedback-teacher/v2/rotation-0` through `rotation-3`: 6,144 actual sessions.
Each session uses the real existing four-path search, one candidate per batch,
at most four candidates and three feedback rounds. Sampling need not cover
every first mask. Count every real teacher prediction and attempted case, not
only unique training rows. Preserve captured progress, feedback, case values,
source/model pins, hashes and durations in a streamed session archive.

Feedback comes only after actual evaluator failures, using the runtime's exact
complete feedback text. A sole remaining mask requires no prediction. Overflow
declines without truncation; such records remain evidence and do not become
fabricated valid feature rows. The first mismatch's observed/expected value is
permitted as actual TDD context. Initial inputs exclude expected outcomes,
desired masks, targets and test suites. Never append a gold class to model text.

Record no invented CI success. The teacher and evaluation use no CI hint; this
isolates failure feedback from repository-status context. The existing runtime
continues to support independently supplied CI hints in other studies.
Deduplicate valid continuation text within each function/language, retaining
the originating session/round receipts. Targets remain the independently
reconstructed full passing-mask support: an attempted failing mask cannot be
a full-contract passing mask. No calibration/development teacher continuation
is added to the training data. Shared authored wording across splits remains
an explicit limitation.

## Selection, evaluation and native dogfood

Select epoch checkpoint and temperature using calibration passing-set NLL for
all three arms; report uniform cross entropy separately. Temperature candidates
are 0.5, 1, 2 and 4. Never use development to choose epochs or deployment.
Before development sessions, freeze a calibration-only candidate selector:
extra candidate attempts, actual model predictions, bilingual disagreement,
packed bytes, then candidate name, all ascending. No implicit default promotion.

Run Go inference parity and per-call allocation/timing probes for all nine
models. Evaluate all nine, frozen v1 joint FP32, frozen v1 independent FP32 and
disconnected enumeration on 384 calibration and 384 development views per arm:
9,216 actual SDK sessions. Report completeness curves for budgets 1/2/3/4,
partial cases, extra candidate attempts, actual initial/feedback calls,
probability mass outside the full target, target-set NLL, bilingual disagreement,
context declines, memory and total time. An auditor adds no inference calls.

Run adopted compiler main `363a3d8aa365c35dd634c241248b444de0050973` and SDK
v0.2.12 on configuration 40, six families, four goals and both languages for
five policies: calibration selected, each new arm's FP32, and disconnected.
This is 240 actual generations and independently compiled Go executions,
3,840 ordered actual invocations. A selected/reference duplication is a replay,
not a new intent. Verify emitted bytes, source binding, every ordered value,
model pins, actual feedback chains, candidate paths and process metrics.

Publish the protocol, model and source pins, raw new evidence, controls,
regressions and PROV-O lineage to public GitHub and a new public HF model repo.
Use a fixed Go allowlist and bounded streamed archive verification. Preserve
v1 immutable publication and prior local work. Public text must contain no
credentials or private host paths. Treat publication as byte availability,
not quality evidence. No Guardian or human approval step is introduced.

## Limits and continuation

Finite 100% after four candidates also holds for enumeration. It is not proof
of general language understanding, arbitrary text generation, larger decision
counts or unlimited program correctness. This phase does not add loop syntax
or new compiler fragments. After the objective/feedback comparison, extend
variable arity, nested branches, loop bounds and independently authored intent
groups based on actual observed costs and completeness.
