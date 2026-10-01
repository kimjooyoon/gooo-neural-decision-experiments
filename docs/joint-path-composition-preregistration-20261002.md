# Frozen joint-path Gooo composition study v1

Frozen before implementing this cohort, its independent oracle or model training.
The earlier source-v3 composition data remains diagnostic history. This study
uses six newly constructed compositions and new parameter/template rules. No
earlier, pretrained or Laya weights initialize either arm.

## Question, model and source ABI

Does judging two interacting choices together reduce the work to construct a
complete finite Gooo contract while retaining the full authored intent?

Compare an independent source-v3 256/48/8 model (12,728 parameters) with a new
joint 512/24/4 model (12,412 parameters). Both have 12,288 first-matrix weights;
head/hidden layout differs, so this is a bounded architecture comparison, not
an isolated loss-only experiment. The joint head ranks absolute legal masks 0–3.
Masks must cover exactly two two-option typed decisions in their declared order.

The joint input contains two complete canonical source-v3 inputs in order:
`gooo;joint2|<byte-length-0>:<input-0><byte-length-1>:<input-1>`. Each part retains
its original 512-byte UTF-8 bound and 64-byte typed-source projection. Lengths
are canonical decimal without leading zeros. The combined bound is 1,088 bytes;
no truncation is permitted. The joint 512 floats are the concatenation of the
two unchanged normalized 256-float v3 vectors, each scaled by 1/sqrt(2).
Initial inputs contain no expected outputs, gold masks, seeds, failed tests or CI.

The new ABI is opt-in and has its own closed metadata/schema. Its workspace is
2,160 bytes. FP32 weights occupy 49,648 bytes; five-trit matrices plus FP32 biases
occupy 2,590 disk bytes, decode to 12,496 tensor bytes plus eight scale bytes.
Five-trit storage is 1.6 bits per matrix weight; runtime arithmetic uses decoded
int8. Existing eight-label models/source-v3 inputs retain their ABI and results.
Original-source binding precedes native inference. Representation overflow skips
all predictions and retains deterministic continuation.

## Six new nonlinear two-choice compositions

All operations use signed int64 wrap behavior. Every typed node/local is used;
all four complete combinations must pass type/scope checks before collection.
Let a=x+d and b=x*s initially. Predicate p is x<=t for even configurations and
x==t for odd configurations. First/second bits describe the two listed choices.

1. `assignment_reference`: write a or b as a+b, then select a or b; return
   selected-x. Choices: assignment target, local reference.
2. `operand_assignment`: z=x*d+1; diff=a-b or b-a; write a or b as z+diff;
   return a+b+a*b. Choices: subtraction order, assignment target.
3. `branch_reference`: when p, a+=d and b*=s; otherwise a*=s and b+=d.
   Swap these arms or keep them; select a or b; return selected+x+a-b.
   Choices: branch layout, local reference.
4. `predicate_assignment`: use x<=t or t<=x (equality swaps equivalently in odd
   configurations); on true, write a or b as a*b+d; on false, b=a-b;
   return a+b+x. Choices: predicate operand order, assignment target.
5. `schedule_operand`: writes a=a+b and b=a*b, in either order; return a-b or
   b-a. Choices: schedule, subtraction order.
6. `schedule_branch`: standalone a=a+b before/after the conditional; the
   conditional writes b=a*b+d on true and b=a-b on false, with swappable arms;
   return a+b. Choices: schedule, branch layout.

Configurations 0–47 use d=((7*c+3) mod 19)-9, s=2+((5*c+1) mod 5),
t=((3*c+1) mod 17)-8. Source fallback mask=(3*c+1) mod 4. Display aliases vary
by configuration and carry no target labels. All four desired masks have full
Korean and English decision intentions. Reference/target descriptions use
first/second declaration roles; order/layout descriptions use source-relative
keep/swap actions. Never expose raw labels, masks, expected values or IDs in
natural text. Train/calibration/development template identifiers are disjoint;
shared core wording is reported, not treated as broad language generalization.

## Splits and full finite target

