# Own-model compiler-context training v1 (preregistered)

The native compiler is pinned to feature revision
`7d8768f8` (resolved full Git SHA in execution receipts), context schema
`gooo/compiler-typed-path-context/v2`, feature ABI
`split_context_intent_ngrams_v2`, Go 1.27.1 and SDK v0.2.9-experimental.
Compiler export and optional generation share source authority and input encoding.
The source binding generates validation projections, not selected candidates.

## Fixed data and matched training

Reuse frozen feedback dataset SHA
`570d1d73bfea32662b869e5cfbf77e6d04df838b703191b076be58c1d2180dbf`
and its 1040 Gooo bilingual pairs: 800 train, 80 calibration, 160 development test,
640 disjoint program groups across these partitions. These are reused synthetic
intentions, not 1040 new independent intentions or an untouched benchmark.

Go reconstructs each original typed fixture, binds its authoritative fallback
Gooo activity via 2080 actual native `body-context` calls (two language views per
pair), and retains raw receipts. Source fallback is forward for this fixed study;
reversed fallback contract regressions are native unit evidence, not this cohort.
Initial input excludes observed failures, expected values and CI outcomes.
Targets preserve all alternatives accepted by the authored sparse finite cases:
one accepted option gets mass 1, ties get 0.5/0.5. Go independently checks the
arithmetic oracle and each compiled typed candidate; intention labels are reported
separately from these finite soft targets. This study does not resolve sparse
observational ambiguity by relabeling it as natural-language correctness.

Two matched arms use the same random initialization (seed 20261012), 256 features,
48 hidden units, 8 path labels (12728 parameters), optimizer, minibatches and data.
`caller_context` uses the original Gooo-view prefix; `compiler_context` uses exact
native exported text. Both preserve the same full natural-language suffix and use
the split feature ABI. This is an input-context comparison, not a capacity change.
Each arm: FP32 20 epochs, QAT from its FP32 checkpoint 20 epochs; 560 optimizer
steps maximum total, MPS locally, no inherited Laya/pretrained weights. PTQ is an
export of FP32, not another training run. Checkpoint choice and temperature
(0.5/1/2/4) use calibration only. Development test never selects a checkpoint.
Python is offline optimization/export only; compilation, orchestration, inference,
finite evaluation and publication verification use Go.

## Primary metrics and stopping

Retain all FP32/PTQ/QAT artifacts, declines and regressions. Report natural-intent
label agreement separately, accepted-set coverage, completeness after at most one
additional typed candidate, added candidate cost, language disagreements, input
bounds, inference time and process CPU/RSS. Evaluate new weights in Go against
exact canonical exported inputs; caller-context-trained weights also receive
canonical inputs in runtime comparison. No automatic default model promotion.
Packed ternary payload uses five trits/byte (1.6 disk bits), decoded int8 runtime;
not packed arithmetic or a whole-process 1.58-bit RAM claim.

Execute actual native dogfood on a deterministic subset of 40 development views
(the first eight rows per family in frozen pair order). Four arms: compiler-context
FP32/PTQ/QAT plus disconnected deterministic fallback. Use full independent oracle
cases, max two candidate attempts, 160 actual native generation calls, plus actual
compiled-Go executions for the first view of each family per arm (20 executions).
Compare exported input hashes with generation receipts, finite actuals with the
oracle and generated-Go actuals. Record all process measurements; startup/compiler
cost is separate from model prediction latency. These are repeated policy views
of old intentions. CI repetitions are separate from local execution counts.

Source/protocol is committed before export and training. Fresh output directories,
clean Git/exact revisions, binary/build/source/model/data hashes, bounded child
processes and append-only captures are required. Any binding mismatch, overflow,
resource failure or evidence mismatch stops that execution with captures retained.
No hosted GPU job, online optimizer, service deployment or arbitrary code model
is claimed. Publication includes only synthetic public inputs and own artifacts.
