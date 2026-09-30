# Persistent stream benchmark

`benchmark-stream` measures the built `gooo-decision-stream` executable as a
separate process. It does not compile the target binary. Supply the path and
required SHA-256 pin for that executable; the report contains the digest but
does not contain local file paths. The report is created with exclusive file
creation, so an existing report is never overwritten.

Run from the repository root:

```sh
mkdir -p audit
go run ./tools/benchmark-stream \
  --stream-bin ./bin/gooo-decision-stream \
  --binary-sha256 <64-character-binary-sha256> \
  --source-sha256 <64-character-stream-source-closure-sha256> \
  --output audit/new-stream-benchmark.json
```

The output parent directory must already exist and must not itself be a
symlink. The tool checks this before starting any benchmark process.

The optional source pin is checked against a deterministic hash of the fixed
runtime source closure: `go.mod`, the stream command, stream implementation,
and the three decision package source files. Its digest includes each
repository-relative filename, a zero byte, file contents, and a zero byte in
the listed order. The report also records the benchmark driver source hash.

The frozen 256-row test split is repeated 16 times per run, for 4,096 records.
The six sequential runs cover `fp32`, `ptq_ternary`, and `qat_ternary` at worker
counts 1 and 4. These repeats measure process throughput on the same finite
test set; they are not new prompts or independent examples. The harness parses
only the operand names and types from the synthetic prompt prefix. Gold labels,
split names, template IDs, and configuration IDs stay out of each selection
request and are used only when scoring returned results.

The report keeps these counts separate: planned and written inputs, observed
outputs, unique sequences, correlation matches, missing/duplicate/invalid
sequences, matched best labels, accepted typed IR, accepted-correct IR, type
abstentions, other abstentions, and errors. A best-label match can still occur
when confidence abstains; `accepted_correct` counts only emitted typed IR whose
operation matches the held-out label.

Each run has a 12-second process timeout and all runs share a 30-second limit.
Input is written incrementally and stdout is decoded as bounded NDJSON lines;
the harness does not buffer the full input or output. Every result sequence is
checked exactly once against its correlation ID.

Child user and system CPU seconds and peak RSS come from `getrusage` through the
child process state. Darwin's `ru_maxrss` is reported in bytes; Linux's value is
converted from KiB to bytes. Average core percent is
`(user + system CPU seconds) / wall seconds * 100`; this is process core use,
not host CPU percent, and it can exceed 100 when multiple cores run. No machine
or hardware identifiers or absolute local paths are written to the report.

This is an end-to-end child-process measurement: it includes process startup,
JSON decoding and encoding, queueing, and stream I/O. The separate `PredictInto`
hot-path benchmark measures model math with a reused workspace and excludes
those costs; compare the scopes as separate measurements.
