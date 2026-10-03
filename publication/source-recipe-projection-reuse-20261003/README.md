# Reuse one checked projection during source recipe expansion

2026-10-03, Apple M4/macOS arm64, Go1.27.1, public SDK v0.2.20. Recipe expansion
formerly generated the original checked Go projection twice. The candidate uses
the first projection for fallback binding within that request. It checks the
exact source, activity, emitted projection digest and successful type/replay
result, then still checks fallback equivalence. Generation/export/replay bind
fresh source. Every request owns its projection and recipe arrays.

## Native behavior and original cost observations

The unchanged public 16 KiB model made 512 predictions during 512 generations.
All 1,024 immediately compiled executions completed. The complete original
search, ranking (apart from duration), generated Go and finite outcomes match.
Budget-8 groups remain 64/64 complete; budget-1 groups retain 52/64 complete.
Across both budgets, all original outcomes total **3,720/4,096 expectations**.
Original failures remain visible. Additional training updates: zero.

This collector/compiler pair was clean revision
[`33e89de3b515e2d39bd8ba830b25f416dec71151`](https://github.com/kimjooyoon/meta-ontology-go/tree/33e89de3b515e2d39bd8ba830b25f416dec71151).
The [protocol note](https://github.com/kimjooyoon/meta-ontology-go/blob/33e89de3b515e2d39bd8ba830b25f416dec71151/examples/order-prepared-native/README.md)
was committed before collection. Native collection took 170.53 s wall,
71.47 s user + 54.10 s system CPU, reported max RSS 87,883,776 bytes. These
process totals include compiled native work.

The historical versus follow-up warm budget-8 recipe decode medians were
0.849 and 0.960 ms; decode + setup + Generate medians were 1.627 and 1.641 ms.
They did not establish a wall improvement. Their collection times differed.
Both full datasets and the comparison remain published in `native.zip` and
`native-cost-comparison.json`, motivating the controlled follow-up below.

## Paired decoder measurement

Identical collector source was built in two clean revisions:

- Prior decoder plus collector: [`3fd41f00433887354fc9b1be3f1080bc4cadd54e`](https://github.com/kimjooyoon/meta-ontology-go/tree/3fd41f00433887354fc9b1be3f1080bc4cadd54e).
- Projection reuse plus collector: [`e28bab390413e31c0c06eed0c6cb3d3bba7e9c8e`](https://github.com/kimjooyoon/meta-ontology-go/tree/e28bab390413e31c0c06eed0c6cb3d3bba7e9c8e).

The [protocol](https://github.com/kimjooyoon/meta-ontology-go/blob/e28bab390413e31c0c06eed0c6cb3d3bba7e9c8e/examples/source-recipe-cost/README.md)
was committed before measurement. Four sequential paired trials alternate
baseline/candidate process order. Each process uses 64 existing requests × two
budgets × eight fresh measured decodes, after one unmeasured warm-up per pair.
**8,192 measured decodes, 4,096 paired calls** all have identical full document
and plan digests. Model predictions in this cost-only collector: zero.

| Decoder interval, pooled median | Prior | Reuse |
| --- | ---: | ---: |
| Decode wall | 0.3714 ms | 0.2921 ms |
| Allocated bytes per decode | 728,580 | 571,800 |
| Allocation count per decode | 8,343 | 6,545 |
| Decode p95 | 0.5717 ms | 0.4895 ms |

Paired saved-time median is 76,562 ns. 3,345 calls were faster and 751 slower;
maximum candidate decode latency (3.195 ms) exceeded the baseline maximum
(2.070 ms). All four trial medians favored reuse. Pooled median decode time and
allocated bytes fell by approximately 21% on this measured workload.
Allocated bytes describe new objects during decoding, not resident memory.
Runtime/heap work can affect allocation counters. Whole CLI, retained transport
and native execution have separate intervals. The first collection's v1 manifest
field `requests:128` counts request/budget records; actual distinct source requests
are 64. Original manifests are preserved; future v2 names both quantities.

`paired.zip` retains every raw JSONL record, original manifest, process resource
record and summary. The build-info first line replaces a local binary path with
its basename; source revision and clean-tree flags remain intact.

## Verify

```sh
shasum -a 256 -c SHA256SUMS
unzip -q paired.zip -d /tmp/gooo-recipe-cost-paired
jq -f paired-summary.jq /tmp/gooo-recipe-cost-paired/records.json
```

Focused race tests verify strict recipes, stale/altered projections, current
source changes, fallback mismatch, cancellation and candidate/stream behavior.
Full local vet passed. Compiler CI/promotion is tracked separately from these
local observations. The implementation branch replays the same production
patches onto dev after the stream command; only collector metadata was clarified.

### 한국어 요약

짧은 레시피를 전개하며 같은 원본을 중복 검증하던 부분을 줄였습니다. 현재 소스와
생성물의 연결, 타입·재현 검사, 대체 바디와의 일치 검사는 계속 수행합니다.
먼저 512회 생성과 1,024회 실행에서 기존 결과·실패가 유지됐음을 확인했습니다.
그 수집에서는 전체 시간 개선이 확인되지 않아, 같은 입력을 두 버전에 번갈아 보내는
추가 측정을 했습니다. 8,192회 전개에서 중앙 시간은 0.371→0.292ms,
호출당 할당량은 약 729→572KB였습니다. 751쌍의 시간 악화와 모든 원본도 공개합니다.
