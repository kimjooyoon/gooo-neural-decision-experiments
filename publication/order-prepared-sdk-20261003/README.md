# Reusing prepared whole-candidate programs

2026-10-03, Apple M4/macOS arm64, Go1.27.1. The new Go SDK package
[`orderprepared`](https://github.com/kimjooyoon/gooo-decision-runtime/tree/v0.2.20-experimental/orderprepared)
retains eight fully checked programs, fixed input features and the identity of a
privately captured model. Every search still predicts and checks fresh cases.
The original whole-candidate implementation and V3/V4 inventories are preserved.

## Paired results

We used the same weights and 64 previously observed new-template/new-constants
requests. Both model/deterministic arms and budgets 1/8 run three modes, alternating
their order over five rounds: **3,840 actual searches, 1,920 model predictions,
zero training updates and zero native executions** in the primary follow-up.

All complete search records, predictions, descriptor aliases and selected Gooo/Go
match. Only the named prediction timer is excluded from semantic comparisons.
All budget-8 results also meet the eight finite evaluation expectations per
request. These are repeats of an observed development cohort.

Model arm, budget 8, 320 timed calls per mode:

| SDK interval | Median | p95 | Median allocated bytes | Median allocations |
| --- | ---: | ---: | ---: | ---: |
| Original whole search | 356.98 µs | 638.83 µs | 497,600 | 7,909 |
| Runtime capture + prepare + search each call | 244.04 µs | 445.29 µs | 352,872 | 5,200 |
| Search using retained candidates | 4.83 µs | 6.79 µs | 2,848 | 13 |

One-time model capture plus candidate preparation has a median of 247.40 µs
across 64 requests. Using each request's measured setup and median old/warm costs,
the estimated break-even is one call for 54 requests and two for 10 at budget 8.
`break-even.json` retains every operand and estimate. This arithmetic estimate is
separate from a directly measured sequence of complete compiler requests.

Three follow-up live Go heap deltas for 64 prepared model plans plus one owned
model are 1,682,336 / 1,715,440 / 1,682,272 bytes. They include private program
objects and may include runtime noise. The shallow Prepared struct is 2,568 bytes;
its model, sources and program arenas use additional memory. Callers control how
many plans remain alive. The API has no global cache.

The collector takes 1.71 s wall and 1.66 s process CPU, with 148.59 MiB maximum
RSS. Its large retained result records and JSON serialization are included in
that process total. Allocation-counter reads and output serialization are outside
individual timed intervals; their effects on the process remain a measurement
limitation. These SDK timings establish the cost of preparation/search only.
Compiler CLI and warm generator measurements follow separately.

## Initial observations and the memory correction

The first clean collector `512d0371cf22ecbc18f36cba36d70881b863183e` completed a
one-round pilot (768 searches, 384 predictions) and a five-round run (3,840
searches, 1,920 predictions). All semantic comparisons passed. The first retained
heap delta in each was negative: earlier large JSON serialization buffers were
being reclaimed between the baseline and retained-object measurements.

Collector `0d5f90749d4bf942c474a2762f09c5a9df31a02d` moves residency observation
before result serialization and performs two baseline collections to drain
prior-cycle pools. Its complete five-round semantic records match the first run.
Canonical SHA256 after removing `.Cost` and setting `.Result.ranking.predict_ns`
to zero is `a60dcd6babc19fc4d30997e5bb10c45f9bfbff9c64e8ac7a50394ac577959330`.
All initial timing, negative heap values and follow-up records are included.

## Linux replay and released sources

[SDK PR #5](https://github.com/kimjooyoon/gooo-decision-runtime/pull/5) merged as
`1e49b457a8cc56a303b3d70ba4d2c23b211a76f5` and is released as
[v0.2.20-experimental](https://github.com/kimjooyoon/gooo-decision-runtime/releases/tag/v0.2.20-experimental).
The final small addition exposes captured model identity without serializing
weights again. It leaves the measured search implementation unchanged.

PR [CI 37101904204](https://github.com/kimjooyoon/gooo-decision-runtime/actions/runs/37101904204)
and push CI 37101901909 passed. Unit, race and vet checks include 512 combinations
of models, fallback masks, budgets, deduplication and finite-case suites compared
with the original search. Cancellation during preparation/evaluation, stale plan
identity, caller overwrites and four simultaneous callers also pass.

The Linux amd64 artifact contains 768 searches and 384 predictions. Its complete
semantic records match the first round of the macOS follow-up, with only cost
fields and the prediction timer excluded. Canonical SHA256:
`f4311a0193287ee86d8bba9b92a0afab09a88c68a6401a50619567809cfe5165`.
The tested head is `bbfd327a0c940481d01092749c2097873c9f5ffb`; GitHub's merge checkout
is `6b6e29264981f038b1cf6fa4892a0ff3aad02ecb`. Head and merged release have tree
`3585a2c78c0229792ff1414f88dab2e13d51c3c4`.

Immutable artifact 11266742127 has downloaded SHA256
`825c5f06746ef3d012e54a2bf2a98522c634f9e52a3c75d27cf9d3af07125fe4`.
`linux.zip` and `linux-artifact.json` retain its contents and GitHub identity.

## Reproduce

The three local ZIPs contain every record, preparation, residency observation and
build manifest. Corresponding `.log` and `.time` files retain process outcomes.
`costs.json` comes from `costs.jq` applied to the follow-up records.

```sh
shasum -a 256 -c SHA256SUMS
unzip -q order-prepared-local-v2.zip -d /tmp/order-prepared-observation
jq -f costs.jq /tmp/order-prepared-observation/records.json
```

To run again, use the SDK release's `examples/order-prepared` and decode the
unchanged [initial fit bundle](../order-judge-initial-20261003). The command accepts
`--input`, a fresh `--output`, and `--repeats 5`. Original weights, authored source,
full intent and plan identities are checked before measurement.

### 한국어 요약

같은 소형 모델과 조립 결과를 유지하면서 준비한 후보를 다시 쓰도록 했습니다.
모델 경로의 탐색 중앙값은 기존 357µs, 매번 준비하는 새 경로 244µs, 준비물 재사용
4.83µs였습니다. 유지하는 계획 64개와 모델 하나는 약 1.7MB의 Go 힙을 더 썼습니다.
첫 메모리 측정에 섞인 임시 버퍼 회수도 원본에 남겼습니다. 컴파일러 전체 속도는
실제 연결 후의 생성·실행 비용으로 확인합니다.
