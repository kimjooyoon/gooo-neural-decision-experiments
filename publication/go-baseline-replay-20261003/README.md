# 저장된 바디 실험을 Go로 다시 확인하기

2026-10-03. [공개 소스와 PR #19](https://github.com/kimjooyoon/gooo-metaprogramming-experiments/pull/19),
측정 소스 `acfeaab2937cfaaba384129895f06a655238b3f4`.
로컬 race/vet와 실제 재실행 및 해당 소스의
[Linux 기본 재현 CI](https://github.com/kimjooyoon/gooo-metaprogramming-experiments/actions/runs/37120308246)가
통과했습니다. 별도의
[revision-2 후보 재현](https://github.com/kimjooyoon/gooo-metaprogramming-experiments/actions/runs/37120308264)은
옛 고정 파일 해시에서 실패했습니다. 이후 Go 도구에 물리 Go1.27.1 경로와 추가
준비 스크립트의 버전 결속을 연결했습니다. 새 로컬·Linux 후보 재현이 통과해
PR #19는 병합됐습니다. [새 실행 원본·후보별 충족 수](../go-revision2-replay-20261003).
아래 표와 여덟 파일의 소스 결속은 최초 `acfeaab` 측정을 그대로 설명합니다.

## 고친 사용상의 문제

옛 바디 실험은 142개 입력 파일의 해시를 고정했습니다. 이후 두 준비 스크립트의
Go 버전을 1.27.1로 바꾸면서, 고정 입력을 확인하는 단계에서 재현이 멈췄습니다.
실패한 [36753863347](https://github.com/kimjooyoon/gooo-metaprogramming-experiments/actions/runs/36753863347),
[37070962850](https://github.com/kimjooyoon/gooo-metaprogramming-experiments/actions/runs/37070962850)은
그대로 남아 있습니다.

현재 파일과 당시 파일을 각각 보존했습니다. 원래 freeze와 결과 파일은 그대로이며,
Go 재현 도구가 140개 현재 파일과 두 개의 명시적으로 결속된 옛 사본을 확인합니다.
옛 준비 스크립트는 자료로 읽고, 저장된 Go 코드만 컴파일·실행합니다.

```sh
cd tools/baseline-replay
go run . --root ../.. --go-bin "$(command -v go)" --output /tmp/gooo-new-replay
```

Go1.27.1과 새 출력 폴더를 사용합니다. [사용법·검사 범위](https://github.com/kimjooyoon/gooo-metaprogramming-experiments/tree/acfeaab2937cfaaba384129895f06a655238b3f4/tools/baseline-replay).

## 정확한 소스에서 재실행한 결과

| 관측 | 원래 컴파일러 기록 | 조건 동치 수정 기록 |
| --- | ---: | ---: |
| 원래 과제 | 32 | 같은 32 |
| 저장된 생성 Go를 실제 재실행 | 22/32 | 30/32 |
| 생성물이 없어 미확인인 과제 | 10 | 2 |
| 선택 입력 | 89/135 충족, 46 미확인 | 125/135 충족, 10 미확인 |
| 평가 입력 | 65/93 충족, 28 미확인 | 87/93 충족, 6 미확인 |

관측 가능한 저장 프로그램 52/52개를 컴파일·실행했습니다. 별도 후보 96개의
컴파일 검사에서는 93개가 통과하고 원래 후보 세 개가 똑같이 실패했습니다.
그 검사에서는 테스트 함수를 실행하지 않았습니다. 모델 호출·새 코드 생성·학습
갱신·새 과제는 모두 0입니다.

각 소스·기대값·원본 stdout/stderr·생성 Go·러너 사본과 해시를 대조했습니다.
잘못 집계된 원래의 측정 32개/미측정 0개와, 개별 기록에서 재구성한 30개/2개를
함께 보존합니다. 수정본은 허용된 세 집계 필드만 바뀌었음을 확인합니다.
int64 경계값을 정수로 그대로 읽고, 누락된 지표를 성공한 0으로 읽지 않습니다.

## 소스 결속과 원본

- `exact-native.zip`: 현재 구현의 모든 프로세스 로그·실행 관측·최종 보고서.
- `exact-report.json`, `source-resolution.json`: 쉽게 읽을 원본 사본.
- 현재 구현 여덟 파일의 해시가 모두 소스 커밋과 일치합니다. 소스 집합 해시는
  `9e1b2b0fb893fc8dfd0a4905e421e32eb0fe8c350ed1be7e7a909c4dc9805e08`입니다.
- 실행파일 해시는 `286b95a14998d2e86cfda4266060974089a33521438452116f4a9286866fdc3c`이며,
  실행 앞뒤로 도구 소스·실행파일·Go 실행파일이 같은지 다시 확인했습니다.
- `initial-pilot.zip`: 먼저 실행한 52개 모듈과 후보 96개 검사. 이때는 도구 소스
  해시 기록을 추가하기 전이어서 러너의 정확한 소스 결속을 UNKNOWN으로 남겼습니다.
- 초기 수집과 현재 구현 수집은 같은 과제를 반복 확인했습니다. 합계 104개 모듈
  실행을 새로운 104개 실험으로 집계하지 않습니다.

초기 수집은 Go 명령의 빌드까지 포함해 79.29초, 현재 실행파일 수집은 31.39초였습니다.
진입 방법과 캐시 상태가 달라 각각의 관측으로 읽습니다. 현재 수집의 user+system은
34.12초, 최대 RSS는 96,452,608바이트입니다. 생성기·모델의 자원 지표와 별도로
현재 재현 프로세스와 자식들의 관측 범위를 기록했습니다.

저장된 결과를 확인할 수 있는 경로를 복구한 작업입니다.
