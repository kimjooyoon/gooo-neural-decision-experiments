# Projecting the candidate that observation has resolved

2026-10-03. Gooo can use the single survivor of a complete declared candidate
enumeration directly. A source reference activity supplies the missing example;
the compiler assembles and checks that body before any model loading or search.
This opt-in mechanism composes with cached candidate outputs.

## What ran

- Clean compiler `9158c3cd923e299d1d89502fb6ab397179364a74`, Go 1.27.1,
  SDK v0.2.17-experimental; [implementation PR 1164](https://github.com/kimjooyoon/meta-ontology-go/pull/1164).
- Collector and independent reader: `7a3e331ff38410d53c3780de1225ab8d2757cc44`.
- Apple M4, darwin/arm64. Each generation starts a fresh process, followed by
  two compiled executions. The frozen compact bag-original FP32 judge is used
  in model-enabled control arms. Training updates and external calls: zero.
- One authored task with three subtraction choices, eight candidate paths,
  Korean/English views, model requested/off, five observation modes and six
  repetitions: **120 generations, 120 actual predictions, 240 compiled runs**.
- The original case `10 -> -3` leaves masks 0 and 7. The declared reference
  activity computes `7-input`; observing `0 -> 7` leaves mask 7. Runtime inputs
  are `[-8,-1,0,3,10,21]`. Repeated execution is counted once in the denominator.
- Arm and model-state order reverse on odd repetitions. This is an interleaved
  mechanism pilot on one task, with no randomized hardware-level causal study.

## Complete comparison

Each row contains 12 generations and 72 runtime expectations. Wall time and peak
RSS are process medians. CPU is `(user + system time) / wall time`, one core =
100%. The requested model is actually skipped in resolved rows.

| Model requested | Observation | Expectations | Predictions | Search attempts | Probe evaluations | Codegen ms | CPU | Peak RSS MiB |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| no | none | 12/72 | 0 | 12 | 0 | 7.699 | 86.83% | 16.72 |
| no | rank only | 12/72 | 0 | 12 | 168 | 8.157 | 89.48% | 17.11 |
| no | oracle, fresh | 72/72 | 0 | 96 | 396 | 8.497 | 88.10% | 17.62 |
| no | oracle, reuse | 72/72 | 0 | 96 | 168 | 8.508 | 89.46% | 17.60 |
| no | oracle, reuse + resolve | 72/72 | 0 | 0 | 168 | 8.273 | 88.44% | 17.35 |
| yes | none | 42/72 | 12 | 12 | 0 | 8.252 | 88.70% | 17.33 |
| yes | rank only | 42/72 | 12 | 12 | 168 | 8.499 | 88.89% | 17.59 |
| yes | oracle, fresh | 72/72 | 48 | 54 | 396 | 9.340 | 92.65% | 18.91 |
| yes | oracle, reuse | 72/72 | 48 | 54 | 168 | 9.491 | 92.45% | 18.77 |
| yes | oracle, reuse + resolve | 72/72 | 0 | 0 | 168 | 8.267 | 88.01% | 17.29 |

Direct projection meets **144/144** expectations. All oracle modes together meet
**432/432**, and all arms including sparse-case controls meet **540/720**.
Every arm passes the original selection case. The 120 actual predictions belong
to controls, preserving a real model comparison without attributing model work
to the deterministic resolved route.

The independent reader checks **24 reuse pairs and 24 resolution pairs**. Bodies,
selected choices and ordered runtime values match their controls. Resolution
receipts bind the final ranking and effective cases, with explicit skipped model,
feedback and search work. Search records remain empty; observed examples are
accounted separately. Compiled execution first replays these source bindings.

## Cost and interpretation

Within this collection, model-requested reuse vs direct projection changes median
generation from **9.491 to 8.267 ms** (12.9% lower) and peak RSS from **18.77 to
17.29 MiB**. Process CPU medians change from **92.45% to 88.01%** of one core.
Model-off generation changes from 8.508 to 8.273 ms. These differences describe
this capture; broader tasks, retained workers and other machines need their own
measurements. Whole-host CPU utilization change is unmeasured.

The initial no-model/no-observation process took 454.691 ms. It remains in the
raw records and medians; no sample was removed and its cause was not measured.
The earlier [96-generation reuse experiment](../path-observation-reuse-20261003)
is preserved: its observation stage improved but whole generation became slightly
slower. Direct projection addresses the subsequent work that experiment exposed.

The output matrix remains a fixed 16 KiB. Other compiler/process allocations are
additional. This experiment reports fresh-process costs, including skipped model
loading. A retained worker has already loaded its model; it skips predictions
per resolved request and records that distinction.

## Language meaning and remaining scope

The small mechanism is useful when an existing specification or reference
activity can distinguish allowed implementations. Gooo retains the intention,
candidate set, new observation and resulting body together. The opt-in request
uses `"reuse_probe_outputs": true` and `"resolve_unique_candidate": true`.
An incomplete enumeration, zero survivors or several survivors continues bounded
search and records why. Default behavior remains available.

The reference defines this experiment's expected behavior. One surviving finite
candidate and 144 passing expectations do not establish correctness on arbitrary
inputs or new tasks. All runtime receipts retain the unresolved execution
permission dimension. No generalization score or learned capability is added.
The research motivation is informative observation from
[LAVOIR](https://arxiv.org/abs/2609.30706), with assembled behavior checked in light
of [decision-composition findings](https://arxiv.org/abs/2609.33971).

## Reproduce and inspect

Build the clean compiler at the pinned revision and use a new output directory:

```sh
go run ./cmd/path-observation-observe \
  --compiler /absolute/path/to/gooo \
  --compiler-sha 9158c3cd923e299d1d89502fb6ab397179364a74 \
  --go-bin /absolute/path/to/go \
  --out publication/a-new-resolution-run \
  --include-reuse --include-resolve --repeats 6
go run ./cmd/path-observation-read --dir publication/path-observation-resolution-20261003
```

The manifest pins binary/model hashes and counts. `processes.json` retains every
sample; generation and runtime documents preserve source-bound evidence.
`independent-reading.json` recomputes arithmetic, receipt links, pair equality and
work counts with zero compiler/model/program calls. Its publication scan includes
decoded parent receipts. `SHA256SUMS` closes the published byte inventory.
