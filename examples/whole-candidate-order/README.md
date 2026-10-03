# 한국어 지시로 두 연산의 순서를 조립하기

원본 바디는 1을 더하고 2를 곱합니다. 조립 계획에는 “2를 곱한다. 이어서 1을
더한다”는 지시와 세 입출력 예시를 둡니다. 자체 16 KiB 모델이 허용된 여덟
후보의 순위를 정하고, 컴파일러가 예시를 만족하는 바디를 Go로 생성합니다.

이 파일들은 공개 네이티브 실험의 `new-template-add-multiply-ko-s0-w1-t2` 과제에서
그대로 가져왔습니다. [256회 생성·512회 실행의 전체 관측](../../publication/order-judge-native-20261003).

## 준비

Go1.27.1과 [main `8c6ec01c`](https://github.com/kimjooyoon/meta-ontology-go/tree/8c6ec01c4931186460244f3a2975013edda62325)의
컴파일러가 필요합니다. 컴파일러 저장소에서 `go build -o gooo ./cmd/gooo`로 한 번
빌드한 `gooo`와 Go1.27.1을 PATH에 두고, 이 연구 저장소의 루트에서 실행합니다.
모델은 저장소에 공개된 최초 가중치를 사용합니다.

## 한 명령으로 조립하고 실행하기

아래 Go 예제는 소스·레시피·기대값을 읽고 `gooo body-path-stream`에 보냅니다.
응답 하나가 도착하면 바로 `body-execute`로 컴파일하고 실행한 뒤 다음 요청을 보냅니다.
기본 두 요청 동안 입력을 열어 두어 후보 준비를 재사용합니다.

```sh
out_root="$(mktemp -d)"
go run ./cmd/order-example \
  --model publication/order-judge-initial-20261003/model.json \
  --out "$out_root/model"

go run ./cmd/order-example --out "$out_root/deterministic"
```

모델 옵션이 없는 두 번째 명령은 결정론적으로 진행합니다. 각 요청의 응답 시간,
실행 시간, 실제 모델 호출 수와 기대값 충족 수가 표시됩니다. 출력 폴더에는 입력 파일,
`order-1-response.json`, `order-1-generation.json`, `order-1-runtime.json`과
전체 `summary.json`이 남습니다. 생성 파일의 `source`에서 실제 조립한 Go를 읽습니다.
새 출력 폴더를 사용하고, 기대값이 미충족이면 기록을 남긴 뒤 오류로 종료합니다.
이 예제는 `Compose` 활동의 한 소스를 순차 반복하는 실행법입니다.

첫 응답 시간에는 시작 중인 워커의 준비 비용이 포함될 수 있습니다. 실행 시간은
별도 명령의 컴파일과 두 번 실행을 포함합니다. 이 두 요청은 실행법 확인이며
처리량 측정과 일반화 평가는 별도 실험에서 다룹니다. Linux·macOS에서는 취소 시
자식 프로세스 그룹을 함께 종료합니다. 전체 예제의 제한 시간은 90초입니다.

[로컬·Linux의 원본, 실제 응답 시간과 실행 비용](../../publication/order-example-immediate-20261003)을
공개했습니다. 두 환경 각각 4회 생성·8회 실제 실행·32/32 기대값을 확인했습니다.

## 생성과 실행을 나눠 보기

새 컴파일러의 `body-path-stream --execute`를 쓰면 실행파일 한 개도 보관할 수 있습니다.
앞 명령에 `--retain-native`를 추가합니다. 첫 요청은 빌드하고, 같은 바디의 다음
요청은 원본과 기대값을 새로 검증한 뒤 현재 입력을 두 번 실행합니다.
`native_artifact_reused`가 실제 재사용 여부를 보여줍니다. 이 모드의 `response_ms`는
생성과 실행을 모두 포함한 왕복 시간이고, `execution_ms`는 그 안의 실행 관측 시간입니다.
두 값을 더하지 않습니다. 개별 생성·빌드·실행 시간과 실제 출력은 보관한 원본에서
읽을 수 있습니다. 모델을 빼면 같은 실행 구조를 결정론적으로 사용할 수 있습니다.
[병합·설치 확인과 전체 비용](../../publication/retained-native-execution-20261003).

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
