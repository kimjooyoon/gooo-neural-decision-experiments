# Fresh Gooo composition study: matched v2/v3 own small models

Frozen before this cohort's program/oracle implementation, collection or training.
This is a follow-on study to the implemented semantic source ABI. Existing
development cohorts are diagnostic history, not new independent evidence here.
No prior Laya or own-model weights initialize either arm.

## Question and fixed arms

Does a direct typed-source array reduce the work to assemble a complete finite
function contract compared with the previous compiler text/hashed context?
Compare `split_context_intent_ngrams_v2` and `semantic_context_intent_v3` using
the same 256/48/8 architecture, 12728 parameters, 1248-byte workspace and
eight structural labels. Native code must bind the original Gooo source first.
Source-feature compression may lose branch/order effects; a regression is a
result, not a reason to change the feature ABI after observing this cohort.

Each arm has independently initialized FP32, post-training ternary export and
ternary-aware optimization from its own FP32 checkpoint. Both arms use matched
initial seeds 20261022, shuffled batch seed 20261023, batch size 256, 20 FP32
epochs and 20 QAT epochs. Use the existing paired trainer's fixed optimizer and
finite-target loss; bilingual consistency weight is fixed at zero. With 3072
train decision views, this is 240 FP32 and 240 QAT updates per arm, 960 total.
PTQ exports have zero optimizer updates. Record offline MPS time and allocated/
driver memory samples; unavailable GPU utilization remains unavailable.

No model/default promotion is automatic. Final candidate choice is calibration
only, lexicographically: fewer extra candidate attempts to completeness, fewer
actual predictions, fewer bilingual initial-path disagreements, smaller packed
weight bytes, then the fixed arm/variant names. Development/native observations
cannot change checkpoints, hyperparameters, feature rules or the selector.

## Six new two-choice compositions

All arithmetic is signed int64 with Go wrap behavior. Every node/local must be
used and each offered combination must pass type/scope checks. Compiler-owned
typed fragments supply all code; the model only ranks legal alternatives.

1. Reference plus operand: two computed locals, a chosen local, and an
   order-sensitive subtraction inside a return that also reads both originals.
2. Assignment plus branch: two computed locals, conditional arithmetic writes,
   a choice of destination in one arm, and a joint return.
3. Predicate operand plus branch: a comparison with swappable operands,
   swappable arithmetic arms writing a used local, and a return.
4. Assignment plus schedule: two declared locals, two potentially dependent
   writes, one target choice and a choice of the writes' root order.
5. Reference plus schedule: dependent writes in two orders followed by a chosen
   reference; the return also reads the two original locals.
6. Branch plus schedule: a standalone write before/after a conditional write,
   swappable arithmetic arms and a joint return reading both locals.

Each family has 48 configurations. Fixed parameters are delta=(config mod 13)-6,
scale=2+(config mod 3), threshold=(config mod 11)-5. Fallback bitmask=config mod 4
sets the two source orientations. Unique display aliases may vary by config but
are not labels/answers. Predicate alternates less_equal and equal by parity;
only the predicate family uses a comparison operand choice. All four desired
choice masks get Korean and English natural views. Desired paths describe
source-relative actions and local roles, never raw model label names, pass
counts, expected values or IDs. Full natural suffixes are preserved.

There are 1152 program/contract groups, 2304 bilingual function views and 4608
decision views per feature arm (2304 bilingual decision pairs). These are six
composition families with four goals each and parameter/language variants;
they are not 2304 or 4608 independent intentions. Any finite equivalence ties
are recorded as ties, not invented unique answers.

## Split and independent functional oracle

Configurations 0..31 train, 32..39 calibration, 40..47 development. A config,
its two languages, both feature encodings and all four goals stay together.
Train/calibration/development use disjoint natural template identifiers fixed
before collection. Counts per arm:

| Split | Program groups | Function views | Decision views | Bilingual decision pairs |
|---|---:|---:|---:|---:|
| Train | 768 | 1536 | 3072 | 1536 |
| Calibration | 192 | 384 | 768 | 384 |
| Development | 192 | 384 | 768 | 384 |

The oracle implements each family's intended arithmetic/state changes in
ordinary Go; it does not call the typed interpreter, generated source or model.
Use these 16 inputs, retaining duplicates in the ordered finite contract:
int64 min, min+1, max-1, max, -17, -7, -1, 0, 1, 7, 17, threshold-1,
threshold, threshold+1, delta, -delta. Overflow is intentional and observable.
Independently prepare/evaluate all four typed combinations against the complete
16-case contract. Preserve all best/full masks as a sparse finite target with
coordinate marginals, including equality/zero-parameter ambiguity. An oracle
or source-binding mismatch rejects collection before training.

Run the released native source-bound context exporter for every function view
and both feature arms: 4608 export calls, 9216 captured decision inputs. Those
are zero-prediction/zero-candidate-test context exports. Preserve raw captures,
full source/document/test and feature hashes, byte bounds and compiler/SDK pins.
If any complete input exceeds 512 bytes, reject the fixed collection before
training; do not truncate, drop groups or modify templates after outcomes.

## Assembly and completeness measurements

Evaluate each of the six frozen model exports on calibration and development
using all 16 independent cases, step size one, total budget four. Compare the
offline deterministic baseline on identical contracts. Initial source inputs
exclude outcomes. Actual failed batches may use source-preserving bounded
`ReconsiderUnfixed`, conditioned only on committed observations and a pinned
compiler CI hint; record each actual prediction. A sole remaining path makes
zero extra predictions. Overflow preserves deterministic continuation.

Report initial passing case fraction and complete-function count separately,
minimum candidate budget for a full finite contract, extra attempts, partial
completeness curve at budgets 1..4, unresolved masks, actual predictions, type
rejections, bilingual initial paths, latency, allocations and packed/resident
weight bytes. Four-candidate completeness alone is enumerative evidence, not
general correctness or model understanding. Never replace a denominator with
only the model's initially selected cases. The own model does not learn online
from these feedback judgments.

Use the released native main source/SDK with build metadata pinned in collection
and dogfood manifests. Native dogfood is fixed to all four desired masks in
development config 40, both languages, all six families: 48 function views.
Compare calibration-selected model, v2 FP32 reference and disconnected baseline:
144 native calls. Independently compile and execute each selected emitted Go
function on its full 16 inputs. Preserve partial/type/cancellation/error receipts;
count only completed independent executions as functional evidence.

## Publication and bounds

Orchestration, collection, audits, independent arithmetic, native execution and
packaging remain Go. Python is offline GPU optimization/export only. Learn only
from deliberately public synthetic Gooo/Korean/English material; publish a fixed
allowlist, hashes and anonymous-download verification on GitHub/Hugging Face.
Exclude credentials, arbitrary environments, local paths, private logs and
caches. Record disk usage rather than duplicating model caches.

Keep source/metadata/weights/curriculum/CI hashes and calibration selection as
append-only evidence. Go/Python feature/logit/probability parity must be measured
on explicit held-out rows, not inferred from a shared implementation. Packed
five-trit bytes are 1.6 storage bits, decoded int8 matrices; no claim of packed
runtime arithmetic, whole-process 1.58-bit RAM or broad language understanding.
Any changed protocol, additional arm or tuning requires a separately frozen
subsequent study and cannot replace this study's negative observations.
