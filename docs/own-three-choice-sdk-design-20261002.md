# Actual own three-choice SDK comparison

This implementation executes the SDK phase of the unchanged
[frozen protocol](own-three-choice-completeness-preregistration-20261002.md).
The nine own 768/24/8 students and v1 independent reference are already trained,
Go-kernel audited and anonymously verified on Hugging Face. No weights, training
target, authored family, split, seed, epoch or temperature changes in this phase.

## Execution and selection

`tools/own-three-sdk-study` loads every committed FP/PTQ/QAT export, verifies its
metadata/weight pins against the complete 4,800-update report, and loads the
frozen independent reference with its original pins. A nil model is the
disconnected control. All eleven policy identifiers have a fixed alphabetical
order written before collection. Model loading and pinning time is recorded
separately from SDK session time.

The full frozen actual-native-source corpus is revalidated before execution.
Each policy receives exactly 512 calibration views and 512 development views,
for 11,264 actual SDK sessions. Candidate budget is eight, advance size is one,
and feedback budget is seven. Every source, ordered case and legal typed
alternative remains unchanged. Empty seeds select the observed initial argmax;
no CI hint, target mask or future result is appended to model input.

Calibration finishes for every policy before the selector is saved. Selection
orders extra candidate attempts, actual prediction calls, bilingual initial mask
disagreement, packed model bytes and policy name. Offline is a control and cannot
be selected. Development is observed only after that selector has been saved.
No default compiler model is promoted by this SDK comparison.

## Independent evidence

Each complete JSONL capture records actual ordered candidate values, emitted
Gooo fragment hashes, initial distributions, full three-choice prediction texts,
every progress receipt and failure-conditioned reconsideration. Independent
reference calls additionally retain their complete individual attempted texts,
fixed coordinates and full derived three-choice contexts; those reconstructions
are not additional predictions.

`internal/threefeedback` audits source-bound emitted fragments and all 16 actual
int64 values for every candidate against independently authored ordinary Go
arithmetic. It reconstructs the eight-mask frontier from observed distributions,
including failed-mask exclusion, exact argmax/tie order, sole-remaining zero-call
steps and full-context zero-call representation declines. Hashes alone are not
treated as enough evidence: every progress counter, best case, selection,
feedback cause and complete source input must agree with reconstructed state.
The earlier training-only seeded teacher verifier retains its original contract.

The separate audit command strictly decodes retained captures and revalidates
their closed inventory, producer pins, prior raw evidence, group/language order,
calibration-only selector and all recomputed metrics. It makes zero operational
predictions. It can verify an incomplete retained prefix, clearly labeled as
incomplete rather than a full comparison.

## Resources and failure retention

The original 768 MiB whole-study raw cap includes the original source, teacher,
prepared features, optimizer receipts and this new SDK phase. Every earlier raw
file is pinned before collection. One JSONL line may not exceed 1 MiB. Two MiB
are reserved for terminal receipts; collection stops before another model session
if a full maximum-sized capture cannot be retained. No earlier captures are
discarded or shortened to make the study fit. A cap failure preserves its exact
completed prefix and prevents dependent native execution.

All policy cells report complete-function and best-passing-case curves at every
budget 1 through 8; prescribed budgets 1/2/4/6/8 correspond to indices 0/1/3/5/7.
Record actual predictions, feedback predictions, declines, extra candidates,
paired-language disagreement and full passing-set initial mass separately.
Median/p95 use the recorded 512 actual session intervals per cell. Recorded
kernel time includes context feature extraction. Collector wall also includes
independent auditing and storage, and is distinct from session wall. CPU is
normalized to one core. Process lifetime RSS is separate from tensor storage;
GPU utilization and causal host CPU changes are not claimed.

Representative validation comprises sixteen actual source rows covering all
eight authored families in both languages, with all eleven policies: 176 actual
sessions, 658 predictions and 634 candidates on the initial implementation.
Round-trip decoding and invalid source/input/case/model/frontier mutations are
checked. This validation is separate from the planned 11,264 sessions and adds
no new authored experiment ways.

Full SDK collection, actual native compilation/execution, CI replay, public raw
publication and compiler default promotion remain distinct evidence boundaries.
