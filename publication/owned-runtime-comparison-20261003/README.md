# 실행 기록을 다음 비교의 입력으로 이어가기

2026-10-03. 컴파일러 [PR #1188](https://github.com/kimjooyoon/meta-ontology-go/pull/1188),
로컬 측정 소스 [`d830766b`](https://github.com/kimjooyoon/meta-ontology-go/commit/d830766b0b1ade80c561e3f994386e927f035f72).
개발 #1188과 [main #1189](https://github.com/kimjooyoon/meta-ontology-go/pull/1189)의
필수 여섯 검사·원본 증거를 독립 확인해 병합했습니다. 깨끗한 main
`05746e4ae712cfcdcc22aa9fb463d0c88ee90dd5`를 Go 1.27.1로 설치했습니다.
파일 실행 기능의 [main #1187](https://github.com/kimjooyoon/meta-ontology-go/pull/1187)에
이어 저장한 실행 결과를 비교 입력으로 사용하도록 연결한 변경입니다.

## 현재 설치본에서 사용하기

`body-path-run --repeat 2`가 저장한 출력 폴더에서 다음 명령을 실행합니다.

```sh
gooo completeness-delta --before body-results/run-1-generation.json \
  --after body-results/run-1-runtime.json
gooo completeness-delta --before body-results/run-1-runtime.json \
  --after body-results/run-2-runtime.json
```

main 설치본에서 모델 경로 2회 생성·실제 판단 2회·4회 실행과 256/256 기대값,
조건·변수 예제 2회 생성·4회 실행과 6/6 기대값을 확인했습니다. 반복 응답은 각각
31.297·31.455ms였습니다. 앞선 설치본의 기록 18쌍도 읽었고 수정 후보의 비교 출력과
바이트 단위로 일치했습니다. 과거 기록 읽기를 새 모델·실행 횟수로 세지 않습니다.

원래 미충족 쌍은 0/2→0/2, 바꾼 기대값은 128/128과 127/128을 비교할 수 없는
조건으로 남았습니다. 같은 유한 과제에서 oracle 관측을 추가한 경우는 0/2→2/2를
기술적인 충족 수 변화 +2로 기록하고 달라진 선택 관측을 보존했습니다.
처음 수집기는 oracle 옵션이 바뀌면 항상 비교 조건이 달라질 것으로 예상했습니다.
실제 계약은 같은 원본·계획·유한 묶음·선언한 예산을 비교한다는 점을 확인하고
검사 기대를 고쳤습니다. 이 과정은 `collector-scope-assumption.json`에 기록했습니다.
원인별 효과를 구분하려면 별도의 통제된 비교가 필요합니다.

`installed-main-replay.zip`, `installed-summary.json`에 새 설치본의 모든 결과를
보존했고 dev/main의 아티팩트·ZIP 해시·독립 검증은 두 `*-proof-verification.json`에
있습니다. 설치 실행파일 SHA256은
`6f2ccb9ad64409635f1e4d128725dffc56c9e7838903d4437bec2b9d4d847af7`입니다.

## 발견한 연결 문제

`body-path-run`은 생성과 실제 실행을 이어주고, 같은 바디의 실행파일을 보관합니다.
이 소유한 실행 경로는 runtime v2 기록을 만들지만 `completeness-delta`의 소비자는
v1만 읽었습니다. 실제 저장한 생성·실행 파일을 전달하면
`runtime parent receipt binding differs`로 멈췄습니다. 처음의 원본 실패와, 수정 전
실제 빌드로 재현한 `runtime-v2-reader-initial-failure.txt`를 남겼습니다.

## 이번 수정

- 생성자의 v1/v2 프로필 이름을 소비자와 공유합니다. 등록한 두 실행 프로필을 읽고
  미래 프로필과 소유권을 잃은 기록은 거부합니다.
- 원래 부모의 바이트와 해시를 보존하고, v2의 소유한 빌드·관측 복사본·관측 해시가
  맞는지 확인합니다. 성공 빌드를 만들지 못한 실제 실패 관측도 읽습니다.
- 생성→실행은 같은 부모에 이어진 별도 관측입니다. 프로필 간 숫자 개선율을 만들지
  않습니다. 동일 실행 범위에서 현재 기대값의 분자·분모를 비교합니다.
- 첫 빌드의 자원 축은 자식 넷, 재사용은 과거 빌드를 제외한 현재 자식 셋입니다.
  단위·분모가 바뀌는 이 축은 숫자 차이를 내지 않습니다. 기대값이 바뀌면 실행
  묶음의 해시도 바뀌어 서로 비교할 수 없는 범위로 남습니다.

기록 일치 검사는 JSON에 들어 있는 내용의 연결을 확인합니다. 실제 실행에 대한
근거는 원본 확인, 도구와 자식 프로세스를 관측한 생성자의 기록에서 읽습니다.
비교 명령의 모델 호출·생성 프로그램 실행·외부 요청·저장소 쓰기는 모두 0회입니다.

## 실제 과거 기록과 자체 모델로 확인하기

앞서 공개한 `c021fc4f` 한영 모델·결정론 기록을 새 CLI에 전달했습니다.
**8개 부모 연결 + 4개 최초 실행/재사용 쌍, 총 12개 비교**를 읽었습니다.
과거 프로그램을 다시 실행한 횟수로 세지 않습니다.

그 뒤 깨끗한 수정 소스의 컴파일러로 자체 공개 모델을 호출해 한국어 활동
`AssembleKorean`의 바디를 새로 생성했습니다. 기존 `2*input+1` 과제의 128개 입력과
기대값을 사용하고 두 요청을 보냈습니다.

| 현재 소스의 새 관측 | 결과 |
| --- | ---: |
| 생성 | 2회 |
| 실제 모델 판단 | 2회 |
| 즉시 컴파일한 프로그램 실행 | 4회 |
| 유한 기대값 | 256/256 |
| 생성된 Go와 이전 원본 비교 | 두 파일 모두 일치 |
| 추가 학습 | 0회 |
| 첫 / 반복 응답 | 472.521 / 26.760ms |

응답은 생성·실행을 포함하고 최초 모델 준비와 저장·출력 비용을 제외합니다.
호스트 캐시를 통제한 성능 비교는 이 수집에서 수행하지 않았습니다. 한 장비의
알려진 산술 과제이며, 128개 입력의 반복을 새로운 128개 실험으로 세지 않습니다.
가중치 16,384바이트와 SHA256
`cf00ccc83d17d28ed73fcb869366151a48ffccd3aa8ca8e635aabf19810b9e78`는 그대로입니다.

새 생성→실행 비교는 `PARENT_RUNTIME_CONTINUATION`, 실행→재사용은
`SAME_MEASUREMENT_SCOPE`였습니다. 유한 충족 축은 128→128, 차이 0으로 기록하고,
자원 축은 4→3의 분모와 바뀐 단위를 보존하며 차이를 `null`로 남겼습니다.

## 검사와 원본

실제 최초 빌드·재사용, 부모의 공백 변경, 부분 도구 실패, 실패하는 자식 빌드,
누락·미래 스키마·관측 해시·빌드 해시·일치시킨 두 복사본의 잘못된 종료값,
bare receipt의 소유 범위 누락과 기대값 변경을 확인했습니다.
bodyexecution/completenessdelta/bodypathstream/cmdgooo의 race와 전체 vet가 통과했습니다.
첫 비교 회귀 검사에서는 테스트가 실제 축 이름을 잘못 지정해 실패했습니다.
`runtime_child_resources`의 기존 계약을 확인해 4/3 분모와 서로 다른 단위를 검사하도록
고쳤고 그 실패도 `runtime-v2-comparison-first-fix.txt`로 보존했습니다.

`cli-replay.zip`에 과거 12개 비교, 새 생성·실행·비교, 원본 입력과 생성된 Go를 담았습니다.
`runtime-v2-cli-summary.json`은 새 호출 수를 별도로 요약합니다.
[사용법과 지원 계약](https://github.com/kimjooyoon/meta-ontology-go/blob/d830766b0b1ade80c561e3f994386e927f035f72/docs/completeness-delta.md)
· [파일 실행 원본](../body-path-file-cli-20261003)
· [자체 모델](https://huggingface.co/asketeddy/gooo-order-judge-tiny-v1).
