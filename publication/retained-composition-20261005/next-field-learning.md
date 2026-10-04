# Next language experiment: field meaning in compact context

Status: planned after compiled-graph reuse; no new training performed here.

The current frozen model transfers three integer-choice ordinals into three
record-field choices. The next small curriculum will describe the field role,
allowed expressions and Korean/English assembly intent directly.

1. Declare record roles in Gooo: title copying, state replacement, reason suffixing,
   preserving an unchanged record and conditionally applying an update.
2. Vary field order, stable IDs, aliases, branch polarity and the placement of
   local assignments. Pair Korean/English intents with the same source semantics.
3. Keep runtime inputs and expected outputs outside model features. Supply actual
   failed-candidate feedback only after its bounded evaluation completes.
4. Split entire source families before fitting. Record separately the finite
   selection cases, held-out source shapes and independently compiled inputs.
5. Compare the frozen ordinal model, a small field-aware model and disconnected
   deterministic ordering at the same attempt budgets. Preserve the initial wrong
   candidate and each field's remaining value.
6. Measure complete named outputs, passed/declared record fields, time to first
   partial body, evaluated candidates, prediction time, resident tensor bytes,
   whole-process CPU/RSS and compiled-program reuse.

The experiment should produce Gooo source and its current executable behavior,
with a clear count of what the supplied examples actually establish. Training
cost and memory are recorded before expanding the field vocabulary.
