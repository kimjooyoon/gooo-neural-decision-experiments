# Public Go SDK kernel benchmark

One bounded Darwin ARM64 run used the public
`gooo-decision-runtime@v0.1.0-experimental` module and three existing pilot
bundles, without local replacement or weight copies. Each model received 128
explicit warmups and 50,000 timed `PredictInto` calls on one fixed instruction.
The raw receipt pins SDK source, module checksum, runner source, and weights.

| Model | Median | p95 | Mean |
| --- | ---: | ---: | ---: |
| FP32 | 7.750 µs | 10.917 µs | 8.192 µs |
| PTQ ternary | 9.250 µs | 9.334 µs | 9.317 µs |
| QAT ternary | 9.208 µs | 9.292 µs | 9.257 µs |

These timings include per-call timers and latency-sample recording. They measure
the model kernel on one input, not end-to-end code generation, request routing,
TDD, cold loading, or accuracy. This is one run; no uncertainty interval is
claimed. The ternary matrices are decoded `int8`, not a native 1.58-bit kernel.

Each measured loop used about 100.02% of one CPU core: the process user/system
CPU delta divided by that loop's wall time, including measurement overhead.
This is not a host-utilization increase. `RUSAGE_SELF.ru_maxrss`, interpreted
using Darwin GETRUSAGE(2)'s byte units, reported 7,569,408 bytes (7.22 MiB) after
all three variants. This is the process-lifetime peak, not current RSS, a
per-model measure, or a before/after increase. No GPU work, training, or Laya
calls were performed.

Separate `AllocsPerRun(100)` diagnostics returned zero allocations for each
kernel. Go's diagnostic performs its own additional warmup; those calls are
excluded from the 50,000 timed-call count. `Model.Decide` and the native provider
may allocate response and validation data; they are outside this result.

An initial runner invocation stopped before loading models because its module
pin parser rejected valid single-line `require` syntax. The corrected runner
then completed once. See [attempts.json](attempts.json); the original failing
source snapshot was not retained, so its exact source hash is unavailable.
The successful receipt and source have not been overwritten.

## Reproduce in a fresh directory

Copy this folder, set `taskModelRoot` to the existing pilot's `models` directory,
and keep the recorded receipt intact:

```sh
go mod download
go mod verify
go run . \
  "$taskModelRoot/fp32/model.json" \
  "$taskModelRoot/ptq_ternary/model.json" \
  "$taskModelRoot/qat_ternary/model.json" \
  "$(go list -m -f '{{.Dir}}' github.com/kimjooyoon/gooo-decision-runtime)" \
  > benchmark-replay.json
```

Recorded `benchmark.json` SHA-256:
`ac2fb30202f7bfea4dc7df43fbfddef3a3d51453cf02b083d7a476887720c920`.
The subsequent SDK v0.1.1 metadata-provenance feature is not part of this frozen
benchmark version.
