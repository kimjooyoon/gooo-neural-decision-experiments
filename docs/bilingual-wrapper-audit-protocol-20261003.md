# Source-bound bilingual wrapper audit

This protocol is written before the new model calls. It investigates the shared
FP32 development observation: all 256 English views choose mask 0; Korean views
choose mask 7 in 191/256 cases. Both languages share authored functional targets.

Reuse the exact frozen 3,072-view source curriculum, all eight families and its
1,024/256/256 train/calibration/development groups. Independently reconstruct
source, typed alternatives and finite targets using `threecohort.Load`. Require
each EN/KO pair to have the same source and full finite target. This cohort has
already been observed. The audit measures input sensitivity on that cohort.

For every view, evaluate five complete input forms: original, bare instruction,
calibration prefix, development prefix, and development phrase as a suffix. The
two wrappers are the existing authored bilingual phrases. Remove only the exact
known curriculum wrapper when constructing a counterfactual; retain its body,
all three source headers and the original record. This is an experimental
intervention with fixed finite targets. No normalization is added to production.

Load the six published dense/shared FP32, PTQ and QAT models. Shared models use
the already verified compact representation. Make 92,160 actual Go predictions
(3,072 views × five forms × six models). Record each complete constructed input
once, its feature digest, and every prediction with its model identity. Recompute
feature arrays and require the 64 source coordinates in each of three parts to
remain unchanged. Count identical feature arrays with disjoint passing-mask sets;
such collisions describe an information boundary of this representation.

Report every split, language and family: first-choice finite completeness,
finite-case matches, static ranked attempts, complete-by-budget curves, mask
histograms, paired disagreement, both-valid agreement, different-valid choices
and same-wrong agreement. Retain per-view rows and all variants. Compare original
development totals against the already published model audit. Ranking uses one
prediction and the fixed complete finite targets; no adaptive feedback is run.

This audit performs zero optimizer updates, compiler calls, native program
executions, provider calls or model promotion. It does not establish unseen
language generalization. A subsequent training or feature change needs its own
published protocol and validation; improvement is not assumed from this audit.

The Go runner must be committed and clean before collection. Write a fresh output
directory, append bounded raw journals, retain any failed prefix, and cap new raw
evidence at 96 MiB. Reuse model and dataset files. Public packaging must exclude
private paths and credentials, include a closed byte manifest, and preserve all
observed conditions and regressions.
