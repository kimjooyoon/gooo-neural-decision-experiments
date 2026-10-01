# Fresh two-choice cohort implementation

The preregistration in `fresh-composition-semantic-v3-preregistration-20261002.md`
was committed at `d9fc4d1c8140c278794c6aabde3a7585af4692c0` before these fixtures,
oracle or tests were implemented. The implementation is in
`internal/compositionstudy`. It does not collect compiler inputs, load models,
train weights or report learned quality.

The six families combine references, assignments, operand order, conditional
arms and root scheduling. Two computed locals are declared before each write.
All four combinations use every declared typed arena node and compile. Display
aliases vary by configuration; source fallback is `config mod 4`. Numeric
parameters, predicate parity, split/config bounds, complete 16-input contracts
and source-relative actions follow the frozen preregistration.

`oracle.go` implements ordinary signed arithmetic/state independently of the
typed interpreter and generated code. Tests cross-check all 1152 program/goal
groups and 2304 bilingual function views, each with all four combinations and
all 16 ordered inputs. Those are repeated finite comparisons, not independent
intentions or compiler-native executions. Arithmetic overflow and duplicate
inputs remain observable. Sparse targets retain all complete finite masks and
their coordinate marginals, including equality-comparison ties.

Every decision's semantic v3 input fits the complete 512-byte bound in local
contract tests. Structural arrays remain identical when literal magnitudes,
display aliases, requested goals and languages change with otherwise identical
structure. No finite expected value is read by source projection. Natural
template identifiers are disjoint across train/calibration/development; the
templates deliberately share core action vocabulary, so this is not evidence
of unrestricted paraphrase generalization.

Next, the released native main must source-bind and export both feature arms
for every view. Collection must reject any complete-input overflow or source/
oracle discrepancy before training. No collection denominator may be reduced
after observations. Matched training, calibration selection, feedback cost and
144 native dogfood calls remain subsequent measured stages.

`tools/fresh-composition-curriculum` consumes clean released main build metadata
and independently verified public implementation evidence. It makes 4608 bounded
native export calls, captures their raw zero-prediction receipts and retains
9216 decision rows across both feature arms. Source/document/input/feature hashes,
finite target ties and complete natural suffixes are checked before rows enter
the collection. Each child has a 15-second deadline and bounded output. A failed
collection retains its actual attempted-call/row counts and cannot be used for
training. This tool is preparation until its fixed collection is actually run.

The first full 4608-call collection used embedded `json.RawMessage` responses,
which omitted the native encoder's final newline. The row hashes pinned complete
original response bytes, so a subsequent byte audit rejected this collection.
Its hashes/counters remain in `publication/fresh-composition-first-capture-rejection-20261002.json`.
No weights were trained and no rows/goals/templates were changed. Capture v2
stores the original byte slice as a base64 JSON value with exact round-trip
tests. A complete recollection is required; reconstructing an omitted newline
for diagnosis does not relabel the old capture as original-byte evidence.

The streaming audit reconstructs all independent finite targets and compares
every source/document/test/input/native-byte hash. It also reports byte-identical
model inputs with conflicting finite marginals, retaining those representation
limits in the study. Collection and audit digests stream through a fixed 32 KiB
buffer, so complete evidence files are not loaded into memory for hashing.
