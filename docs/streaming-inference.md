# Bounded Go decision stream

`cmd/gooo-decision-stream` is an NDJSON entry point over the existing closed
binary-operation decision model. It loads the model once, shares that immutable
model among a fixed worker pool, and keeps one reusable scratch workspace in
each worker. It does not add a provider, network, or arbitrary source-generation
path.

Run it with a model bundle and a worker count from one through eight:

```sh
go run ./cmd/gooo-decision-stream --model runs/pilot-mps-20260930-v1/models/qat_ternary/model.json --workers 4
```

Send one request object per line. A request uses the existing decision request
shape inside a stream envelope:

```json
{"schema":"gooo/tiny-ir-decision-stream-request/v1","correlation_id":"call-17","request":{"schema":"gooo/tiny-ir-decision-request/v1","text":"Add the quantity and fee.","left":{"name":"quantity","type":"Int"},"right":{"name":"fee","type":"Int"}}}
```

Each input record produces one JSON result line. The result includes the
monotonic input `sequence` and the caller's `correlation_id`; completed records
contain the same typed decision response as the one-shot command. Results are
written as workers finish, so callers must correlate by ID or sequence instead
of assuming completion order matches input order. Malformed records, duplicate
keys, unknown fields, oversized records, and model-level validation failures
produce a `rejected` result and do not discard later lines. An oversized record
is drained to its newline before reading continues; it may not have a usable
correlation ID, so the sequence still identifies its position.
The outer `status: "completed"` means the request was processed; inspect
`response.status` and `response.typed_binary_ir` to tell a selected IR from a
model abstention.

Each non-newline input record is capped at 16 KiB, matching the one-shot
request envelope cap. The model still limits the instruction text itself to
512 UTF-8 bytes. Queue and result channels each hold at most one record per
worker, and the producer blocks when the bounded queue is full. The program
uses a fixed number of worker goroutines, not one goroutine per request.
Cancellation closes both the input and output to interrupt pending reads and
writes. Callers should provide an `io.ReadCloser` and `io.WriteCloser` whose
`Close` methods unblock concurrent `Read` and `Write` calls. On normal EOF the
stream leaves both caller-owned objects open.

The model's hot `PredictInto` benchmark measures the fixed-array inference path
with a reused workspace and excludes JSON parsing, response encoding, channels,
and process startup. The stream tests exercise those protocol and scheduling
costs separately. Do not present hot-path nanoseconds or allocations as
end-to-end stream latency; benchmark the persistent command with representative
NDJSON input when comparing production throughput.

Run the focused stream tests and race detector with Go 1.27:

```sh
go test ./internal/decisionstream ./cmd/gooo-decision-stream
go test -race ./internal/decisionstream ./cmd/gooo-decision-stream
```
