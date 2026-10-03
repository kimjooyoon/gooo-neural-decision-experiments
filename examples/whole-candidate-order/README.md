# 한국어 지시로 두 연산의 순서를 조립하기

원본 바디는 1을 더하고 2를 곱합니다. 조립 계획에는 “2를 곱한다. 이어서 1을
더한다”는 지시와 세 입출력 예시를 둡니다. 자체 16 KiB 모델이 허용된 여덟
후보의 순위를 정하고, 컴파일러가 예시를 만족하는 바디를 Go로 생성합니다.

이 파일들은 공개 네이티브 실험의 `new-template-add-multiply-ko-s0-w1-t2` 과제에서
그대로 가져왔습니다. [256회 생성·512회 실행의 전체 관측](../../publication/order-judge-native-20261003).

## 실행

Go1.27.1과 [#1176의 컴파일러](https://github.com/kimjooyoon/meta-ontology-go/pull/1176)가
필요합니다. 컴파일러의 `gooo` 실행 파일과 Go1.27.1을 PATH에 두고, 이 연구 저장소의
루트에서 실행합니다. 모델은 저장소에 공개된 최초 가중치를 사용합니다.

```sh
gooo body-codegen --json --activity Compose \
  --path-plan examples/whole-candidate-order/recipe.json \
  --path-model publication/order-judge-initial-20261003/model.json \
  examples/whole-candidate-order/source.gooo > /tmp/gooo-order-generation.json

gooo body-execute \
  --source examples/whole-candidate-order/source.gooo \
  --path-plan examples/whole-candidate-order/recipe.json \
  --generation /tmp/gooo-order-generation.json \
  --cases examples/whole-candidate-order/cases.json \
  --go-bin "$(command -v go)" > /tmp/gooo-order-runtime.json
```

생성 결과의 `source`에 실제 Go 코드가 들어 있습니다. 이 과제에서는 다음 순서로 조립됩니다.

```go
var value = input
value = value * 2
value = value + 1
return value
```

`report.body_paths.whole_candidate_judgment`에는 후보의 연산 구조, 순위와 예측 시간이,
`report.body_paths.search`에는 실제 평가한 후보와 예시 결과가 기록됩니다.
실행 결과의 `observation.cases`는 여덟 입력의 실제 값과 기대값을 보여줍니다.
예를 들어 입력 3의 기대값은 7입니다. 모델 옵션을 빼면 같은 예시와 예산 안에서
결정론적으로 조립합니다. 이 예제의 성공 범위는 선언한 연산과 유한 입력 집합입니다.
