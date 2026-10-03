# Existing Gooo worker: prepared candidates and immediate native execution

Fixed before collection, 2026-10-03. Use the existing `gooo-body-worker` command
and Gooo compiler from clean revision `a1f56d47af22e709a98310469a428c0d28094034`,
SDK v0.2.20, original 16 KiB weights, and original native records digest
`47315bf031f4734497f1a9f3710e93b6c0e8f43ed385f78ec7d7cc51adfa2be1`.
This adds observations of the existing transport; no compiler behavior changes.

Take eight already observed `new-template` requests: four arithmetic families,
English/Korean, source order 0, wanted order 1, template 2. Budget is 8.
For each model/deterministic arm and worker count 1/4:

1. Keep one worker process and its stdin open. Send each request twice, receiving
   and immediately building/executing its result before sending the next.
2. Send two bounded batches of four requests, with two different plans repeated
   within each batch. Match output by sequence and correlation ID; it can reorder.
3. On receiving each result, immediately execute that exact projection twice
   before reading the next result. Other submitted work can continue in parallel
   and the output pipe supplies backpressure. Close stdin only after all results.

Planned totals: 96 generations, 48 model predictions, 192 compiled executions,
768 finite native expectations. Only completed records count. Compare the complete
search, complete ranking except `predict_ns`, selected Go, and all ordered runtime
cases with the original observations. Sequential model repeats must reuse
preparation exactly on the second call. Batch reuse is observed, without requiring
a schedule-dependent count. The disconnected arm must make zero predictions.

Request-to-response timing includes request construction/JSON, pipe transport,
source recipe expansion, generation and response decoding. The first request also
includes worker startup/loading. Batch response latency includes time spent
executing earlier responses in the collector; use those batches for bounded
concurrency/completion evidence. No parallel speedup estimate is planned.

The worker gets a three-minute process deadline, while each native execution has
the existing bounded process-group cleanup. Archive every response-derived
generation and runtime result, setup records, per-arm observations and a final
manifest. Startup/transport failure stops collection and preserves written files.
No training or external model calls are involved.

Build `./cmd/order-prepared-worker` from a clean research checkout. Required flags:
`--worker`, `--compiler`, `--compiler-sha`, `--go-bin`, `--baseline`, `--model`,
and a fresh `--out` directory. The collector verifies both binaries' clean source
identity and public SDK version, the original baseline digest and model weights.
