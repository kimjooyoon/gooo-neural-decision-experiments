# Whole-candidate judge inside the Gooo compiler

Observed on 2026-10-03, Apple M4/macOS arm64, Go1.27.1. The published 16 KiB
model now ranks actual bodies **inside source-bound code generation**. Each
generated result was immediately compiled and executed twice before the next
generation. This collection completed 256 generations, 128 real model predictions
and 512 compiled executions, with zero training updates or network inference.

The 64 requests are the previously observed `new-template` and `new-constants`
development groups: four arithmetic families, Korean/English, both original source
orders and both desired orders. Three selection inputs are -2, 0, 3; the eight
evaluation inputs are -3 through 4. Five evaluation inputs are absent from the
selection suite. This is a direct integration replay of that cohort.

## Functional observations

| Route | Attempt budget | Complete requests | Passed finite cases | Evaluated bodies | Equal-descriptor skips |
| --- | ---: | ---: | ---: | ---: | ---: |
| Deterministic | 1 | 32/64 | 264/512 | 64 | 0 |
| Model | 1 | 52/64 | 418/512 | 64 | 0 |
| Deterministic | 8 | 64/64 | 512/512 | 96 | 0 |
| Model + exact descriptor reuse | 8 | 64/64 | 512/512 | 76 | 24 |

At budget 1, the learned route completes 28/32 English and 24/32 Korean requests;
the deterministic control completes 16/32 in each language. With budget 8 both
routes complete all listed requests. The model and reuse together evaluate 20.8%
fewer bodies. The earlier standalone comparison includes a deterministic reuse
control with unchanged results; this native collection retains the ordinary
compiler deterministic route.

All 256 selected masks, evaluated-body counts and native outcomes match their
frozen standalone observations. Every recorded model prediction matches the
original complete score/ranking arrays; original source, plan, intent and weights
are bound by digests. One-shot failures remain in the raw files. Across all four
arms the finite total is **1,706/2,048**, counted once per generated result after
the two compiled executions agree. Repeated execution is not counted as a new
independent test input. These counts describe the listed inputs and supported
two-update bodies.

## Measured cost

Budget-8 medians, 64 cold CLI processes per route:

| Measure | Deterministic | Model + reuse |
| --- | ---: | ---: |
| Prediction only | 0 | 2.75 µs |
| Compiler's internal codegen interval | 0.702 ms | 1.342 ms |
| Generation command wall time | 8.579 ms | 9.256 ms |
| Generation command CPU time | 7.579 ms | 8.628 ms |
| Generation command one-core CPU percentage | 88.14% | 92.50% |
| Generation command maximum RSS | 17.56 MiB | 18.60 MiB |
| Native build + two executions | 309.784 ms | 311.233 ms |

Whole collection: 88.13 s wall, 35.93 s user + 27.59 s system CPU, 84 MiB maximum
RSS as reported by `/usr/bin/time -l`. Per-command CPU comes from OS-reported
process resource usage, expressed against that command's wall time; it is not
global machine utilization. Native command cost includes compilation. This is a
cold-process observation, separate from warm retained-model concurrency tests.
No request timed out in this cohort; four simultaneous retained-model callers
and cancellation also passed focused race coverage. Broader workload deadlock
freedom has no measured claim here.

Fewer evaluations have not yet reduced complete generation latency: the observed
median increases by about 0.68 ms. Candidate preparation, model loading and receipt
construction remain larger costs than prediction. Reusing prepared candidates is
the next bounded performance experiment; the initial weights remain fixed.

## Exact sources and publication state

- Compiler: [`9d570ab1def810137cf789384548b3bc4c0d65bd`](https://github.com/kimjooyoon/meta-ontology-go/commit/9d570ab1def810137cf789384548b3bc4c0d65bd), [PR #1176](https://github.com/kimjooyoon/meta-ontology-go/pull/1176).
- Collector: [`4cbe48c714189f87babfbb2c547cbfbe52aa9e5d`](https://github.com/kimjooyoon/gooo-neural-decision-experiments/commit/4cbe48c714189f87babfbb2c547cbfbe52aa9e5d), `cmd/order-judge-native`.
- Protocol committed before collection: [`2e3d1210f19306b290a4e671027564ff94f37a49`](https://github.com/kimjooyoon/gooo-neural-decision-experiments/commit/2e3d1210f19306b290a4e671027564ff94f37a49).
- Go SDK: [v0.2.19-experimental](https://github.com/kimjooyoon/gooo-decision-runtime/releases/tag/v0.2.19-experimental), [cross-platform replay](../order-judge-sdk-20261003).
- Model: [gooo-order-judge-tiny-v1](https://huggingface.co/asketeddy/gooo-order-judge-tiny-v1), original weights SHA256 `cf00ccc83d17d28ed73fcb869366151a48ffccd3aa8ca8e635aabf19810b9e78`.

This collection uses the exact feature branch above. Main promotion and installed
compiler checks are recorded separately when completed. Model sampling, batch
feedback and bodies outside the declared profile return explicit errors; omit
`--path-model` to use ordinary deterministic generation.

The first compiler CI run [37098811528](https://github.com/kimjooyoon/meta-ontology-go/actions/runs/37098811528)
found two obsolete v0.2.18 checksums during module tidying. Follow-up commit
`9f5d936f` removes those two lines from `go.sum`; runtime code and the selected
v0.2.19 module are unchanged. The original collection stays bound to `9d570ab1`.

## Files and reproduction

`native.zip` contains all original source/recipe/case files, generation and native
receipts, ordered progress journal, records, summary and manifest. `costs.json`
is derived with `costs.jq`. The collector independently recomputes the authored
arithmetic reference before comparing native results. The additional
`verification.go.txt` checker verifies all 1,024 source/emitted/generation/runtime
digests and reconciles native case totals; its output is `verification.log`.
No local home paths, access tokens or private messages are included.

```sh
unzip -q native.zip -d /tmp/gooo-order-native
jq -f costs.jq /tmp/gooo-order-native/records.json
cp verification.go.txt /tmp/verify-order-native.go
go run /tmp/verify-order-native.go /tmp/gooo-order-native
```

For a fresh native run, build the pinned compiler and collector with Go1.27.1,
decode `initial.zip` and `runtime.zip` from the earlier initial evidence bundle,
and pass their directories, the fixed model, compiler/collector revisions and
a fresh output directory to `order-judge-native`. The source-bound recipe goes
into `body-codegen --path-plan ... --path-model ...`; `body-execute` receives that
same source, recipe and generated result. No externally selected body or seed is
substituted for model inference.

### 한국어 요약

자체 16 KiB 모델을 컴파일러 안에 직접 연결했습니다. 한 번 시도로 완성한 요청은
32/64에서 52/64로 늘었고, 최대 여덟 번 시도하면 양쪽 모두 64/64를 완성했습니다.
실제 후보 평가는 96회에서 76회로 줄었지만 생성 시간 중앙값은 8.58ms에서 9.26ms로
늘었습니다. 추가 학습은 0회이며, 다음 작업은 후보 준비·로딩·기록 비용을 줄이는 것입니다.
모든 실패와 비용을 함께 공개하고, 한영의 작은 정수 조립 범위부터 개선합니다.
