# Repeated Gooo generation with prepared candidates

2026-10-03, Apple M4, macOS arm64, Go1.27.1. Compiler and collector:
[`a1f56d47af22e709a98310469a428c0d28094034`](https://github.com/kimjooyoon/meta-ontology-go/commit/a1f56d47af22e709a98310469a428c0d28094034),
public SDK v0.2.20-experimental. Both binaries were built from the same clean
revision. [Compiler PR #1178](https://github.com/kimjooyoon/meta-ontology-go/pull/1178)
is undergoing CI at publication; this collection precedes promotion/installation.

## What changed

The compiler binds the current Gooo source to the current plan, then reuses the
eight prepared candidate programs when their complete plan digest matches. One
generator retains at most one plan. Each request still makes a fresh model
prediction and evaluates its current cases. A changed plan replaces preparation.
Concurrent callers can independently prepare a miss and publish only complete
immutable objects. There is no wait on a cache-building owner.

The model is the unchanged public 16 KiB whole-candidate judge. Training updates:
**zero**. The original and frozen V3/V4 implementations remain intact.

## Actual generation and native execution

We used the original 64 new-template/new-constants requests, budgets 1 and 8,
and two ownership modes: two freshly constructed generators, or two calls on one
retained generator. Mode order alternated between pairs. Every generated result
was immediately built and executed twice before the next generation started.

- **512 generations, 512 model predictions, 1,024 compiled executions.**
- 128 second calls reused preparation. All other calls prepared their candidates.
- Full search records, predictions, descriptor aliases, selected Go and eight
  ordered native case outcomes match the original study. Only ranking prediction
  duration is excluded from that semantic comparison.
- Budget 1 remains **52/64 complete requests, 418/512 case expectations** per mode/trial.
- Budget 8 remains **64/64 complete requests, 512/512 expectations** per mode/trial.
- All repeats together: **3,720/4,096** finite expectations. Original budget-one
  failures are included. These repeated development requests are already observed.

## Measured cost

Compare the second call of each mode at budget 8: 64 calls per row, matched
requests. The fresh owner loads the model and prepares candidates for this call.
The retained owner already has the matching model and candidates.

| Interval or counter | Fresh owner | Retained owner |
| --- | ---: | ---: |
| Generate API median / p95 | 1.017 / 1.518 ms | 0.693 / 1.056 ms |
| Recipe expansion + constructor + Generate median / p95 | 2.225 / 2.829 ms | 1.627 / 2.205 ms |
| Preparation acquisition median | 0.261 ms | 0.000208 ms |
| Allocated bytes during Generate, median | 1,084,784 | 768,808 |
| Allocations during Generate, median | 13,672 | 8,498 |
| Separate native build + two runs, median | 308.34 ms | 310.10 ms |

55/64 matched Generate calls are faster with retained preparation; nine are slower.
The median paired saving is 0.309 ms. Comparing the two medians gives a 31.9%
reduction in Generate time and 26.9% in the sum of recipe expansion, constructor
and generation. `costs.json` contains every paired operand, both budgets, both
trials, outliers and allocation counters. This is one collection with two calls
per pair, rather than an estimate of performance variation across machines/runs.

Generation still performs source parsing/binding, selected-source emission,
typechecking and fresh receipts. Keeping candidates does not remove those costs.
Repeated recipe expansion is about 0.85 ms at its median in the retained second
call. This is a concrete remaining cost to investigate before enlarging the model.

The full collector plus its child work takes **167.51 s wall**, **70.58 s user +
53.54 s system CPU** as reported by `time` (74.1% of one core on average), with
**87,703,552 bytes** reported maximum RSS. These process/resource observations
include native compilation and execution. They do not measure whole-host peak CPU
utilization or per-request resident memory. Native child medians are also
recorded separately. Generate allocation-counter reads and result serialization
are outside the timed generation interval; runtime activity can affect allocation
counts and GC timing.

All API calls share a collector process. The historical 9.26 ms fresh compiler
CLI observation includes a different startup/serialization interval. It is not
an old/new speedup comparison. The [SDK-only experiment](../order-prepared-sdk-20261003)
separately measures candidate preparation, searches and retained heap.

## Initial failures retained

The first collector (`3f685eff`) tried to read the short source recipe as an
already expanded document and stopped at `unknown field "choices"` before any
Generate call. The correction uses the same `DecodeSourcePathDocument` API as
the CLI. Its first launch then rejected an output name already occupied by the
collector binary. That launch also performed zero generations. Both original
logs and resource reports are included. The complete run used a fresh directory;
none of its observations were removed or retried.

## Reproduce and inspect

The [collector and pre-collection protocol](https://github.com/kimjooyoon/meta-ontology-go/tree/a1f56d47af22e709a98310469a428c0d28094034/examples/order-prepared-native)
specify the fixed baseline hashes, two calls per ownership mode and immediate
native execution. Build collector and compiler from that clean revision. Use the
unchanged [native baseline](../order-judge-native-20261003) and original
[model metadata/weights](../order-judge-initial-20261003).

```sh
shasum -a 256 -c SHA256SUMS
unzip -q native.zip -d /tmp/gooo-prepared-native
jq -f costs.jq /tmp/gooo-prepared-native/records.json
```

`native.zip` includes all source recipes, emitted results, native reports, flushed
progress records and the complete manifest. Generated reports bind source/plan,
compiler revision and model identities; no weights were changed or uploaded by
this collector. Final focused race/vet/semantic logs are published alongside.
The earlier full macOS unit run still has existing namespace/symlink/source-split
failures; two native child timeouts under that load passed isolated replay.
Canonical Linux PR CI is tracked separately and has not yet authorized promotion.

### 한국어 요약

같은 조립 도면의 후보를 다시 준비하지 않도록 했습니다. 추가 학습 없이 512회 생성과
1,024회 실제 실행에서 기존 결과를 유지했습니다. 두 번째 API 호출의 생성 중앙값은
1.02→0.69ms, 조립 계획 읽기·모델 준비까지 더하면 2.22→1.63ms였습니다. 모델은 매번
새로 판단합니다. 최대 여덟 번 시도하면 기존 64개 과제를 모두 완성했고, 한 번 시도에서
남았던 12개 미완성 과제도 그대로 기록했습니다. 전체 호출에서 남은 큰 비용은 소스와
짧은 조립 계획을 읽고 확인하는 부분입니다.