Configurations 0–31 train, 32–39 calibration, 40–47 development. All languages,
goals and feature/head views of a configuration stay together. Counts: 1,152
program/contract groups, 2,304 bilingual function views and 4,608 independent
decision views. These are six compositions plus parameter/goal/language variants,
not that many independent intentions. Train has 768 bilingual function pairs,
calibration 192 pairs, development 192 pairs.

An ordinary Go arithmetic/state oracle does not call the typed interpreter,
generated source or model. Retain these 16 ordered inputs, including duplicates:
MinInt64, MinInt64+1, MaxInt64-1, MaxInt64, -23, -9, -1, 0, 1, 9, 23,
t-1, t, t+1, d, -d. Compare all four typed combinations against all 16 cases.
Preserve every full-equivalent mask uniformly. Independent targets use exact
coordinate marginals; joint targets use the complete four-mask distribution.
Report product-of-marginals mass outside the full set and all input conflicts.

Collect released native source-bound v3 inputs for all 2,304 function views:
2,304 zero-prediction context exports, 4,608 individual inputs and 2,304 joint
inputs. Retain exact bytes/source/document/test/feature hashes and native/SDK
pins. Any incomplete input or oracle/source mismatch rejects collection before
training; do not drop groups or alter templates after observations.

## Offline optimization and selection

Both architectures use own random seed 20261025 and shuffle seed 20261026.
Architecture-specific random initial states differ and must be recorded. Train
20 FP32 epochs and 20 QAT epochs, AdamW lr=.001, decay=.01. Each batch has 128
bilingual function pairs: independent predicts both coordinates/languages, joint
predicts both languages. Use mean finite-target NLL only; consistency weight=0.
Six batches per epoch make 120 FP32 and 120 QAT updates per arm, 480 total.
PTQ has zero optimizer updates. QAT starts from its arm's calibration-selected
FP32 checkpoint. Temperature uses the fixed .5/1/2/4 calibration NLL grid.
Record MPS time/allocation/driver memory and process lifetime RSS. Do not invent
GPU utilization if unavailable.

Keep all six FP32/PTQ/QAT exports. Candidate selection uses calibration only:
fewer extra attempts to a complete contract, fewer actual predictions, fewer
bilingual initial-mask disagreements, smaller packed bytes, then arm/variant
names. Freeze the selector before development session judgments. Development
cannot replace a checkpoint, architecture, feature, template or hyperparameter.
No default model promotion is automatic.

## Continuation, native dogfood and public evidence

Evaluate all six exports and disconnected control on all 384 calibration and
384 development function views, all 16 cases, step=1 and total budget=4.
Independent ranking uses unchanged `ReconsiderUnfixed`; joint ranking conditions
one four-mask distribution on actual committed failures, preserving source
headers. Use a pinned compiler CI hint as context, not authority. One remaining
mask makes zero further calls. Optional input overflow preserves deterministic
continuation. Record all calls, partial cases, type rejections, canceled work,
receipt hashes, language choices and completeness curves at budgets 1–4.

Joint seeded sampling may choose the initial legal mask reproducibly from the
full four-mask probabilities; seeds never enter model text. Default ranking is
stable argmax with mask-order tie break. Native sessions retain immutable plans,
bounded frontiers and nonblocking busy/cancellation receipts. No human/Guardian
review belongs in this loop.

Fixed native dogfood: six families, development config 40, all four goals and
both languages: 48 views. Policies are calibration-selected candidate,
independent FP32, joint FP32 and disconnected: 192 actual native calls. Duplicate
selected/reference policies remain counted as replays. Independently compile
and execute every emitted function over all 16 ordered inputs: 192 executions
and 3,072 function invocations. Native inputs and candidate observations must
match the frozen Go sessions. Record child timing/CPU/RSS separately from warm
model latency/allocations and host utilization.

Go owns runtime, context collection, targets, native orchestration, audit and
publication; Python is offline GPU optimization/export only. Publish fixed
allowlists, all variants/negative results, source hashes, PROV-O derivations and
anonymous immutable byte verification. Preserve old bundles and raw rejections.
Four candidates being enumerable proves finite construction only, not universal
semantic completeness or unrestricted natural-language code generation.
