# Direct typed source features v3: compiler/SDK contract

Frozen before implementation or new-model training. This phase implements a
source-bound structural representation, not a new natural-language generator.
Feature ABI `semantic_context_intent_v3` remains 256 input / 48 hidden / 8 output
labels / 12728 parameters, 1248-byte workspace. Old feature versions keep their
exact semantics. The proposed ten-output head is a separate subsequent study.

## Source feature projection

`PreparedPlan.SourceFeatures(decisionID)` reads its immutable validated snapshot,
applies declared fallbacks to a local fixed arena and marks statically reachable
source statements/expressions. It reads no intention text, tests, expected or
observed outputs, CI, model, seeds or selected model proposals. Native use follows
original Gooo/source equivalence binding. The SDK alone does not bind a source
file. The projection is a lossy structural view, not a proof of intended meaning.

64 bytes, values 0..128. Boolean features are 0/128; counts saturate at 16 and
multiply by 8. Literal magnitudes, spellings of names, stable IDs and target
answers are absent. Renaming unique locals or changing integer constants does
not change these features; source binding and finite actuals remain separate.

| Channels | Meaning |
|---|---|
| 0..4 | local reference / assignment / operand / branch / root choice kind |
| 5..6 | source result Int / Bool |
| 7..12 | target expression input / int / bool / local / binary / hole |
| 13..20 | source operator add / subtract / multiply / less_than / less_equal / equal / and / or |
| 21..24 | target statement let / assign / if / return |
| 25..32 | direct left/right child input, local, int, bool |
| 33..34 | source operator commutative / order-sensitive |
| 35..43 | reachable expressions, statements, roots, definitions, assignments, branches, local reads, binary child edges, assignment/read dependencies |
| 44..53 | option zero facts, as defined below |
| 54..63 | option one facts, as defined below |

Reference/assignment option facts: equals source target name, unique declaration,
first declaration, last declaration, declaration depends on input, declaration
contains multiplication, source assignment exists, reachable read count, reads
in source return expressions, belongs to source fallback option.

Operand facts: reverses source, left/right input, left/right local, left/right
integer, left operation add/multiply, preserves source. Branch facts: reverses
source, then add/multiply/subtract, else add/multiply/subtract, predicate input
on left, predicate equality, preserves source. Arm effect is its first
reachable arithmetic assignment/return/definition, not a complete effect system.

Root facts: preserves source order, first assignment targets first/last declared
local, first assignment reads first/last local, second assignment targets
first/last local, second assignment reads first/last local, inter-assignment
dependency. These facts are source observations, not an intended ordering rule.

Reject ambiguous duplicate local declarations, input shadowing or an unreachable
target explicitly. Do not substitute false scope facts. Native optional ranking
declines atomically to deterministic continuation; legal compiler paths and
finite TDD remain available. Prepared/input caller mutation cannot affect facts.

## Codec and feedback

Canonical text: `gooo;sem64=` + 128 lowercase hex digits + `;intent: ` + complete
natural suffix. Validate one choice-kind and one result-kind flag and byte bounds.
Reject invalid UTF-8, malformed/noncanonical hex, empty natural suffix, unknown
version or more than 512 UTF-8 bytes. No truncation. Direct 64 structural values
and unchanged 192 positioned natural-byte ngrams normalize independently before
the fixed presence scale. Valid feature encoding allocates zero heap objects.
Invalid input leaves feature/prediction outputs unchanged.

After actual partial tests, feedback retains the exact source feature header and
appends observed feedback to the natural channel. Initial inputs never contain
outcomes. Entire oversized feedback declines before predictions/frontier edits;
hash and byte count identify the complete attempted input. One remaining path
still skips unnecessary ranking. Confidence remains an observation, not a veto.

## Native evidence and publication

Native export explicitly selects `--feature-version semantic_context_intent_v3`;
default remains split v2. V3 export and ranking must produce identical byte/hash
inputs, source-feature hashes and normalized source basis. Record representation
declines, source/input/model hashes and actual prediction/finite-case counts.
Projection has no model calls, candidate tests, selected emission or writes;
source validation projections are generated/typechecked as before.

Tests cover all five kinds, reversed fallbacks, literal/name invariance, source
mutation rejection, outcome/seed independence, fixed bounds, concurrency, direct
codec zero allocations and source-preserving bounded feedback. Publish research
source before SDK extraction; pin every relocated source/test byte in a new SDK
manifest, preserve the prior manifest/tag, and run local/CI unit/race checks.
Native feature -> dev -> exact-tree main promotion uses the existing six machine
checks and independently verified proof. No human review/Guardian is added.

New weights need a separate preregistered fresh composition cohort and matched
training, calibration-only selection and native dogfood. ABI implementation and
deterministic contract-weight fixtures are not learned-model improvement or new
independent intention evidence. Runtime/orchestration stay Go; Python remains
offline GPU optimization/export only. Five trits per byte remain 1.6 disk bits,
decoded int8 resident matrices, not whole-process 1.58-bit RAM.
