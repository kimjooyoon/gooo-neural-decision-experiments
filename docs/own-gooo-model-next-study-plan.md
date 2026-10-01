# Next studies for our independent Gooo model

The first compiler-context model is an independently initialized tiny MLP, now
published with explicit Go compiler input and runtime contracts. Do not promote
this checkpoint automatically: matched caller-context FP32 needs 68 additional
candidates vs compiler-context FP32 80 on the same reused development cohort.

## Evidence-led priorities

The source-bound read-only diagnostic identifies the main compiler-context FP32
cost: Korean root ordering needs 31 extra candidates across 32 views; Korean
branch layout needs 16/32, English operand order 16/32 and English local reference
15/32. Assignment targets are initially correct in both languages (32/32 each).
These are observed strata, not a causal diagnosis or new experiments. Independent
finite arithmetic was rechecked without new model/native calls.

## Three bounded matched comparisons

1. **Paired judgment objective.** Keep parameters, source context, partitions,
   optimizer updates and random initialization fixed. Compare zero versus a small
   preregistered bilingual consistency penalty on accepted-option distributions.
   Preserve all finite target ties. Choose checkpoints using calibration only;
   report extra candidate cost and disagreement for every family/language.
   The existing development set is already inspected; call it a reused diagnostic
   cohort. Register fresh program/templates before claiming unseen generalization.
2. **Candidate effects from Gooo.** Compare one-hop source text with explicit
   compiler-derived def/use and branch-effect features. Names and statement
   indices alone do not describe variable definitions or branch computations.
   Keep the source fallback basis and legal alternatives, with exact provenance.
   If adding facts exceeds 512 bytes, record representation decline or introduce
   an explicitly versioned feature ABI; never silently truncate Korean text or
   reinterpret old weights. Run matched size/latency/resident-memory controls.
3. **Per-candidate scoring.** Score each legal alternative with a shared tiny
   option scorer rather than a global fixed-label classifier. Keep candidates
   in bounded fixed arrays and expose optional logits to existing finite TDD.
   Prototype as a separate schema/ABI, with random-init data and Go numerical
   parity, before any native default adoption. Decision dependencies and observed
   partial-test/CI context belong in explicit later feedback stages, never in the
   initial input's authored answers.

## Operational acceptance

Primary measurements: finite completeness across bounded alternatives, added
candidate attempts, extra model calls, representation declines, source/semantic
binding, Korean/English disagreement, latency distribution and resident/packed
memory. First-choice intention agreement remains a separate observation.
Ternary QAT/PTQ must be measured against FP32 and disconnected execution, with
all regressions published. Packed payload (five trits/byte) and decoded runtime
memory are separate. Process CPU time is not host utilization.

Training stays a bounded offline GPU job; compiler export, model inference,
evidence reconstruction and publication verification remain Go. Only synthetic
or deliberately public Gooo inputs enter public artifacts. Failed collections
are retained and excluded by explicit receipts. Never infer unrestricted source
writing, general semantic correctness, or a default model promotion from finite
case coverage.
