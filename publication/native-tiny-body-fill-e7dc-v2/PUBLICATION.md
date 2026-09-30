# Native tiny body-fill experiment, v2

This directory publishes the path-neutral text evidence listed in `manifest.json`. The original 149-file allowlist is copied as `publication-allowlist-v2.tsv`; its SHA-256 is recorded in the manifest. The allowlisted files were copied byte-for-byte. Executables, model weight bytes, and the preparation README containing a temporary host path are excluded.

## Run and source status

The frozen run used compiler source commit `e7dc8198600f27fa3d6fc07d2a8fd8b6a4af3b27` and CLI binary SHA-256 `5e7f75badf8079d15d971bc05f65c08516a9b79c4bb0539036ccb318c093f79c`. It ran eight CLI children once each: six tiny-model cells, followed by two offline baseline cells. All eight CLI children exited successfully. There were zero external provider calls. The two offline wrapper receipts retain their original validation failures; a separate audit determined that each had exactly the same two known completeness-validator errors, checked the raw no-provider evidence, and then replayed the saved generated Go.

The experimental source CI attempt reported failures in the `gofmt` and unit-test jobs. A later run was cancelled after a new head was submitted. This evidence does not claim a complete CI pass or main promotion.

## Results and report timing

The initial capture report is a checkpoint written before the later Go replay audits. It says the six tiny outputs were queued for replay and the two offline rows were still unknown in the fixed 48-case denominator. Those statements describe that report's checkpoint and remain preserved as written.

Later, the six tiny outputs independently compiled and passed 36/36 finite cases in `results/independent-replay-v2/`. The separate audit in `results/offline-baseline-audit-v1/` checked the two saved offline rows and compiled and passed their 12/12 finite cases. Together, the eight saved outputs passed 48/48 of these declared finite cases across the two later reports. The original six-cell 48-case replay remains unchanged at 36 observed, 0 failed, and 12 unknown; the later offline audit does not rewrite its denominator or status. These repeated finite suites are not holdouts and do not establish full-domain correctness.

The model's raw operation matched the declared gold operation in 0/6 tiny cells. The model's operation was applied in 2/6 cells; the two applied candidate choices scored 0/12 cases. Four fallback cells scored 21/24 for their deterministic fallback candidates. Fallback performance is reported separately from model predictions. After local TDD correction, the final selected candidates passed 36/36 tiny finite cases.

Across all eight CLI children, local model predictions numbered six and external calls numbered zero. Warmups and retries were zero. The capture report includes per-child wall, CPU, and RSS observations; it reports one sequential observation per cell and is not a latency benchmark. The separate generated-Go replay records one build and one execution per saved output, without CPU/RSS sampling.

## Build the standalone tools

A nested `go.mod` keeps these standalone experiment files outside the repository's parent Go module test traversal. The three tools each define their own `main`, so build them separately from this directory:

```sh
go build -o /tmp/native-tiny-capture ./capture.go
go build -o /tmp/native-tiny-replay ./replay.go
go build -o /tmp/native-tiny-offline-audit ./offline_baseline_audit.go
```

The focused source tests can also be run separately:

```sh
go test ./capture.go ./capture_test.go
go test ./replay.go ./replay_test.go
go test ./offline_baseline_audit.go ./offline_baseline_audit_test.go
```

These commands build the audit tooling; they do not rerun the native CLI or contact a model/provider. `manifest.json` binds every published file except itself by relative path, byte count, and SHA-256.
