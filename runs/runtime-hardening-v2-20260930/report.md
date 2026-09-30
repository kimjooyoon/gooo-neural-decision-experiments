# Runtime hardening v2

Baseline v1 commit: `8a21ddef9918b033631b02a4abd088e657a2c9e6`. The original v1 audit, model files, dataset and saved parity vectors remain byte-for-byte unchanged.

## Fixes

- Validate raw JSON bytes as UTF-8 before `encoding/json` tokenization, so invalid bytes cannot be replaced before request or metadata validation.
- Reject Go keywords and the blank identifier `_` while preserving the ASCII-only identifier rule.
- Fail closed if finite model data overflows a hidden activation, output logit, or calibrated logit. Probability stages also reject nonfinite values. Failed predictions clear their output.

## Checks

- Go 1.27 targeted tests and `go vet` passed for `internal/decision` and `cmd/gooo-decision`.
- The compiled CLI rejected the saved raw invalid-UTF-8 request with exit code 2 and a JSON `status: rejected` response. Raw request and response bytes are saved beside the receipt.
- Independent Go audit passed all 96 saved parity rows at absolute tolerance `1e-4` and scored all 256 held-out rows for each model: FP32 256/256, PTQ ternary 245/256, QAT ternary 242/256. No cold CLI timing runs were made.
- Model metadata and weight hashes, dataset SHA and parity-vector SHA match the frozen v1 audit.

Machine-readable details and hashes are in `hardening-report.json` and `go-audit.json`.
