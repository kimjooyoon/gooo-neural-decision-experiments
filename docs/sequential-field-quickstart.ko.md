# 필드를 순서대로 만들고 실행하기

2026-10-05. 사용할 컴파일러 소스는 `a5f6e8c668ff9fc7cfe3367a5019b50fbcadc1fa`, Go1.27.1·SDKv0.2.23입니다.

제목을 고르고 상태를 바꾼 뒤, 바뀐 제목과 상태로 설명을 만듭니다.
작은 자체 판단기는 세 필드의 두 후보식을 읽어 여덟 조합의 시도 순서를
제안합니다. 조건·현재 값·저장값·타입은 Gooo 본문의 의미대로 실행됩니다.

## 1. 소스와 실행 파일 준비

새 체크아웃의 루트에서 실행합니다.

```sh
git clone https://github.com/kimjooyoon/meta-ontology-go.git
cd meta-ontology-go
git switch --detach a5f6e8c668ff9fc7cfe3367a5019b50fbcadc1fa
gooo_go_bin="$(GOTOOLCHAIN=go1.27.1 go env GOROOT)/bin/go"
GOWORK=off "$gooo_go_bin" build -trimpath -o gooo ./cmd/gooo
./gooo version --build
```

## 2. 작은 모델 받기

`hf`가 설치된 환경에서 아래 두 파일을 받습니다. 동일한 공개 가중치를
계속 사용합니다. 다운로드가 끝나면 추론·생성·실행은 로컬에서 진행됩니다.

```sh
hf download asketeddy/gooo-record-shared-field-tiny-v1 \
  models/qat_ternary/model.json models/qat_ternary/weights.bin \
  --revision e7a532589ccc062534863bf4f9c935181d9ca439 \
  --local-dir ./gooo-field-model
```

가중치 파일은446 B, 실행 텐서는2,096 B·scale8 B입니다. 컴파일러와
빌드 도구를 포함하는 전체 프로세스 RAM은 별도로 측정합니다.

## 3. 조립한 코드를 바로 실행

```sh
./gooo body-compose \
  --source examples/body-codegen/record-field-updates.gooo.fixture \
  --cases examples/body-codegen/record-field-updates-cases.json \
  --model ./gooo-field-model/models/qat_ternary/model.json \
  --out /tmp/gooo-sequential-fields > /tmp/gooo-sequential-fields.json
```

출력 폴더는 새 경로여야 합니다. 이미 사용한 경로라면 새 이름을 지정합니다.
`--model`을 생략하면 같은 후보식을 정해진 순서로 평가합니다.
예제는 여덟 후보까지 시도하므로 두 방식 모두 아래 유한 사례를 충족합니다.

| 결과 | 설치본에서 확인한 분자·분모 |
| --- | ---: |
| 조립할 때 제공한 레코드 필드 | 15/15 |
| 실행할 때 제공한 Select·Label 출력 | 14/14 |
| 실행할 때 제공한 레코드 필드 | 21/21 |
| 생성 단계의 모델 호출 | QAT1회·결정론적0회 |
| 생성된 프로그램 실행의 새 모델 호출 | 0회 |

선택 사례는5개이며 별도 실행 입력은7개입니다. 두 활동을 연결해 실행합니다.
모델 확률과 실제로 충족한 기능의 비율은 각각 읽습니다. 이번 한 예제의 목표는
세 대안식을 모두 고르는 조합입니다. 새로운 요구의 판단률은 별도 혼합 요구·
새 표현 실험을 참고합니다.

## 4. 긴 결과를 읽기

연구 레포의 루트에서 요약 도구를 실행할 수 있습니다.

```sh
GOWORK=off GOTOOLCHAIN=go1.27.1 go run ./cmd/record-completeness \
  --input /tmp/gooo-sequential-fields.json
```

```text
Selection Select: cases 5/5 (100.00%); fields 15/15 (100.00%); evaluated candidates: 1
Named outputs: 14/14 (100.00%); unobserved: 0
Record output fields: 21/21 (100.00%); unobserved: 0
```

위 후보 평가 횟수1은 QAT 관측입니다. 결정론적 실행의 평가 횟수는 따로 출력합니다.
예산을 줄여 부분 결과가 남으면 실제 값과 기대값, 미충족 필드가 표시됩니다.

## 5. 저장한 선택 다시 실행

컴파일러 체크아웃에서 실행합니다.

```sh
./gooo body-compose --source /tmp/gooo-sequential-fields/original.gooo \
  --cases /tmp/gooo-sequential-fields/cases.json \
  --composition /tmp/gooo-sequential-fields/composition.json
```

원래 선택을 재구성해 실행하며 새 추론은0회입니다. 생성 기록의 이전 호출 수는
원래 관측으로 남습니다. 출력 폴더의 `realized.gooo`에서 선택한 본문을 읽을 수 있습니다.

## 본문을 작성할 때

```gooo
let copy = input0
let saved = copy
copy.state = "ready"
copy.reason = saved.reason + ":" + copy.state
return copy
```

`saved`는 복사 시점의 값을 유지하고 다음 문장은 갱신된 `copy.state`를 읽습니다.
`field_update at "0"`은 첫 필드 대입의 오른쪽 식을 고릅니다. 생성자의
`field_value` 번호는 별도로 셉니다. 현재는 기존 지역 레코드의 required Text
필드를 갱신할 수 있습니다. 함수 입력은 읽기 전용입니다.

[전체 관측·실패·비용](../publication/record-field-updates-20261005)
· [다음 값 출처 실험 계획](local-value-origin-plan.ko.md)
· [언어 문법](https://github.com/kimjooyoon/meta-ontology-go/blob/a5f6e8c668ff9fc7cfe3367a5019b50fbcadc1fa/docs/record-field-updates.md)
