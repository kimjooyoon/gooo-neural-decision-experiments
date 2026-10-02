# Own Gooo three-choice completeness study — frozen before collection

## Purpose and scope

Extend actual typed Gooo body construction from two binary choices/four masks
to three binary choices/eight masks. Investigate interactions among condition
orientation, nested branch layout, local references, assignment targets and
execution order. Source declarations and legal typed alternatives remain
authoritative; a model ranks complete masks before tests and code emission.

This is a new authored composition study. It does not reclassify parameters,
goals, languages, seeds, variants, native replays or CI runs as new authored
experiment ways. Finite full-budget exhaustion is not semantic completeness
for arbitrary Gooo or natural language. No pretrained/Laya/previous student
weights initialize the new models. Human reviews and Guardian are absent.

## New closed ABI

- Model schema `gooo/tiny-three-choice-path-model/v1`.
- Feature version `triple_semantic_context_v3_joint_v1`.
- Exactly three complete canonical semantic-context-v3 UTF-8 inputs in declared
  choice order, length-prefixed under `gooo;joint3|`, maximum 1,600 bytes.
- Each part preserves the existing 512-byte contract and all 256 source/intent
  features. Concatenate three vectors scaled by `1/sqrt(3)`; no truncation.
- Dense architecture 768/24/8, ReLU, eight complete mask labels `mask_0` through
  `mask_7`; 18,656 parameters. Fixed request workspace is 3,200 bytes.
- FP32 weights: 74,624 bytes. Five-trit matrix packing plus FP32 biases:
  3,854 bytes. Decoded ternary matrix bytes 18,624, FP32 biases 128 bytes,
  and two FP32 matrix scales 8 additional bytes. These are tensor/storage
  quantities, separate from allocator and process RAM.
- Metadata cap 64 KiB; three-choice weights cap 128 KiB. Existing two-choice
  models retain their ABI, four labels and 64 KiB weights cap.
- Nil, disconnected, unsupported arity or representation overflow must reach
  deterministic paths with zero prediction calls. Record declines and full
  attempted byte/hash identity; never silently drop a choice or shorten intent.
- Valid warmed kernel calls allocate zero heap objects. Models are read-only;
  request workspaces and session state are caller-owned. Busy/canceled calls
  must retain committed progress and must not wait indefinitely on a mutex.

## Author the curriculum before model observation

Eight source families, each with three distinct declared editable targets:

1. Three chained operand orientations in local arithmetic.
2. Two successive assignment targets followed by a local return reference.
3. Conditional branch layout, one branch's assignment target and final reference.
4. Assignment target, later local reference and final subtraction orientation.
5. Root statement order, dependent assignment target and operand orientation.
6. Outer/inner branch layout and a final local reference.
7. Boolean comparison orientation, branch layout and branch assignment target.
8. Boolean local reference, conditional layout and final integer orientation.

Use configurations 0–23 and complete goal masks 0–7 in Korean and English:
3,072 function views / 1,536 bilingual groups. Each view has 16 ordered int64
cases, covering both sides and equality at authored conditional thresholds.
The corpus tool must bind actual main-generated source/semantic inputs and
enumerate all eight masks through an independent ordinary Go arithmetic oracle.
All alternatives in this authored cohort must type-check. Functional ties are
retained as complete passing-mask sets, not forced into a single label.

Train configurations 0–15: 2,048 views / 1,024 bilingual groups. Calibration
16–19: 512 views / 256 groups. Development 20–23: 512 views / 256 groups.
Shared authored wording limits generalization. Development is never used for
optimizer updates, epoch/temperature selection or default promotion.

Freeze concrete fixture source, cases, goal text, model inputs, dataset and
independent-oracle hashes after export and before teacher collection or training.
Source export uses the adopted compiler revision recorded in preexecution;
unsupported native export or type/source binding fails the phase and preserves
the prefix. Do not replace an inconvenient family after viewing model results.

## Actual own-teacher failures

Use the frozen v1 independent FP32 own model, pinned by metadata/weight SHA in
teacher preexecution. It supplies observations only, never student initialization.
For each training view perform two actual SDK sessions with explicit seed strings
`three-source-path-teacher-0` and `three-source-path-teacher-1`: 4,096 sessions.
Each session allows eight candidates, one per advance, up to seven actual-failure
reconsiderations using the existing unfixed-coordinate path. No CI hint.

