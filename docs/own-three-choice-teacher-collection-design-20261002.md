# Three-choice actual teacher capture and independent audit

This implementation follows the immutable three-choice preregistration. The
source corpus was published at `99a83c96c6c68ae629957773ef526fcc4089e99d` before
this collector or any three-choice teacher collection. Dataset SHA-256 is
`9a887dc09caf2f2b2b947641509328a2ee6f25dcefb6b52efe178fe8aff4fb3a`.
The corpus manifest, zero-inference audit and protocol retain their original pins.

## Operational calls and student inputs

The teacher is the frozen **independent** own v1 FP32 structural classifier,
256/48/8, loaded with `decision.LoadPath`. It is not the two-choice joint model,
Laya, a pretrained model or a source of student initialization. Two actual
seeded sessions per training view produce the prescribed 4,096 captures. Each
session starts with three independent predictions, advances one candidate at a
time, and uses up to seven explicit unfixed-coordinate reconsiderations.
Committed fixed coordinates and a sole remaining mask make zero predictions.
No CI hint or development/calibration teacher feedback is supplied.

The independent teacher's predicted individual feedback inputs remain verbatim
in its receipts. Separate full-text reconstructions also retain individual
zero-call declined inputs and their hashes without claiming another prediction.
Student contexts are a separate **derived observation**, constructed
from that receipt's actual first retained failure and all three selected labels
in the linked progress record. This matches `ReconsiderThree`'s complete ordered
selected-label prefix. Every original source header and natural-language intent
is retained through `FeedbackThreeWithParts`. The derived observation makes
zero student predictions; it must not be presented as a joint teacher response.

Every derivation retains full framed text, all three full parts, byte counts,
SHA-256 identities, representability and failure origins. Oversize inputs are
retained and excluded from optimizer rows; no truncation or coordinate dropping
is allowed. A sole remaining mask has no ranking input and contributes no
continuation row. Valid identical contexts are deduplicated per function view,
with every actual receipt origin retained. Calibration/development have only
their original initial rows. Eight-mask passing sets remain the targets.

## Audit and failure behavior

`internal/threecohort` revalidates all 3,072 ordered source rows, all three native
inputs, original source/document hashes and the independent eight-mask oracle.
`internal/threefeedback` reconstructs every candidate's ordered int64 values and
the frontier using captured probabilities, seeded initial sampling, scheduled
masks, committed-mask fixed coordinates and actual feedback proposals. It also
checks immutable progress/feedback chains and reconstructs the exact complete
student contexts. The audit makes no operational model calls and does not claim
to prove unobserved model probabilities or general natural-language accuracy.

The collector writes the actual result before auditing it. Cancellation, runtime
failure, representation decline and partial collection remain accountable.
Failures stop the phase and retain its prefix; there is no implicit restart or
fixture replacement. The 768 MiB cap includes existing `runs/own-three-*`
evidence, including the earlier failed native-export prefix. One MiB is reserved
for phase metadata; JSONL lines are bounded to one MiB. Publication files reject
host paths, credentials, symlinks, duplicate keys and unexpected schema fields.

An initial validation used the operation-model loader on a structural model;
the loader correctly rejected the schema before any prediction. It was corrected
to the existing structural loader; the frozen teacher bytes and ABI are unchanged.
The 16 actual native goal-7 rows selected by fixed family/configuration/language
rules supply 32 representative validation sessions, 300 actual predictions and
105 candidates per fresh unit-test run. These are implementation checks, separate
from the planned 4,096 recorded teacher sessions and eight authored study ways.

## Cost reporting and phase boundaries

Session median/p95 cover SDK ranking, search, evaluator and receipt hashing;
they exclude subsequent student-context derivation and immediate independent
audit. Whole collection CPU time also includes state serialization and audit,
normalized to one core. Process lifetime peak RSS includes source reconstruction
and retained data and is separate from neural tensor RAM. These are not isolated
kernel measurements or controlled host-utilization deltas.

Collection and offline audit do not constitute MPS training, student quality,
native compiled-Go execution, default promotion or public raw-byte verification.
All three fresh training arms, nine exports and their matched SDK/native studies
remain required by the original protocol.
