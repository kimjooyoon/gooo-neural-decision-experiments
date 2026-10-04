# Installed native record values

Observed from clean installed compiler main
[`b629a664ea4a667945ab5f894557f42e97cafa35`](https://github.com/kimjooyoon/meta-ontology-go/commit/b629a664ea4a667945ab5f894557f42e97cafa35),
Go1.27.1 and SDK v0.2.21-experimental. Development PR1223 and main PR1224 merged
after their own complete CI and independent proof verification. `installation.json`
retains both installed binary digests; `compiler-proof.json` retains the exact
promotion tuple and downloaded proof digest. Local binary paths are omitted.

## Same declared graph after installation

This is a new twelve-row cohort, separate from the [candidate cohort](../README.md).
It met 432/432 named expectations, checked 648 input slots, 432 producer deliveries
and 1,152 field observations across 24 native executions. Three fresh model
requests made three actual predictions. Saved controls made zero new predictions.
Selected Gooo source, generated Go and driver hashes match the candidate for
every corresponding row. The own model ranks Score; record bodies follow Gooo.

| Fresh request median | Own model | Deterministic |
| --- | ---: | ---: |
| Graph generation | 13.167ms | 12.010ms |
| Runtime/build observation | 327.485ms | 323.884ms |
| Whole command | 352.606ms | 347.405ms |
| User CPU | 0.15s | 0.15s |
| System CPU | 0.11s | 0.11s |
| Maximum observed resident memory | 83.42MiB | 83.44MiB |

Actual prediction median was 29,666ns (29.7µs). Resources include the native build
and child processes. Fixed order and warm build caches limit timing comparisons.
The model checked two candidates against eight deterministic candidates, while
whole generation cost remained a little higher with the model.

## Separate controls

- Concurrent fresh requests: 72/72 outputs; approximately 366ms and 503ms wall
  times. Both finished. This finite pair is retained separately from serial timing.
- Eight further runtime-only cases through each saved composition: 96/96 outputs,
  256 field observations, zero new predictions.
- One changed state expectation: 35/36 named outputs and 35/36 expected record
  output fields. The actual `ready` and expected `wait` remain visible.
- Previous installed scalar-join saved compositions: 49/49 per mode, zero new
  predictions. Historical generation source stays attached to its original
  receipts; execution uses the new installed compiler.
- A second graph takes a caller's complete `Item`, copies it to a local, replaces
  its state with `state + "!"`, and passes it to a label activity. Four cases met
  8/8 named outputs and 8/8 output-record fields in fresh construction and saved
  replay. Each has 24 actual input/output field observations; no inference is used.

## Read functional completeness quickly

From the research repository root with Go1.27.1:

```sh
go run ./cmd/record-completeness \
  --input publication/native-record-values-20261004/installed/partial.json
```

The result shows 35/36 (97.22%) and identifies `Propose.state`, `ready` versus
`wait`. `--json` provides case/activity/field gaps. `partial-completeness.json`
is the actual generated summary; `completeness.json` reports 36/36 on the complete
case file. Output field ratios use caller expectations for record outputs;
input field delivery observations have their own trace counts.

`unscored-records.json` is an actual saved replay with all three record-output
expectations omitted. It met 18/18 scalar named expectations. The result reader
reports record output fields 0/0, percentage `null`, 36 unobserved fields and 18
unscored outputs. This keeps missing expectations visible instead of adding
match credit. The cases retain exact int64 boundaries.

The second graph can be run directly with the installed compiler:

```sh
gooo body-compose \
  --source publication/native-record-values-20261004/installed/external-source.gooo.fixture \
  --cases publication/native-record-values-20261004/installed/external-cases.json
```

The finite counts describe these declared cases and profile. Required string
record fields, source-bound constructs and the separate scalar assembly are
the supported scope. Record-valued learned assembly, richer field profiles and
cross-invocation compiled artifact reuse remain further language work.

## 한국어 요약

main `b629a664`로 명령 도구와 실행기를 설치한 뒤 실제 관측을 다시 남겼습니다.
열두 반복 제어에서 출력 432/432와 필드 관측 1,152개를 확인했고, 후보와 같은
소스·Go·연결 코드 지문을 유지했습니다. 이전 입력 연결의 저장 결과도 두
경로에서 49/49씩 재실행했습니다. 새 기록 입력 예제는 지역값 전체 교체와
다음 활동의 실제 필드 읽기를 확인합니다.

결과 요약은 기대 출력과 기록 출력 필드의 분모를 따로 보여줍니다. 상태 기대값
하나가 다르면 35/36과 `Propose.state`를 안내하고, 기록 기대값을 생략한 실제
실행은 해당 필드 36개를 미관측으로 표시합니다. 모델 추론 중앙값은 29.7μs,
전체 생성은 13.17ms·결정론 12.01ms, 명령 전체는 약 353ms·347ms였습니다.
이번에 모델을 새로 학습하지 않았으며, 정수 경로의 순위를 제안하는 기존
가중치를 사용했습니다.
