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

## Observed results

The committed runner `a2b7c0c6960888d93f408f0f45af698c579d68cf` completed
144 SDK searches and 144 separate diagnoses, observing 576 typed candidates.
Search made 144 actual own-FP32 predictions; diagnosis made zero predictions.
Twelve distinct generated Go programs executed all 31 probes, giving 372 actual
function invocations. All values and 46 candidate-pair witnesses match the
independent state-transition oracle. Repeated witnesses are not independent tasks.

Each row below has 24 existing contract/language views. Ambiguity counts compare
output vectors to the selected body, rather than every pair in the candidate set.

| Arm / contract | All declared cases pass | Case-ambiguous views | Views with a distinguishing probe | Views with unresolved alternative |
|---|---:|---:|---:|---:|
| Disconnected / complete | 14 | 6 | 0 | 6 |
| Disconnected / sparse | 20 | 24 | 20 | 6 |
| Disconnected / contradictory | 0 | 6 | 0 | 6 |
| Own FP32 / complete | 24 | 4 | 0 | 4 |
| Own FP32 / sparse | 24 | 24 | 20 | 4 |
| Own FP32 / contradictory | 0 | 4 | 0 | 4 |

For sparse FP32, every view passes all declared tests but every view still has
a case-indistinguishable alternative. Twenty have an alternative with a differing
probe output. Four retain bounded probe agreement with an alternative. Neither
condition establishes the correct interpretation of the original natural language.
An observed witness supplies a useful additional input, with its expected result
still unset. It does not itself repair a program or improve functional accuracy.

Median additional diagnosis time is **0.089–0.093 ms** across these six groups.
Median SDK search time is **0.034–0.054 ms**. Both exclude model loading and plan
preparation, and are distinct from native request latency. Four more candidate
checks are extra work; this study supports opt-in diagnosis of uncertainty,
without evidence for an always-on latency improvement. Host CPU and process RSS
were not measured in this phase. There is no tuning, GPU, native or Laya call.

`runs/path-diagnosis-sdk-20261001` retains the complete captures, preexecution
pins and actual emitted-Go values. Offline audit reconstructs typed sources and
checks independent oracle outputs without new predictions or execution processes.
Captured model-ranking origin relies on the clean pinned runner/model and source
CI; the audit does not independently repeat those predictions.
