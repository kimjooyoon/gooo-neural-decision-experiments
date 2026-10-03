# Prepared candidates through the existing Gooo worker

2026-10-03. This follow-up uses the existing `gooo-body-worker` transport with
the public 16 KiB model, SDK v0.2.20 and compiler revision
`a1f56d47af22e709a98310469a428c0d28094034`. Collector/protocol revision:
[`d8fcc0422ae86c2cb4a4a72f9de76547f22f2e4c`](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/d8fcc0422ae86c2cb4a4a72f9de76547f22f2e4c/cmd/order-prepared-worker).
The [protocol](../../docs/prepared-worker-protocol-20261003.md) was committed
before collection. No compiler change or additional training was needed.

## What completed

Eight previously observed English/Korean requests cover four arithmetic families.
Each model/deterministic arm uses one retained process, with worker counts 1 and 4.
Sequential requests are repeated twice while stdin remains open. Two further
four-request batches alternate between two plans and allow response reordering.
Every received result immediately builds and executes twice before the next
response is read. Already submitted requests may continue concurrently.

**96 generations, 48 model predictions, 192 compiled executions, 768/768 finite
expectations matched.** Full search, full ranking (except prediction duration),
selected Go and ordered native case outputs match the frozen original records.
All four processes exited normally after input EOF within their deadlines.

All 16 sequential second model calls reused preparation. Their preceding calls
prepared candidates again. In this collection none of the interleaved batch calls
reused preparation: the owner holds only the latest plan, and the two different
plans keep replacing it. Parallel misses can prepare independently. This exposes
the intended bounded-memory tradeoff; it is useful evidence for scheduling similar
requests together before considering a larger cache.

## Request-to-response observations

Sequential phase, eight calls per cell, milliseconds:

| Worker slots | Deterministic first / second median | Model first / second median |
| --- | ---: | ---: |
| 1 | 1.991 / 2.076 | 2.034 / 1.995 |
| 4 | 2.154 / 1.883 | 2.478 / 1.870 |

Only one request is in flight during this phase, even with four configured slots.
Intervals include request construction/JSON, transport, recipe expansion, source
binding, generation and response decoding. The first request of a process also
includes startup/loading. These small samples describe observed latency; the
[larger API comparison](../order-prepared-native-20261003) has a different interval.

Batch times are preserved but include the collector's roughly 300 ms native work
for preceding results. Use that phase to inspect completion, out-of-order delivery,
bounded queues and cache replacement. It does not estimate parallel speedup.
The observed workload completed without a timeout or transport deadlock.

Full collection: 31.08 s wall, 13.11 s user + 9.94 s system CPU, 87,638,016 bytes
reported maximum RSS. CPU/resource totals include native build/run work. Per-call
resident memory and whole-host peak utilization are unmeasured. The setup records
show one immutable 16,384-byte model tensor per model process.

## Use and reproduce

The [two-request example](../../examples/whole-candidate-order/worker-requests.jsonl)
and its [instructions](../../examples/whole-candidate-order) exercise the same
existing worker with one Korean source/recipe. The compiler must include #1178
to emit the prepared-candidate reuse receipt. The base body worker also supports
disconnected deterministic operation by omitting its model option.

```sh
shasum -a 256 -c SHA256SUMS
unzip -q native.zip -d /tmp/gooo-prepared-worker
jq -f prepared-worker-costs.jq /tmp/gooo-prepared-worker/records.json
```

The archive contains every source/recipe, emitted generation, native report,
setup record, per-process record set and final manifest. Original baseline hashes
and both binary hashes are recorded. `prepared-worker-costs.json` includes all
latency extrema and reuse counts. The collector is Go and uses bounded process
deadlines; it performs no remote model calls or training.

### 한국어 요약

기존 Gooo 워커를 실제로 유지하면서 한영 요청을 생성·실행했습니다. 96회 생성과 192회
실행에서 768개 기대값이 일치했습니다. 입력을 닫기 전에도 응답했고, 네 요청을 함께
보낸 묶음도 완료했습니다. 같은 계획의 연속 두 번째 호출 16건은 후보 준비를 재사용했습니다.
서로 다른 두 계획을 교차한 묶음에서는 재사용이 0건이었습니다. 최신 계획 하나만 보관하는
특성을 고려하면, 반복 조립을 같은 소스별로 모아 처리하는 방식부터 실험할 가치가 있습니다.
