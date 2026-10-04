# Field-value assembly with a finite completion budget

Gooo can declare the alternatives inside a record constructor, together with
typed inputs, expected record values and an attempt budget. This study uses an
existing own small model to order those combinations, then compares each result
with deterministic search. [Language guide](language-guide.md).

A three-field candidate is like a labelled tray: title, state and reason each
have two allowed parts. The field metric shows how many spaces have their
expected value, even while the whole record still differs. The compiler emits
the selected Gooo body and ordinary Go, and a second activity reads the actual
record through a source-declared connection.

The feature is merged and installed from compiler main
`f6da667e11951b6939fc5e30442e01a09ca06e85` after PR1226 and its complete CI.
[Fresh installed controls](installed) retain all 30 candidate program identities
and finite results. Full-budget prediction median is 31.625µs, graph generation
9.890ms/9.337ms and whole command 323.462ms/324.194ms for model/deterministic
construction. The result reader shows both selection and runtime ratios plus
the concrete remaining fields. [Ternary pilot](quantized-pilot).

[Clean candidate observations](candidate) retain four budgets, three paired
trials per mode and saved full-budget replays. The complete selection reaches
15/15 fields and 5/5 cases; compiled execution reaches 21/21 record-output fields
and 14/14 named outputs. With 1/2/4 attempts, selection fields reach 40/60/80%.
The model and deterministic controls reach equal finite counts on this fixture.

The original candidate is source `48f7027584281ef87e43220f680093bc693f4488`.
Compiler [PR1225](https://github.com/kimjooyoon/meta-ontology-go/pull/1225) adds the
feature and the following Go modernizer change. Candidate measurements retain
their exact earlier revision. This cohort records work before main installation.
The follow-up source `cb316851` also keeps the syntax package independent of
model runtime dependencies and supports saved JSON indentation/key ordering.
An actual saved partial-model construction was replayed with 6/14 native outputs
and zero new predictions after reproducing the earlier formatting error.

## Reproduce and inspect

Use a clean checkout of the compiler revision supporting field assembly, then
build its `cmd/gooo` with Go 1.27.1. The observer records the revision reported by
the actual executable. From this research checkout:

```sh
go run ./cmd/record-field-observe \
  --gooo /path/to/clean/gooo --compiler-source EXACT_COMPILER_SHA \
  --source publication/record-field-assembly-20261004/candidate/source-budget-8.gooo.fixture \
  --cases publication/record-field-assembly-20261004/candidate/cases.json \
  --model "$PWD/models/own-three-feedback-v1/set-feedback/models/fp32/model.json" \
  --out /tmp/gooo-field-observations --private-out /tmp/gooo-field-resource-logs

go run ./cmd/record-field-observe \
  --verify-json publication/record-field-assembly-20261004/candidate/budget-1-deterministic-0.json \
  --compiler-source 48f7027584281ef87e43220f680093bc693f4488

go run ./cmd/record-completeness \
  --input publication/record-field-assembly-20261004/candidate/budget-1-deterministic-0.json
```

The capture command measures macOS process resources. Captured value/counter
verification works on other hosts. CI independently builds the pinned compiler,
generates both modes at all four budgets, runs the native graph and replays saved
compositions. It retains partial runtime results as well as complete results.

## What the experiment changes

The source now owns the actual field expression alternatives. The model's
initial order is followed by immediate finite selection and real compiled graph
execution. Partial candidates remain inspectable, usable bodies; field and
whole-case measurements stay separate. Saved source and replay keep the same
program without another prediction.

The model is still the existing frozen integer-choice model. The field profile
uses a separately named ordinal context containing stable field IDs, both
expressions and bilingual intent. Expected outputs and case inputs are excluded.
It makes one initial prediction; this profile has no failed-case reranking yet.
One repeated authored shape leaves cross-shape generalization unmeasured.

## Next learning target

Collect varied source-owned field decisions and the observed improvements to
each field. Train a compact field model on that explicit record-oriented
contract, with separate source shapes for evaluation. Compare fields completed
under the same attempt budget, actual attempted candidates, prediction latency
and resident tensor bytes. Records with optional/nested fields and richer field
types are further language work.

## Next use and latency experiments

The current `body-compose` runtime builds the generated Go in a fresh workspace
on every invocation, including saved-composition replay. Saving a composition
reuses the selected source and removes new model predictions; each invocation
still observes fresh native outputs. The measured whole command cost motivates
the following small steps:

1. Retain one compiled graph per local worker and reuse it when generated Go,
   driver, toolchain and target environment agree. Execute every new input and
   expected-value suite, and report the original build and current run costs.
2. Compare first build, repeated inputs, changed expectations and changed source
   with the same field/output denominators. Keep a bounded workspace and release
   it when the worker closes or is cancelled.
3. Train a field-oriented judge with source-shape splits and Korean/English
   intents. Use observed field improvements to rank the next permitted assembly.
   Compare completion at budgets 1/2/4 with prediction and execution cost.

These are planned follow-ups. The published field observations use fresh builds
and unchanged frozen weights. The Go completeness reader now also shows the
selection fields and concrete remaining field differences before runtime scores.

## 한국어

이제 Gooo의 기록 생성자 안에 필드별 후보식을 적고, 입력과 기대 출력도 같은
선언 안에 둘 수 있습니다. 제목·상태·사유를 각각 채우는 예제에서 시도 예산
1·2·4·8회에 따라 기대 필드 15개 중 6·9·12·15개를 맞췄습니다. 전체 사례는
마지막 단계에서 2/5에서 5/5로 올라갑니다. 이 차이를 보면서 부분적으로
완성된 코드도 계속 사용할 수 있습니다.

기존 자체 모델을 실제로 호출했지만 이 예제의 완성률과 시도 수는 결정론
방식과 같았습니다. 전체 예산에서 추론 중앙값은 약 29µs였고 명령 전체는
약 0.35초였습니다. 새 학습은 아직 진행하지 않았습니다. 다음 목표는 다양한
필드 문맥에서 적은 시도로 더 많은 기대 필드를 채우는 것입니다. 해석기의
선택 지표와 실제 컴파일된 프로그램의 실행 지표를 분리해서 공개합니다.