Retain initial/feedback predictions, selected masks, ordered actual/expected
values, source/model/test pins, progress and feedback chains, canceled/declined
inputs and real failure text. Record actual counts; do not infer them from planned
calls. Construct unique representable three-input continuation states only from
observed training failures. Exclude representation declines from optimization,
with counts and complete input/hash evidence preserved. Do not append goal masks,
gold labels, future test results or synthetic perfect CI responses to inputs.

## Matched fresh students and fixed optimization budget

Three arms: uniform initial-target loss, passing-set initial loss, and passing-set
loss using initial plus actual teacher-failure states. All use the same freshly
initialized 768/24/8 network, seed 20261031, shuffle seed 20261032 and source data.
Archive the common random initialization SHA before optimization.

Each bilingual contract group has total weight one, split equally by language.
If valid continuation states exist, initial state weight is 0.5 and unique
continuations together weight 0.5; otherwise initial weight is one. Uniform
initial loss uses the normalized passing-mask targets. Passing-set loss is
`-log(sum(p[passing masks]))`. No marginal target overwrites joint equivalence.

For each arm: 100 FP32 epochs and 100 QAT epochs, 128 bilingual groups per batch,
eight batches per epoch: 800 updates per stage, 4,800 actual updates total.
PTQ adds zero updates. Each QAT arm starts from its own calibration-selected FP32
checkpoint. Python is limited to offline MPS optimization/export. Go owns source
collection, teacher/student inference, selection, codegen, audits and publication.

Pick epoch and temperature from {0.5, 1, 2, 4} by calibration passing-set NLL;
ties use earlier epoch, then lower temperature. Preserve all nine FP/PTQ/QAT
exports, parity data, loss curves, optimizer/update counts, memory and failures.
No cherry-picking seeds, replacing negative variants or restarting on timeout.

## SDK and actual native execution

SDK policies: all nine students, frozen v1 independent FP32 and disconnected
control, on both 512-view splits: 11,264 actual sessions. Budget eight candidates,
one per advance, seven feedback rounds, no CI hint. Preserve all policy/split
captures. Selector orders calibration extra candidates, actual predictions,
bilingual disagreement, packed bytes, candidate name. Offline is a control.

Native policy set: calibration-selected candidate, three own arm FP32 students
and disconnected control. Development configuration 20 gives 128 views per
policy: 640 actual Gooo generations, 640 independently compiled Go executions,
10,240 ordered actual invocations. Record exact deployed/feature source, SDK,
Go 1.27.1, binary and model hashes. A clean public feature source is labeled as
such until six-check native main promotion succeeds; never call it deployed main.

Report completeness at candidate budgets 1, 2, 4, 6, 8; ordered passing cases,
extra candidates, predictions, type rejections, remaining masks, declines and
paired-language agreement separately. Independently audit captured values and
failure text without rerunning operational predictions. Measure kernel, context
projection, model setup, codegen and compilation/execution separately. CPU is
normalized to one core; OS lifetime peak RSS is not model tensor RAM or a
causal change in host utilization. No overall speedup without a controlled pair.

## Disk, publication and phase boundaries

Retained raw experiment evidence is capped at 768 MiB; no individual JSONL line
exceeds 1 MiB. A cap failure preserves its completed prefix and stops dependent
learning; do not discard or truncate records to make the experiment pass.
Keep the active offline MPS environment and all frozen own-model baselines.
Use bounded Go streams and compressed public evidence; remove only verified
duplicate expanded publication copies after their audits complete.

Public GitHub/Hugging Face releases contain synthetic source, Go/Python offline
code, all declared models/results, PROV-O lineage and raw evidence. Scan text for
private host paths and credentials. Pin manifests and anonymously verify actual
uploaded bytes. Retain incomplete/negative results with their original source.
ABI implementation, collection, optimization, SDK behavior, actual native
execution, promotion and public byte verification are separate phase receipts;
none alone establishes full language or natural-language completeness.
