# One Go example: generation followed immediately by native execution

2026-10-03. [Source `bcda476c`](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/bcda476c7976579771313b1833d025ddc44e9e3b/cmd/order-example)
provides `go run ./cmd/order-example`. It reads the public Gooo source, short
recipe and independent finite cases; sends one request to `gooo body-path-stream`;
executes the received projection immediately; then sends the next request while
the stream's input remains open. The caller keeps each original response,
generation, native execution report and a compact summary.

[Runnable instructions](../../examples/whole-candidate-order).
Compiler: clean installed main `cb2892cb583e693190bd68f75eb4df223819f053`,
Go1.27.1, SDK v0.2.20-experimental, macOS arm64. The original 16 KiB model and
known Korean `input * 2 + 1` example are unchanged. **Four generations, two actual
model predictions, eight compiled executions, 32/32 finite expectations** passed.
All generated sources match the preceding installed-stream observations.

| Route / request | Response ms | Native command ms | Predictions | Prepared reuse | Passed / total |
| --- | ---: | ---: | ---: | --- | ---: |
| Model / first | 51.393 | 655.812 | 1 | false | 8 / 8 |
| Model / second | 2.056 | 309.063 | 1 | true | 8 / 8 |
| Deterministic / first | 7.191 | 328.262 | 0 | false | 8 / 8 |
| Deterministic / second | 2.016 | 312.335 | 0 | false | 8 / 8 |

Response measurement starts when submitting a request. The first interval can
include residual worker startup and model loading. Native command time includes
source replay, toolchain validation, compilation and two executions. These are
two sequential requests per route; route order and warm system state are not
controlled. This confirms the execution flow and locates costs in these records.
It does not attribute a speedup or slowdown to the model.

The model-route process took 1.44 s wall, 0.51 s user+system CPU (35.4% of one
core on average) and OS-reported maximum RSS 87,457,792 bytes. The deterministic
route took 0.65 s wall, 0.49 s CPU (75.4%) and 86,605,824 bytes. These include the
Go compiler and child work; 16,384 bytes describes model tensors alone. Peak
system CPU utilization was not sampled. Every resource line is retained.

## What the execution records show next

All four emitted executable digests are equal. Build intervals are 97–267 ms;
first executions are 168–351 ms; second executions are 4–8 ms. Both building
and first execution contribute to the cost. We retain these stages separately
in `runtime-stages.json` and make no claim about the cause of the first-execution
delay. A [next protocol](../../docs/retained-native-execution-protocol-20261003.ko.md)
proposes retaining one exact compiled body while executing current inputs anew.
That executor is not implemented by this example.

Tests cover bounded input reading, mismatched/rejected responses, execution
before the next request and input EOF, finite failures retaining summaries,
fresh output directories and cancellation of a blocked response. Linux/macOS
commands cancel their child process groups; other platforms retain direct-child
cancellation. The whole example has a 90-second deadline and 1..16 sequential
requests. Race tests and vet passed locally.

[Linux `order-example` job](https://github.com/kimjooyoon/gooo-neural-decision-experiments/actions/runs/37107355051/job/111158387657)
also passed at this exact source, including model/deterministic native execution.
The downloaded artifact `11267424325` matched SHA256
`0ea67fa598ed5c23a6b6213b7e7318fbc6bda9293a47a13378f51accd0bb8919`.
Its separate four generations, two model calls, eight compiled executions and
32/32 expectations passed; all generated Go sources match the local originals.
Model response times were 8.084/4.575 ms, with native command times
5,351.994/182.901 ms. Deterministic times were 7.275/5.276 ms and
182.979/182.150 ms. The long first native command remains in the original
records. Hardware, platform and cache state differ from the local collection.
The existing whole-research CI contains a preserved historical arithmetic replay
failure; this job's result is scoped to the example.

`native.zip` contains all local original receipts and source comparisons.
`summary.json` counts this local collection only. Build information replaces its
local path with the collector basename. Verify publication files with
`shasum -a 256 -c SHA256SUMS`. New training updates: zero.
