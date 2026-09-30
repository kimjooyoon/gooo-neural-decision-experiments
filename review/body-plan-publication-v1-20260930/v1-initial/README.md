# Actual Laya and Go typed-body capture

Frozen source: `7d626b9ab0b503f5c3d571be09524d689271bd48`.
Frozen cohort SHA-256:
`e517b948a8c1d744cf832b5d711918dd2fef0b90cd951126f579ac54ada427a9`.
All sources and cases were fixed before these inference calls. No training or
model tuning occurred during this capture.

## Scope and execution

128 parameterized program scenarios across eight families contain 222 operation
holes. They include arithmetic, comparisons, Boolean composition, reassignment,
piecewise and nested branches, polynomial order, and signed boundaries. Each arm
uses the same 3,240 training and 1,280 heldout cases. These are 128 scenario
intents, not 128 unrelated language features or 1,280 distinct experiments.

All 1,280 native Gooo generation cells and 45,200 independent compiled Go case
executions were observed. Unknown case count and native failed/unknown cell
count are zero. There were 222 actual Laya HTTP POSTs, no warmups or retries,
and 666 tiny-model predictions. Search reuses each initial choice; it does not
call any model again. Total pipeline wall time was 47,122.5 ms, including native
generation and Go compilation/testing.

| Initial selection | Training correct / 3,240 | Heldout correct / 1,280 | Gold holes / 222 | Search attempts to final full training score |
| --- | ---: | ---: | ---: | ---: |
| Declared fallback | 944 | 384 (30.0%) | 0 | 717 |
| Tiny FP32 | 2,414 | 950 (74.2%) | 165 | 377 |
| Tiny PTQ ternary | 1,724 | 679 (53.0%) | 133 | 599 |
| Tiny QAT ternary | 2,312 | 916 (71.6%) | 160 | 491 |
| Laya multilingual | 2,389 | 945 (73.8%) | 144 | 404 |

Every search arm reached 3,240/3,240 training and 1,280/1,280 heldout correct,
including model-free search. Search has the same maximum of 64 candidates per
program, not identical observed work. Attempts include the initial program.
FP32 reduced observed candidate attempts by 47.4% and Laya by 43.7%, relative
to fallback search. The fallback operations were deliberately incorrect for all
holes; its 30% score is not a tuned deterministic baseline.

These results support using a model to propose an initial finite choice and
reduce search work. They do not attribute final correctness to model inference.
The FP32/Laya difference is small and uses different choice protocols: the tiny
model predicts eight global labels; Laya chooses among two or three supplied
typed candidates. Case rows within a family are correlated, and the independent
Go oracle shares a scenario designer with the plans. No broad model superiority
or all-input correctness is claimed. The tiny model's original 100% template
test score fell to 74.2% on these new, more complex hole instructions.

## Laya response and process measurements

The existing upstream Laya service ran on CPU with four threads and cached
multilingual weights at revision
`55cf4c4ebb4ebe31b2550e8bdf3bd21b99753851`. The connection and experiment runner
are Go; this optional reference service is upstream Python/PyTorch. The tiny
models and their inference path run directly in Go.

For the 222 unique Laya calls, HTTP wall time, including response-body reading,
was 36.2 ms minimum, 40.9 ms median, 46.8 ms p95 and 184.3 ms maximum. The p95
uses the floor-index order statistic. This is not GPU or model-kernel time.

The separate process monitor observed 761 samples across about 190 seconds,
including idle time before and after the 47.1-second experiment. Cumulative CPU
time increased by 12.12 seconds over that complete window. RSS was 453,520 KiB
at the first sample (442.9 MiB) and peaked at 701,520 KiB (685.1 MiB), an increase
of 242.2 MiB. The largest sampled rolling CPU estimate was 89.7% on a one-core
basis. This is neither exact inference-window utilization nor whole-machine
CPU increase. No GPU utilization was measured. The owned service and monitor
were stopped after all calls settled.

## Evidence interpretation

`report.json` is the terminal, completed report. `selected-cells.jsonl` contains
selection-stage snapshots written before compiled execution. `progress.json`
preserves the final incremental running snapshot; it is not the terminal status.
All are retained verbatim.

Each body has a Gooo source, native generator output and Go projection. Each arm
has independently compiled Go source/tests and exact per-case markers. The 128
Laya exchange files preserve all 222 requests and base64-encoded raw replies.
`laya-process-observations.jsonl` and `resource-monitor.go.txt` preserve the
separate sampling evidence and generic monitor source.

Pinned executable SHA-256 values:

- Go study runner: `b1f54a9ee175249d65042df40b0d3447c3bd3e3cb5cbc50954a0a24a930d7611`.
- Native Gooo: `7329b8d255b083bacfd7d44c7271caa4e3c8254bd48cc085068c92665a02591a`, built from `60cf7f49b0e302a6bebb42bc8da90f3ed19b2b82`.
- Go 1.27 toolchain: `a19a71df81715c12d9a7e81bab036c12696fec1ddbd4258b48a2131a9080b267`.
- Resource monitor source: `cfed251abeb1f1caadd23e81a36e981f2d120f15acf37b0c017dd8ca96b924b5`.
- Resource monitor binary: `ff81b409daadc51a4657ac0f6fc54a475a0e123b2e29c4ffd37936f572ae3436`.

The runner rechecked executable digests after execution. This is recorded local
evidence and an independent artifact audit, not a signed hardware attestation.
The matching eight-arm CI study is
[run 36722049671](https://github.com/kimjooyoon/gooo-neural-decision-experiments/actions/runs/36722049671).
It observed 1,024 native generation cells and all 36,160 Go cases. Its audit and
the separate local/Laya audit are under `review/body-plan-ci-v1-20260930/`.
