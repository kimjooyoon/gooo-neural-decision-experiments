---
license: mit
language:
- ko
- en
tags:
- gooo
- metaprogramming
- golang
- ternary
- local-inference
---

# Gooo shared record field judge v1

## 순차 필드 조립으로 확장 중

같은 공개 가중치를 사용해 지역 레코드의 필드를 하나씩 갱신하는
[새 실험](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/record-field-updates-20261005)을
실행했습니다. 앞에서 바꾼 상태를 다음 설명 필드가 읽고,저장한 레코드와
문자열 값을 사용하는 여섯 코드 형태입니다. Gooo의`field_update` 선택을
연결한 [PR1233](https://github.com/kimjooyoon/meta-ontology-go/pull/1233)이 병합 검사 중입니다.

72개 유효한 그래프·144회 실행에서 QAT는 첫 후보로 필드144/144개와
활동 출력96/96개를 맞췄습니다(각 시도 예산의12개 그래프).
모든 목표가 같은 mask7이므로 이 수치에는 대안식을 선호하는 편향이
작용할 수 있습니다. 학습0회,추론 중앙값17.50~18.46µs,전체 명령은약310ms입니다.
초기24개 미사용 변수 오류와 수정 계획을 포함한 원본을
[여기](experiments/record-field-updates-20261005)에서 읽을 수 있습니다.
현재 입력 배열에서는`copy.state`와`saved.state`가 동일해 실제 모델 응답도
같습니다. 다음 개선은 지역 변수의 현재 정의와 값의 출처를 표현하는 작업입니다.

Gooo에서 제목·상태·사유 필드에 들어갈 두 후보식을 준비하고, 작은 자체 모델이
요구 문장을 읽어 조립 순서를 고르는 실험입니다. 같은 판단기를 필드 세 곳에
쓰고, 그 점수를 합쳐 여덟 코드 경로를 정렬합니다. 설계도에서 부품을 하나씩
고른 뒤 실제로 조립해 보는 방식입니다. 조건문·변수·필드의 형태와 실행은
Gooo 컴파일러가 담당하며, 예시에서 맞은 기능의 비율을 별도로 기록합니다.

## 같은 자료에서 관측한 결과

| 모델 | 다른 본문·기본 표현: 첫 완전 조합 | 다른 본문·새 표현 | 본문 평가의 맞은 필드 | 가중치 파일 |
| --- | ---: | ---: | ---: | ---: |
| 앞선 dense v2 FP32 | 424/1,536 (27.60%) | 81/512 (15.82%) | 2,942/4,608 (63.85%) | 74,624 B |
| 공유 FP32 | 1,152/1,536 (75.00%) | 120/512 (23.44%) | 4,224/4,608 (91.67%) | 8,288 B |
| 공유 PTQ 삼진화 | 168/1,536 (10.94%) | 88/512 (17.19%) | 2,304/4,608 (50.00%) | 446 B |
| 공유 QAT 삼진 학습 | 1,344/1,536 (87.50%) | 102/512 (19.92%) | 4,416/4,608 (95.83%) | 446 B |

세 필드와 여덟 작성된 본문/후보식 모양을 재사용한 벤치마크입니다. 모든 여덟
요구 조합은 학습에 포함됩니다. 새 표현에서는 첫 조합 비율이 크게 떨어집니다.
직접 삼진화한 PTQ의 품질도 낮습니다. 본문·표현·필드 지표를 함께 읽어야
현재 어느 부분까지 조립 가능한지 알 수 있습니다.

FP32의 문맥 해석을 포함한 따뜻한 추론 중앙값은9.50µs, 앞선 dense 모델은
같은 실행에서15.96µs였습니다. 미리 만든 특징 배열 한 행의 계산은3.04µs,
QAT는3.13µs였고 준비된 배열의 FP32/QAT 계산은 호출당 힙 할당0회였습니다. 배열 계산은
텍스트/식 해석을 제외한 측정입니다. 전체 코드 생성 속도는 별도로 측정합니다.
삼진 파일은 행렬 원소를 다섯 trit씩 묶으며 FP32 bias도 포함합니다. 실제 실행은
int8 배열2,096 B와 scale8 B, 요청별 workspace3,200 B 등을 사용합니다.
파일의1.58bit 수준 표현과 전체 프로세스 RAM은 서로 다른 크기입니다.

## 사용

SDK **v0.2.23-experimental**, 모델 특징
`triple_record_field_context_v1_shared_v1`, 산술 `float32_separate_v1`입니다.
[개발 PR1231](https://github.com/kimjooyoon/meta-ontology-go/pull/1231)과
[main PR1232](https://github.com/kimjooyoon/meta-ontology-go/pull/1232)을 각각 전체12개CI와
독립 증거 확인 후 병합했습니다. 깨끗한 main `2c807d461afc68a7ebfbd277d1dd6e97f5168e92`를 Go1.27.1로
빌드·설치하고 공개 가중치로 실제 실행을 확인했습니다.
예제의 후보 예산은8이며, 입력을 전달하는 두 활동을 실제로 생성·실행합니다.

```sh
hf download asketeddy/gooo-record-shared-field-tiny-v1 \
  models/qat_ternary/model.json models/qat_ternary/weights.bin \
  demo.gooo.fixture demo-cases.json \
  --revision 87f8c482d23dc0eaa41f5575f652e7ca61f44280 \
  --local-dir ./gooo-field-model

gooo body-compose --source ./gooo-field-model/demo.gooo.fixture \
  --model ./gooo-field-model/models/qat_ternary/model.json \
  --cases ./gooo-field-model/demo-cases.json
```

`--model`을 생략하면 결정론적 후보 순서로 조립합니다. 모델이 있으면 한 번의
추론으로 후보를 정렬하고, Gooo의 유한 테스트 결과로 이어서 선택합니다.
완성된 코드 실행과 저장한 선택의 재구성에는 새 모델 호출이 없습니다.
제공한 문맥의 필드 수나 크기가 맞지 않으면 이유를 남기고 결정론적으로
진행합니다. 필드용 모델을 scalar 본문에 지정해도 이 방식으로 진행합니다.

Go SDK의 `LoadRecordSharedThree`와 `PredictRecordSharedInto`로 같은 모델을
읽을 수 있습니다. `PredictRecordSharedFeaturesInto`는 미리 준비한
`[768]float32` 배열을 받아 텍스트 해석 비용을 분리합니다.
`RecordChoiceMarginals`는 각 필드의 첫/둘째 후보 확률을 반환합니다.
요청마다 workspace/output을 소유하고 가중치를 함께 읽습니다.

## 실제 조립과 실행

동일한24개 소스에서 결정론적 순서, 앞선 v2, 세 공유 모델을 후보 예산1/2/8로
비교했습니다.360개 그래프와720회 네이티브 실행을 마쳤고 원시 출력에서 지표를
다시 계산해 모든 요약과 일치함을 확인했습니다. 예산8의120개 그래프는 모두
선택 예시15/15 필드, 새 런타임 입력12/12 필드와8/8 이름 붙은 출력이 맞았습니다.

| 후보 하나일 때, 실제 변환 필드 | 결정론적 | 앞선 v2 | 공유 FP32 | 공유 QAT |
| --- | ---: | ---: | ---: | ---: |
| 활성 입력의 요구 필드 | 72/144 | 108/144 | 126/144 | 126/144 |
| 입력과 다른 값이 필요한 필드 | 60/120 | 92/120 | 108/120 | 106/120 |

평가 축을 나누면 기본 표현의 활성 필드는 앞선 v2 76/96개, 공유 FP32/QAT
96/96개였습니다. 새 표현8개 소스는 앞선 v2 32/48개, 공유 모델30/48개로
줄었습니다. 전체 합계의 개선과 새 표현 묶음의 퇴행을 함께 기록합니다.

유지 분기의144/144 필드는 별도로 계산했습니다. 전체 명령 중앙값은
결정론적395.80ms, 공유 FP32 394.65ms, QAT 395.30ms였습니다. 생성·빌드·실행
시작 비용을 포함하며 고정된 순서로 측정했습니다. 작은 추론이 빨라진 관측만으로
전체 코드 생성의 속도 향상을 판단하기 어렵습니다. 런타임과 선택 재구성의
추가 모델 호출은0이며, 생성 요청당 실제 모델 호출은1입니다.

`evidence.zip`은 새 실험 원시자료37.91MiB를 압축해 담습니다. 기존 소스/특징
뱅크는 v2의 고정 공개 아카이브를 재사용합니다. `audit.json`,
`native-summary.json`과 `SHA256SUMS`에서 지표와 바이트 근거를 읽을 수 있습니다.
[연구 CI37226752241](https://github.com/kimjooyoon/gooo-neural-decision-experiments/actions/runs/37226752241)은
전체26개 작업을 통과했습니다. Linux의360개 그래프·720회 실행 원본을 내려받아
소스·후보 선택·필드·값 전달·실제 출력·프로세스 완료를 독립 재계산했습니다.
고정120개 NumPy/Go 비교의 최대 logit 차이도0이며, 모든 모델의 품질 수가
macOS 관측과 같습니다. `validation/linux-validation.json`에 원본 지문을 기록합니다.

최초 로컬 원본은39,748,306 B로 계획한64MiB 이내였습니다. 후속 Linux 재생의
논리적 원본40,178,256 B를 합치면76.22MiB로 그 목표를 넘습니다. 두 관측을
유지하고 범위를 공개합니다. 로컬에는Linux 자료의8MB압축 파일을 보관해
메모리에서 검산하며 별도 압축 해제 복사본을 만들지 않았습니다.

## 학습과 근거

- 계획/학습 소스: [862dd0e](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/862dd0ee6ca3e9e4391b651f510d0bd83f0b2f69).
- 상세 계획: [shared field pilot](https://github.com/kimjooyoon/gooo-neural-decision-experiments/blob/862dd0ee6ca3e9e4391b651f510d0bd83f0b2f69/docs/record-shared-field-design-20261005.md).
- 원본7,168개 소스/특징: [앞선 v2 공개 자료](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/05db7d6e13e66259d2d116ef5c97d36216b04c4b/publication/paired-field-intents-20261005).
- 새 Go 초기값 seed20261055,2,072개 매개변수. 같은40 FP32+40 QAT epoch,
  총3,840회 갱신. 학습3,072/보정1,536; 평가1,536/512/512. 보정만으로
  checkpoint/temperature를 선택했으며 진단 후 재학습하지 않았습니다.
- Go가 소스·특징·추론·실행·지표를 담당하고 Python은 오프라인 MPS
  최적화/내보내기에 사용했습니다. 독립 NumPy 산술과 Go의 고정120개
  출력 비교에서 최대 logit 차이는0이었습니다.
- 학습 프로세스9.45초, CPU5.06초(한 코어 기준 평균53.5%), 최고 RAM
  477.9MiB. 학습 루프7.24초, MPS tensor13.56MiB/driver50.72MiB를
  표본 관측했습니다. GPU utilization과 호스트 전체 CPU 증가량은 미측정입니다.

필드별 독립 점수라는 구조가 이 작은 과제의 학습을 도왔습니다. 필드 사이의
의존 관계, 새 자연어 표현과 더 다양한 타입·본문은 후속 언어 연구에 필요합니다.
확률은 후보 순서를 정하는 값이며 테스트에서 확인한 기능 완전성은 실행
관측으로 계산합니다. 현재 한 번에 높은 정답률을 요구하기보다 남은 기능을
측정하고 다음 후보로 진행하는 사용 흐름을 발전시키고 있습니다.

## 공개 모델을 개발 중에 실제 사용한 기록

공개 리비전`87f8c482`의21개 파일을 원본과 바이트 단위로 확인했습니다.
개발 후보`d515db9d`에서 두 작업자의32개 요청을4회 관측했습니다. 매회
선택 예시의 필드480/480개가 맞았고 첫 응답은 입력을 닫기 전에 도착했습니다.
첫 실행은497.015ms, 세 반복은8.752/12.342/11.866ms였습니다.
각 반복은 작업자 시작과 문맥 해석·조립을 포함합니다. 최초 실행도 보존합니다.
메모리는약22.6~22.9MiB, 프로세스CPU시간은0.114~0.158초였습니다.
호스트 전체CPU 증가량과 일반적인 처리량은 미측정입니다.

설치한 main`2c807d461afc68a7ebfbd277d1dd6e97f5168e92`에서도 같은 공개 파일과 여섯 실제 실행 대조군,
32개 혼합 요구의 작업자 요청을 확인했습니다. 생성된 프로그램 실행의
추가 모델 호출은0회입니다. 원본 보고서는`validation/worker-trials`와
`validation/installed-public-dogfood.json`에 있습니다. 데드락 검토의 범위는
각60초 제한에서 두 작업자의32개 고정 요청을 완료하는 관측입니다.

Go 독자는`go run validation/read-model.go --text MODEL.json CONTEXT.json`으로
필드 이름·의도·두 후보식·확률과 제안식을 읽을 수 있습니다. 예제의 제목은
50:50동률, 상태는약99.9%로 관측했습니다. 동률의 첫 후보도 실제 유한 예제를
통과할 수 있으므로 후보 확률과 기능의 완전성을 따로 읽습니다.
[짧은 사용 흐름](https://github.com/kimjooyoon/gooo-neural-decision-experiments/blob/main/docs/shared-field-quickstart.ko.md)
· [main CI37228772888](https://github.com/kimjooyoon/meta-ontology-go/actions/runs/37228772888).

## 참고한 원리와 연구

[Laya](https://huggingface.co/convaiinnovations/laya)는 상태와 타입이 있는 질문을
받아 선택과 확률을 반환하는 접근을 보여줍니다. Gooo에서는 이를 실제 후보식에
대한 작은 판단 질문으로 좁혀 실험합니다. 이번 가중치는Go의 새로운 초기값과
Gooo에서 만든 자료로 학습했습니다.

[Ma 등의 BitNet b1.58 연구](https://arxiv.org/abs/2402.17764)는 가중치의
삼진 표현`{-1,0,1}`과 그에 맞는 학습을 다룹니다. 우리의 작은 판단기에서는
PTQ와QAT를 따로 관측하고 파일 크기·실제 텐서·전체RAM·기능 충족률을
각각 계산합니다.

[W3C PROV-O](https://www.w3.org/TR/prov-o/)의 개체·활동·사용·생성 관계는
Gooo가 원본, 생성 과정과 관측 결과를 연결하는 어휘의 바탕입니다. 소스와
선택의 이유, 실제로 전달한 값이 같은 기록에 남도록 언어를 발전시키고 있습니다.

## 설치본의 반복 관측

설치본의 첫 응답은483.816ms, 세 반복은12.164~36.625ms, 작업자RAM은22.53~23.27MiB였습니다. 모든4회에서32요청·선택 필드480/480개가 맞고 입력 종료 전에 첫 응답이 도착했습니다.

개발 후보의 반복8.8~12.3ms와 설치본의 반복을 각각 보존합니다.
가장 빠른 값만 고르지 않고36.6ms 반복도 포함합니다. `validation/installed-worker-summary.json`은
각 요청 묶음의프로세스CPU초/전체벽시간×100을 한 코어 기준 평균으로
계산합니다. 두 작업자와 런타임의 합계이므로100%를 넘을 수 있습니다.
호스트 전체CPU 증가량과GPU utilization은 미측정입니다.
