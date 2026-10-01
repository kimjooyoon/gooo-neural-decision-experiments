# Gooo path ambiguity diagnosis

Passing every declared finite case can leave multiple bodies with different
behavior. Natural-language ranking helps select a body, while a separate
deterministic diagnosis records how much the available cases distinguish it.
This complements the existing completion vector without making first-shot
accuracy an acceptance condition.

## API

`prepared.Diagnose(ctx, selectedChoices, cases, probeInputs, maxCandidates)` uses
the same owned typed plan as search. It compiles the reference, then ascending
remaining masks, rechecking combined type/scope constraints. It compares actual
output vectors on the declared cases, including contradictory duplicate inputs.
Equal pass counts with different output vectors are distinguishable by the cases.

For each case-indistinguishable alternative, it reports the first supplied probe
whose output differs. The receipt contains input, reference output and alternative
output. Both outputs are observations. Neither becomes a test expectation or
permission to modify source. Alternatives matching every probe remain unresolved;
finite agreement does not prove all-input equivalence.

Limits are 64 observed candidates, 128 cases, 32 probes and the existing sixteen
binary decisions. A deadline is required. The reference is observed first even
when its mask lies outside the first 64 ascending masks. Unobserved combinations
are counted explicitly. Cancellation retains completed candidates and leaves an
interrupted candidate unobserved. Model predictions are zero. No session or source
is modified. Diagnosis can be called concurrently on immutable prepared state.

Reference values use fixed `[128]int64` and `[32]int64` arrays, 1,280 bytes in
total. Only the reference and current candidate programs are needed during
comparison; the implementation retains bounded receipts rather than every body
or an all-pairs matrix. These are storage descriptions, not process RSS claims.

## Controlled study

The [preregistration](path-diagnosis-preregistration.md) fixes the existing
72-view cohort, two-candidate disconnected/own-FP32 search, four-candidate
diagnosis and 31 generic probes. The original intention mask and separate-input
expectations are evaluator-only. Actual emitted Go executes all probes, and an
independent integer-state oracle checks the captured vectors and witnesses.
Timing separates search from additional diagnosis work; model loading and plan
preparation are excluded. Probe inputs are supplied after selection.

Source tests cover sparse and contradictory cases, probe agreement remaining
unresolved, combined scope rejection, bounded partial enumeration including mask
65,535, concurrency/result ownership, cancellation and tampered evidence. The
maximum-mask fixture initially failed ordinary Go unused-local and expression-depth
checks; its fixed references and balanced expression tree were corrected before
the study. Those fixture failures were not model or measured runtime outcomes.

The SDK extraction/publication and actual study result are recorded separately.
This feature has not yet been integrated into the deployed native compiler.
