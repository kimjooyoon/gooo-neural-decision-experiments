# 한국어 지시로 두 연산의 순서를 조립하기

원본 바디는 1을 더하고 2를 곱합니다. 조립 계획에는 “2를 곱한다. 이어서 1을
더한다”는 지시와 세 입출력 예시를 둡니다. 자체 16 KiB 모델이 허용된 여덟
후보의 순위를 정하고, 컴파일러가 예시를 만족하는 바디를 Go로 생성합니다.

이 파일들은 공개 네이티브 실험의 `new-template-add-multiply-ko-s0-w1-t2` 과제에서
그대로 가져왔습니다. [256회 생성·512회 실행의 전체 관측](../../publication/order-judge-native-20261003).

## 실행

Go1.27.1과 [main `cb2892cb`](https://github.com/kimjooyoon/meta-ontology-go/tree/cb2892cb583e693190bd68f75eb4df223819f053)의
컴파일러가 필요합니다. 컴파일러 저장소에서 `go build -o gooo ./cmd/gooo`로 한 번
빌드한 `gooo`와 Go1.27.1을 PATH에 두고, 이 연구 저장소의 루트에서 실행합니다.
모델은 저장소에 공개된 최초 가중치를 사용합니다.

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

## 같은 프로세스에서 두 번 조립하기

위의 `gooo`로 반복 조립도 실행할 수 있습니다. 두 번째 요청의 `reused`가 `true`가
되며, 모델 판단과 현재 예시의 평가는 매번 새로 수행합니다. 연구 저장소 루트에서
실행합니다.

```sh
gooo body-path-stream --workers 1 \
  --model publication/order-judge-initial-20261003/model.json \
  < examples/whole-candidate-order/worker-requests.jsonl \
  > /tmp/gooo-order-worker-results.jsonl

jq '{id:.correlation_id,status,preparation:.response.report.body_paths.whole_candidate_preparation}' \
  /tmp/gooo-order-worker-results.jsonl

jq -s '.[1].response' /tmp/gooo-order-worker-results.jsonl > /tmp/gooo-order-generation.json
```

마지막 생성 파일에 앞 절의 `body-execute` 명령을 그대로 사용할 수 있습니다.
각 줄의 `response.source`는 생성된 Go 코드입니다. 모델은 매 요청 새로 판단하고,
워커 시작 시 준비한 가중치와 같은 계획의 후보를 재사용합니다. `--model`을 생략하면
결정론적으로 진행합니다. 결과는 준비되는 순서로 나오므로 병렬 요청은
`correlation_id`로 연결합니다. 시작 정보는 표준 오류로, 요청별 결과는 표준 출력으로
나옵니다. 전체 명령이 정상 종료해도 각 결과의 `status`를 확인해야 합니다.

[설치본의 모델·결정론 경로 확인](../../publication/body-stream-installed-main-20261003)
· [실제 워커·즉시 실행 관측](../../publication/prepared-worker-native-20261003).
