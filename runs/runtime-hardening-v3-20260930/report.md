# Runtime hardening v3

This follow-up makes `PredictInto` commit results atomically. It clears the caller output, calculates a candidate in a local `Prediction`, and copies it to the caller only when all inference and probability checks pass. If a later class fails, earlier logits remain local.

## Regression and validation

- Added a last-class calibrated-logit overflow case with earlier nonzero finite outputs. The call returns an error and leaves the caller `Prediction` zeroed.
- Go 1.27 tests and vet passed for the decision runtime and CLI.
- FP32 and QAT ternary hot-path benchmarks both remained at 0 B/op and 0 allocs/op.
- The independent Go audit passed 96/96 saved parity rows and evaluated all 256 held-out examples per model: FP32 256/256, PTQ ternary 245/256, QAT ternary 242/256.

## Preservation

V2 reports were not rewritten. Their exact runtime/test source is archived under `runs/runtime-hardening-v2-initial-20260930/` with hashes in `manifest.json`. V1 model weights, metadata, dataset, parity rows and audit remain unchanged.

Machine-readable details are in `hardening-report.json` and `go-audit.json`.
