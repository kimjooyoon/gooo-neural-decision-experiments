# Retained legacy arithmetic replay contract — 2026-10-03

This follow-up repairs the continuing failure of the historical initial-audit
job. It adds no optimizer updates and changes no model, arithmetic kernel,
original expectation or numerical tolerance.

The original macOS and Linux archives already contain different legacy results.
A new Go reader will compare a fresh audit with the frozen archive for the
reader's actual platform: Go 1.27.1 darwin/arm64 or linux/amd64. Other platforms
will fail closed. Platform names supplied in an input report will not choose the
baseline. Both original publication manifests will have fixed byte hashes.

Acceptance requires all 36 journals and all 18,432 decoded rows in order, including
full text/source/target identities, float32 logits/probabilities, selected masks
and complete eight-candidate rankings. The 24 compact metadata/weight files must
match their frozen bytes. The report keeps the existing mass/NLL tolerance of
0.001 and export parity bound of 1e-5; all discrete summary fields stay exact.
Missing, extra, duplicate, malformed or changed observations fail.

The reader will also retain the original cross-platform comparison. Its complete
summary difference list must match the frozen platform-specific list, including
each path and both values, and every changed ranking will retain its input hash
and both orders. An observed legacy mismatch remains `MISMATCH` in the new record.
This is separate from the success of a same-platform historical replay.

The explicit `float32_separate_v1` job continues to require exact cross-platform
hidden/logit digests and all rankings/finite outcomes. That is the arithmetic
contract used by the subsequent compiler integration. The independent jobs both
remain required to pass the research workflow; old failed workflow runs remain.

Validation before publication: local frozen/archive replay, negative tests for
changed rows/ranks/targets/weights/reports, unsupported platforms, incomplete or
extra rows and bounded decoding; then a fresh macOS audit from clean source and
the complete Linux workflow. A diagnostic reader makes zero model predictions.
Fresh audits report their actual calls separately. Repeated known observations
do not increase the number of intentions or experiments.
