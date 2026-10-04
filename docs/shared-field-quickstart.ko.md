# 작은 필드 모델을 연결해 Gooo 조립 결과 읽기

2026-10-05. 컴파일러 main `2c807d461afc68a7ebfbd277d1dd6e97f5168e92`, Go1.27.1,
SDKv0.2.23-experimental과 공개 모델을 사용합니다.

제목·상태·사유에 각각 두 후보식을 적어 둡니다. 작은 판단기가 한영 요구와
실제 후보식을 읽고 시도 순서를 고릅니다. 컴파일러가 조건과 변수, 타입,
입력 연결을 검사하고 Go 프로그램을 만들어 실행합니다. 설계도의 세 칸에
부품을 끼운 다음 실제 입력으로 움직여 보는 예제입니다.

## 1. 컴파일러 준비

변경이 없는 별도 체크아웃에서 빌드하면 결과에 정확한 소스 연결이 남습니다.

```sh
git clone https://github.com/kimjooyoon/meta-ontology-go.git
cd meta-ontology-go
git switch --detach 2c807d461afc68a7ebfbd277d1dd6e97f5168e92
GOWORK=off GOTOOLCHAIN=go1.27.1 go build -trimpath -o gooo ./cmd/gooo
./gooo version --build
```

소스에는 새 SDK가 정식 버전으로 들어 있습니다. 로컬 `replace` 설정은 필요하지
않습니다. [main 검사 37228772888](https://github.com/kimjooyoon/meta-ontology-go/actions/runs/37228772888)
· [병합 PR1232](https://github.com/kimjooyoon/meta-ontology-go/pull/1232).

## 2. 네 파일 받기

`hf`가 설치된 환경에서 실행합니다. 이 예제는 고정된 공개 리비전의 모델,
가중치, Gooo 예제와 새 런타임 입력/기대값을 받습니다.

```sh
hf download asketeddy/gooo-record-shared-field-tiny-v1 \
  models/qat_ternary/model.json models/qat_ternary/weights.bin \
  demo.gooo.fixture demo-cases.json \
  --revision 87f8c482d23dc0eaa41f5575f652e7ca61f44280 \
  --local-dir ./gooo-field-model
```

가중치는446 B이고 실제 실행 텐서는2,096 B·scale8 B입니다. 전체 프로세스
메모리는 컴파일러, 문맥 해석, Go 빌드와 실행에 필요한 메모리까지 포함합니다.

## 3. 생성하자마자 실행

```sh
./gooo body-compose --source ./gooo-field-model/demo.gooo.fixture \
  --model ./gooo-field-model/models/qat_ternary/model.json \
  --cases ./gooo-field-model/demo-cases.json \
  --out field-results > field-result.json
```

`field-results`는 새 출력 폴더입니다. 원본, 선택된 소스, 생성된 함수와 실제
값이 남습니다. 공개 예제에서 관측한 결과는 선택 예시5/5건·15/15필드,
새 입력의 이름 있는 출력8/8개·레코드 필드12/12개였습니다. 생성의 모델 호출은
1회, 생성된 프로그램 실행의 추가 모델 호출은0회입니다.

`--model`을 생략하면 결정론적으로 조립합니다. 예제의 `attempts "8"`은
여덟 조합까지 확인할 수 있는 예산입니다. 이 유한 예제에서 두 경로는 같은
완전성을 얻습니다. 더 작은 예산에서 부분 결과가 남으면 실제로 맞은 항목과
미충족 항목을 함께 읽습니다.

## 4. 긴 JSON을 짧게 읽기

연구 레포의 Go 도구를 사용합니다. `field-result.json`에는 위 실행의 실제
파일 경로를 지정합니다.

```sh
go run ./cmd/record-completeness --input /path/to/field-result.json
```

전체 예제의 짧은 출력:

```text
Selection Select: cases 5/5 (100.00%); fields 15/15 (100.00%); evaluated candidates: 1
Named outputs: 8/8 (100.00%); unobserved: 0
Record output fields: 12/12 (100.00%); unobserved: 0
```

별도 한 후보 결정론 관측에서는 선택 필드12/15개·실제 출력4/8개였습니다.
요약 도구는 미충족 `Select.flag`와 실제 `"wait"`, 기대 `"ready"`를 표시했습니다.
선택 예시의 분모와 실제 런타임 입력의 분모를 각각 보여줍니다. 모델 확률은
후보 순서의 힌트이고, 여기의 완전성은 제공한 사례의 실행에서 계산합니다.

## 지금 유용한 범위

한영 요구에 따라 이미 선언한 두 필드식을 고르고, 작은 예산의 미충족 기능을
기록하며 다음 후보를 이어가는 조립에 사용합니다. 세 필드의 독립 판단을
공유합니다. 모든 여덟 요구 조합은 학습에 포함됐고, 새 표현에는 약합니다.
실제 새 표현8개 소스에서는 활성 필드30/48개로 앞선 모델32/48개보다 낮았습니다.

두 작업자의32개 요청은 입력 종료를 기다리지 않고 완료됐습니다. 최초 실행
첫 응답497ms, 반복3회8.8~12.3ms, 작업자RAM22.6~22.9MiB를 함께 기록합니다.
제한된 요청 관측이며 생성 결과의 선택 필드480/480개를 확인했습니다. 실제
그래프 실행은 위의 `body-compose` 경로에서 별도로 확인합니다.

[모델·학습·전체 지표](https://huggingface.co/asketeddy/gooo-record-shared-field-tiny-v1)
· [공유 판단 설명](https://github.com/kimjooyoon/meta-ontology-go/wiki/Shared-Field-Judgment)
· [완전성 요약 도구](../cmd/record-completeness).

## 후보식을 어떤 순서로 제안했는지 읽기

컴파일러 체크아웃에서 예제의 실제 후보식 문맥을 내보냅니다. 이 단계의
모델 판단과 후보 실행은0회입니다.

```sh
./gooo body-context --activity Select \
  --feature-version triple_record_field_context_v1_shared_v1 \
  ./gooo-field-model/demo.gooo.fixture > field-context.json
```

연구 체크아웃에서 모델 파일과 위 문맥의 경로를 지정합니다.

```sh
go run ./publication/record-shared-field-20261005/validation/read-model.go \
  --text /path/to/gooo-field-model/models/qat_ternary/model.json \
  /path/to/field-context.json
```

필드 이름과 원래 의도, 두 실제 후보식, 각각의 확률과 제안식을 표시합니다.
고정된 공개 예제에서 제목 두 식은50:50으로 동률이고 상태의 `"ready"`는
약99.9%로 관측했습니다. 여기서 추가 판단은1회이며 가중치는 그대로입니다.
정답에 대한 보증 확률로 해석하지 않고, 실제 기능은 앞선 실행 결과로 읽습니다.

## 설치본에서 다시 관측한 값

설치본의 첫 응답은483.816ms, 세 반복은12.164~36.625ms, 작업자RAM은22.53~23.27MiB였습니다. 모든4회에서32요청·선택 필드480/480개가 맞고 입력 종료 전에 첫 응답이 도착했습니다.

위 개발 후보의497ms·반복8.8~12.3ms와 같은 구조의 설치본 관측을 따로
보존합니다. 새 설치본의36.6ms 반복도 포함합니다. 프로세스CPU 시간과
전체벽시간의 비율은한 코어 기준 평균이며 호스트 전체CPU 증가량은 미측정입니다.
