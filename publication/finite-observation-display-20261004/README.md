# 실행하지 못한 기대값과 실제 0점 구분하기

2026-10-04 KST. 깨끗한 후보 `472db874461f0604d4eeab3faece63328d098ce9`,
[개발 #1200](https://github.com/kimjooyoon/meta-ontology-go/pull/1200)의 실제 관측입니다.
개발 필수 CI를 진행합니다. 현재 설치는 앞선 main `93fa2742`입니다.

## 사용하면서 달라지는 안내

실행 도구가 없거나 FIFO라서 실행 전에 실패하면 이전 터미널은 `0/128`로
표시했습니다. 실제 128개 출력을 관측한 뒤 기대값과 모두 달랐을 때도 같은
숫자가 보였습니다. 원래 두 실패 테스트와 원본 출력을 보존하고 표시를 구분했습니다.

| 현재 관측 | 터미널의 finite expectations |
| --- | --- |
| 유효한 128개 기대값, 관측 출력 없음 | `unobserved (128 declared)` |
| 잘못된 기대값 형식 등 runtime 관측 이전 거부 | `unobserved` |
| 실제 출력 128개와 재실행 확인, 모두 불일치 | `0/128` |
| 실제 출력 일부 또는 첫 실행 뒤 재실행 실패 | `observed 1/2 (replay incomplete)` 등 |

부분 관측과 재실행 실패의 표시는 아홉 가지 합성 상태 테스트에 포함했습니다.
이 합성 테스트를 실제 부분 native 실행으로 집계하지 않습니다. 완료된 기대값
표시, JSON의 선언 수·충족 수·원래 상태, 종료 코드, runtime v3와 source/parent
증거는 유지합니다. 표시 함수는 원래 결과를 보관한 다음 실행합니다.

## 자체 모델을 사용한 실제 확인

같은 한영 원본 두 개에서 모델·결정론 네 조건을 각 두 번 사용했습니다.
**8회 생성·4회 실제 자체 모델 판단·16회 바디 실행**에서 **1,024/1,024**와
모든 이전 생성 Go가 일치했습니다. 재사용 응답은 모델 한영 28.00·26.44ms,
결정론 24.91·25.24ms였습니다. 최초 647.68·483.71·308.38·311.70ms도 보존합니다.
속도 비교를 위한 대조 실험은 이번 자료에 포함하지 않았습니다.
추가 공구 조회 실패 한 요청은 기대값 128개 미관측·판단/실행 0회로 남겼습니다.

별도 다섯 CLI 조건은 정상 지표와 따로 읽습니다.

| 조건 | 원래 종료·상태 | 실제 판단·실행 | 관측 기대값 | 표시 |
| --- | --- | --- | --- | --- |
| 도구 없음 | 1 · execution_failed | 0 · 0 | 미관측, 선언 128 | `unobserved (128 declared)` |
| 기대값 필드 누락 | 1 · rejected | 0 · 0 | runtime 관측 없음 | `unobserved` |
| 쓰기 상대 없는 FIFO | 1 · execution_failed | 0 · 0 | 미관측, 선언 128 | `unobserved (128 declared)` |
| 기대값을 +1, 결정론 | 0 · completed | 0 · 2 | 0/128 | `0/128` |
| 기대값을 +1, 모델 | 0 · completed | 1 · 2 | 0/128 | `0/128` |

마지막 두 조건은 원래 기대값 128개에 각각 1을 더한 고의적인 변경입니다.
원래 입력·Gooo·레시피·가중치·생성 Go는 유지했고 실제 출력은 변경된 기대값보다
1 작았습니다. 합계 4회 생성·1회 판단·4회 실행·0/256 관측이며, 정상 모델의
정답률에 합치지 않습니다. 저장 기록 읽기 전용 확인 다섯 회도 통과했습니다.
새 의도·학습 갱신은 0개입니다. 아홉 합성 상태와 다섯 실제 요청은 별개입니다.

자체 모델은 4,096 FP32 매개변수, 16,384바이트이며 가중치 SHA256은
`cf00ccc83d17d28ed73fcb869366151a48ffccd3aa8ca8e635aabf19810b9e78`로 같습니다.
macOS arm64/Apple M4/Go1.27.1의 기존 과제 반복입니다. 이번 변화의 부모·모델
단독 RAM과 호스트 CPU 사용률은 미관측입니다.

## 자동 지표에서 읽는 순서

`summary.json`의 `passed`와 `total`은 원래 충족 수와 **선언한 기대값 수**입니다.
`status`, `native_runs`와 해당 runtime의 `observation.cases` 길이,
`runtime_replayed`를 함께 읽습니다. 자식이 시작돼도 출력 해석에 실패하면
관측된 case는 0개일 수 있습니다. 첫 실행에서 출력을 얻고 두 번째 실행이
실패하면 관측 case와 불완전한 replay가 함께 남습니다.

미관측 요청은 모델 점수 0으로 넣는 학습 자료에서 구분합니다. 기존 출력의 실제
불일치와 고의적인 기대값 변경도 원래 범위에 남깁니다. 생성 확률·관측 충족률·
재실행 확인은 각각 다른 사실입니다. [위키 실행법](https://github.com/kimjooyoon/meta-ontology-go/wiki/File-Based-Body-Run).

## 원본과 재실행

- `candidate-native.zip`: 정상·공구 조회 실패·다섯 조건의 stdout/stderr, 프로세스,
  응답·runtime·원본·고정 생성 Go. 모든 파일과 닫힌 목록을 대조했습니다.
- `candidate-smoke-summary.json`, `candidate-controls-summary.json`: 범위별 집계.
- `first-tests.raw`: 실제 실행 전 실패와 형식 거부가 `0/N`으로 보이던 원래 실패.
- `corrected-tests.raw`, `race-tests.raw`, `cli-tests.raw`, `vet.raw`, `modernizer.raw`:
  소스 단위/race/CLI/전체 vet/Go1.27.1 고정점의 실제 검사.
- `smoke/main.go`, `expectation-probe/main.go`, `archive-readback/main.go`: Go 수집기와
  파일 대조기. 합성 표시 테스트는 컴파일러 `internal/bodypathstream/finite_display_test.go`에 있습니다.

```sh
unzip candidate-native.zip -d saved
go run ./archive-readback candidate-native.zip saved
gooo body-path-run --verify-timing --out saved/candidate-controls/observed-zero-model
GOOO_LOCAL_GO=/path/to/go1.27.1 go run ./expectation-probe \
  /path/to/clean-gooo saved/inputs ../order-judge-initial-20261003/model.json \
  saved/expected fresh-controls SOURCE_SHA
```

FIFO 수집은 Unix용입니다. 수집기는 원래 stdout/stderr와 프로세스 결과를 저장한
뒤 조건을 확인하고, 생성한 FIFO를 프로세스 종료 확인 뒤 제거합니다.
