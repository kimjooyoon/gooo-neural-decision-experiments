# Reusing Gooo candidate observations — 2026-10-03

The compiler can keep the first bounded matrix of candidate outputs and filter it
when a declared Gooo reference activity supplies another expected value. This
experiment uses the frozen compact bag-original FP32 judge during actual code
generation and compares fresh evaluation with output reuse.

## What ran

- Compiler: `87d8afae97c22db4b5bd6e604a88fccb53af20ba`, clean Go 1.27.1 build,
  SDK v0.2.17-experimental. [Integration PR](https://github.com/kimjooyoon/meta-ontology-go/pull/1162).
- Collector: `f79eb169e741558e848b20a79f71aab3b316c3bc` in this repository.
- Apple M4, darwin/arm64; fresh compiler process for each generation. Compiled
  execution follows each generation and runs twice. Model weights are unchanged.
- One authored three-subtraction task, eight candidate paths, Korean/English
  intent views, model on/off, four observation modes and six repetitions:
  **96 generations, 120 real model predictions, 192 compiled executions**.
- Initial selection case: `10 → -3`. Candidates 0 and 7 both pass. A separate
  source reference activity computes `7-input`; observing `0 → 7` leaves mask 7.
  The supplied oracle defines the intended finite behavior in this experiment.
- Runtime inputs: `[-8,-1,0,3,10,21]`. Each generation has six expectations;
  repeated executions are not counted twice in the accuracy denominator.
- No optimizer update or external inference call is performed.

## Results

Each row contains 12 generations. Time and peak RSS are medians of fresh-process
observations. CPU is `(user + system time) / wall time`, with one core = 100%.

| Model | Observation | Expectations met | Predictions | Probe evaluations | Codegen ms | Observation ms | CPU, one core | Peak RSS MiB |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| off | none | 12/72 | 0 | 0 | 7.739 | 0 | 86.98% | 16.55 |
| off | rank only | 12/72 | 0 | 168 | 8.220 | 0.190 | 88.21% | 17.27 |
| off | oracle, fresh | 72/72 | 0 | 396 | 8.314 | 0.472 | 88.76% | 17.79 |
| off | oracle, reuse | 72/72 | 0 | 168 | 8.502 | 0.423 | 89.44% | 17.59 |
| on | none | 42/72 | 12 | 0 | 8.206 | 0 | 87.71% | 17.43 |
| on | rank only | 42/72 | 12 | 168 | 8.577 | 0.188 | 88.47% | 17.58 |
| on | oracle, fresh | 72/72 | 48 | 396 | 9.213 | 0.603 | 92.35% | 18.83 |
| on | oracle, reuse | 72/72 | 48 | 168 | 9.372 | 0.333 | 93.02% | 18.83 |

Both oracle modes met **288/288** expectations together. Including all controls,
the count is **396/576**. All arms pass the original sparse selection case.
The English model view chooses mask 7 immediately; the Korean view needs further
search once the reference observation is available.

Reuse reduces probe evaluations from **33 to 14 per request**, a 57.6% reduction.
The continuation performs two cached comparisons and keeps three probe outputs,
with zero new evaluations. Independent reading checks all **24 fresh/reuse pairs**:
generated source, selected choices, model-call counts and ordered runtime values
are equal. The initial fixed candidate-output matrix is 16 KiB; other session,
plan, receipt and process allocations are additional.

Whole-codegen latency increased slightly in this collection: 8.314 → 8.502 ms
without the model and 9.213 → 9.372 ms with it. The shorter observation phase has
not produced a demonstrated end-to-end latency improvement. The prior SDK kernel
microbenchmark measures only a cached continuation operation. Host-wide CPU
utilization change and performance on other hardware remain unmeasured.

## What this suggests for Gooo

This establishes a smaller repeated computation cost and preserves the finite
behavior. The compiler still performs model-guided or deterministic search after
the oracle has left one candidate. In the model oracle arms that means 48 model
predictions and 54 search attempts per 12 generations in both modes. Connecting
the resolved candidate directly to projection is the next narrow experiment.
Its receipt must still distinguish complete finite enumeration from an incomplete
candidate budget and from behavior on inputs outside the supplied suite.

The model contributes a local route preference; the language carries source,
typed choices, reference behavior and observed failures. This study adds no new
task-generalization result. Execution-permission completeness is still UNKNOWN
in all runtime receipts.

## Reproduce and inspect

From the repository root, using a clean compiler built at the commit above:

```sh
go run ./cmd/path-observation-observe \
  --compiler /absolute/path/to/gooo \
  --compiler-sha 87d8afae97c22db4b5bd6e604a88fccb53af20ba \
  --go-bin /absolute/path/to/go \
  --out publication/a-new-reuse-run --include-reuse --repeats 6
go run ./cmd/path-observation-read --dir publication/path-observation-reuse-20261003
```

`manifest.json` pins compiler/model hashes and counts; `processes.json` retains
every sample. Each generation and runtime receipt is kept separately. The reader
recomputes candidate arithmetic, added observations, evaluation/reuse counts,
source bindings, parent receipt identity, model calls and pair equality. It makes
zero compiler, model or generated-program calls. Public strings and decoded parent
receipts are checked for private-path and credential patterns. `SHA256SUMS` records
the published byte inventory. The earlier 24-generation pilot is preserved in
[`path-observation-loop-20261003`](../path-observation-loop-20261003).
