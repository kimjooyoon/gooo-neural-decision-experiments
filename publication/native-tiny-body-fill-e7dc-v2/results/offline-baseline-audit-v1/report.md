# Offline baseline audit — revision 1

This separate audit adjudicates only the two saved `offline_baseline` captures. It uses compiler source commit `e7dc8198600f27fa3d6fc07d2a8fd8b6a4af3b27`, native CLI SHA-256 `5e7f75badf8079d15d971bc05f65c08516a9b79c4bb0539036ccb318c093f79c`, frozen run-plan SHA-256 `1c5a86b321eacf9549d03f4b7df2c94f8ea8171b7dbb04f384094895869c3a58`, and Go `go1.27.0 darwin/arm64`.

## Capture adjudication

Both original wrappers exited with validation errors after saving their CLI stdout, invocation receipts, and generated Go. The errors are exactly the two known offline completeness-validator mismatches:

- `offline baseline completeness scope is not deterministic and local`
- `completeness receipt counts tiny_go as a Laya decision`

The separate preflight rechecked the raw receipts and found no additional errors. The raw records show `laya_provider=deterministic`, a Laya decision observation of `UNKNOWN` with 0/1 observed, external network boundary PASS (1/1), provider execution accounting PASS (1/1), zero local model predictions, zero external provider calls with that count known, and no model metadata or weights provenance. This is an independent adjudication of the saved evidence; it does not rewrite the original wrapper statuses or raw files.

## Generated Go replay

Both saved outputs compiled and executed against the six frozen finite cases for their fixture. All twelve observed cases passed: 2/2 cells, 12/12 cases, zero failures, zero unknown cases within this two-cell audit. The corresponding six-case suites are the declared finite evaluation cases, not holdouts or full-domain guarantees.

| Cell | Build wall | Execution wall | Cases |
|---|---:|---:|---:|
| arithmetic offline baseline | 256.973 ms | 362.990 ms | 6/6 |
| boolean offline baseline | 146.914 ms | 636.589 ms | 6/6 |

These are single sequential observations. The audit did not sample CPU or RSS, and these timings are not a latency benchmark. The original six-tiny-cell replay remains separate and retains its fixed 48-case denominator with twelve offline cases unknown there; this two-row audit does not modify that report.

No native CLI, model, or provider calls occurred during this audit. Generated executables are local verification artifacts and are excluded from the text publication allowlist.

## Bound evidence

- Original capture summary SHA-256: `405f782ee8866be0bf6b2f4e342b2a6986783295d653436fe95a8bb566dd3e57`
- Original capture evidence manifest SHA-256: `69cb3f5c8db02437e84c4d5266370f46674eac52a2442ee7c02510fe8b8162b3`
- Offline audit source SHA-256: `0d047afdd95e753a3f64832cbe1d67a09f168fbced67a0530baf030b1173b414`
- Offline audit tests SHA-256: `ee40a90a220b79db83e9684c643e21687f40b18c2132e86552a103c1ca24acf2`
- Offline audit runner SHA-256: `a351a913a89f004a554aba2377ea362cd89ce52a0002211a93444a10af804297`
- Detailed machine-readable result: [`audit-report.json`](audit-report.json)
- Exact mismatch preflight: [`preflight-report.json`](preflight-report.json)
