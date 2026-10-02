# Complete split and wording audit of the fixed full-input models

Registered before this audit's model calls. This completes the remaining
split/form observation in the [original full-input protocol](full-input-judgment-preregistration-20261003.md).
The [native integration observation](full-input-native-results-20261003.md)
already records generated programs and execution. This phase keeps the trained
weights, temperatures, source tasks, finite targets and default model fixed.
It performs zero optimizer updates and selects no deployment checkpoint.

## Inputs, models and comparisons

Reuse all 3,072 frozen language views: 2,048 training, 512 calibration and 512
previously observed development views. The dataset digest remains
`9a887dc09caf2f2b2b947641509328a2ee6f25dcefb6b52efe178fe8aff4fb3a`.
Reconstruct each original source-derived input and all eight finite targets in Go.

Evaluate all eleven forms named by the prior plans:

1. `original`.
2. The four training additions: `request-prefix`, `please-prefix`,
   `request-suffix`, `please-suffix`.
3. The two evaluation additions: `task-prefix`, `complete-suffix`.
4. The four additional wrapper controls: `bare`, `calibration_prefix`,
   `development_prefix`, `development_suffix`.

The additions use the original fullinputstudy.Apply definitions. Wrapper controls
use the exact authored-prefix transformation from the frozen bilingual wrapper
audit. They retain original and intervened full text with separate hashes and an
explicit intervention label. These controls describe research input changes;
production compilation continues to preserve caller text. Reject oversized forms
explicitly with zero prediction calls, full attempted bytes and the original ID.
Require equal representability and unchanged source-feature coordinates for V3/V4.

Use all twelve explicit-arithmetic models in both layouts from the existing
public bundle `publication/full-input-separate-arithmetic-20261003`, whose
manifest SHA-256 is
`ce4ad854c0d7fcb3e7fa049658f40ea4b0393a9bff39a2ef21f64d83f221bc3a`.
The model files are pinned to research commit
`600dc282fb059918c0fd6588d955d3a56c380b58` and HF commit
`7c4501789b34d885c14ab54aee1d40995eb8b1e6`. Require the declared feature contract
and `float32_separate_v1`; retain the original legacy-arithmetic observations in
their earlier appendices. No existing parity expectation is relaxed.

If every form is representable, collection has 33,792 view/form inputs,
405,504 paired prediction rows and **811,008 actual model invocations**.
Record actual calls, completed rows and declined forms independently. For each
pair compare complete feature, hidden, logit and probability bits and the selected
mask. Keep original full expanded predictions, both state digests and timing;
retain both predictions on any mismatch. Model identity lives in a source-bound
per-condition header rather than being copied into every row.

## Measurements

For every arm, precision, form, split, language and family record:

- First selected path's finite matches and full completion, with denominators.
- Static ranked completion at budgets 1, 2, 4 and 8, extra candidates and partial
  coverage. Exact equal-score ties use the existing lowest-mask ordering.
- Korean/English both-valid, different-but-valid, same-wrong and disagreement
  counts. Pair only matching original program groups and targets.
- Stable float64 passing-set NLL from logits and the frozen temperature.
  Record ten fixed confidence bins for selected-path validity, squared error and
  calibration summaries with their sample counts.
- Full feature collisions per representation and form, including intersections
  of valid-mask sets. Preserve every conflicting group and involved source/input
  identity. Keep the original operation-order alias as a separate control.
- Actual prediction time, process CPU/peak RSS, source preparation and audit wall
  time. Native generation metrics remain in the separate native observation.

All three splits have been used or observed previously. These results describe
known source tasks and explicit wording interventions. Report every condition and
regression. The model default and optimizer state remain unchanged regardless of
which condition has the largest completion count.

## Independent replay and bounds

Publish the collector, metric definitions and controlled synthetic tests before
collection. Freeze source/model/protocol identities. A separate reader reconstructs
original inputs, transforms, source coordinates and target sets; replays both
model layouts; recomputes every count and collision group; and compares all rows
and terminal summaries. A complete replay makes its own 811,008 calls if all forms
were accepted, reported separately from original collection.

Use the existing checkouts, model files and Go 1.27.1. Require 4 GiB free. Inventory
all retained own-three phases, counting both full-input and separate-arithmetic
phases toward the 768 MiB study increment and all own-three phases toward 3 GiB.
This audit permits 224 MiB raw evidence including a 16 MiB failure reserve,
and at most 64 MiB for its public archive/summary. The combined 288 MiB allowance
must fit the existing study cap before calls begin. Stream journals and bound
each row to 64 KiB. Reuse fixed prediction arrays; do not duplicate model weights.
One collection or replay has a 15-minute deadline and zero automatic retries.
On failure retain the exact prefix, actual calls and failed stage; a changed
budget or repaired continuation requires an explicit public record.

Publish the full inventory, all observations, negative results and resource scope
on GitHub and Hugging Face after privacy checks. This completes an existing model
audit obligation. Richer language constructions, natural-language capability
planning, order-sensitive representations and issue 1023's shared completeness
profiles remain the language development work that follows from the findings.
