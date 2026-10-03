# 저장된 후보와 실패 피드백을 Go로 다시 실행하기

2026-10-03. [공개 소스·PR #19](https://github.com/kimjooyoon/gooo-metaprogramming-experiments/pull/19).
실제 로컬 측정과 아래 Linux 기록의 소스는
`be3e73e6d52ff0418860459998dc87bfeeb8fb2c`입니다.
이후 문서만 고친 `843a0c664bf2c43a78b422844b588f4c89cfe16d`에서도
일곱 검사군·13개 공개 체크가 통과해 PR #19를 병합했습니다. 현재 main은
`aad6520ed37c5de0c153409febf548275c83a75b`이며, 도구 11개 파일은 측정 소스와
동일합니다. 병합된 전체 트리도 PR 최종 트리와 일치함을 확인했습니다.

이전 후보 재현은 준비 스크립트의 해시 변경에서 멈췄습니다. 같은 경로에 옛 물리
Go1.27.0 요구도 남아 있었습니다. 현재 준비 파일은 Go1.27.1을 유지하고, 당시
파일은 해시가 결속된 자료로 보존했습니다. Go 재현 도구가 원래 142개 입력,
revision-2의 536개 입력과 797개 결과·소스 결속을 확인한 뒤 저장된 Go를 실행합니다.
원래 [실패한 CI](https://github.com/kimjooyoon/gooo-metaprogramming-experiments/actions/runs/37120308264)와
[실패 원문](../go-baseline-replay-20261003/revision2-ci-original-failure.txt)은 보존했습니다.

## 사용법

```sh
cd tools/baseline-replay
go run . --revision2 --root ../.. --go-bin "$(command -v go)" --output /tmp/gooo-candidates-new
```

Go1.27.1과 새 출력 폴더를 사용합니다. `--revision2`를 생략하면 앞선 두 컴파일러의
저장 프로그램 52개를 확인합니다. [두 모드의 안내](https://github.com/kimjooyoon/gooo-metaprogramming-experiments/tree/be3e73e6d52ff0418860459998dc87bfeeb8fb2c/tools/baseline-replay).

## 실제 재실행 결과

| 관측 | 원래 후보 | revision-2 후보 |
| --- | ---: | ---: |
| 의도 | 32 | 같은 32 |
| 선언된 후보 | 96 | 96 |
| 컴파일·실행한 후보 | 93 | 96 |
| 컴파일 실패 | 3 | 0 |
| 상태 미확인 후보 | 0 | 0 |
| 실제 정수 입력행 | 660/684 | 696/696 |

원래 세 후보의 컴파일 실패가 그대로 재현됐습니다. 그 후보들에 계획된 24개
입력행에는 실행 결과가 없습니다. 나머지 후보들의 기능 충족 수는 후보별로 기록했고,
틀린 대안의 실패도 그대로 남아 있습니다. 컴파일·실행 성공만으로 기능 충족을
집계하지 않습니다.

revision-2에서 지정된 정답 후보는 선택 입력 137/137과 평가 입력 95/95를 충족합니다.
32개 선택 입력 묶음 모두가 정답 후보와 두 대안의 차이를 드러냅니다. 이 정답 후보는
실험 설계에서 지정한 것이며, 이번 재현의 모델 판단·새 코드 생성·새 의도는 모두
0입니다. 앞선 버전과 같은 과제를 다시 확인했습니다. Gooo 소스 AST 완전성은
별도 컴파일러 기록에서 확인하는 지표입니다.

## 결과가 새로 실행됐음을 확인하는 방법

- 두 기준 함수 모듈과 두 후보 모듈을 임시 복사본에서 실행했습니다.
- 복사된 후보 결과 파일 189개를 실행 전에 지웠습니다. 각 유효 후보에서
  `TestCompiledCandidateFiniteCases`가 실제 통과해야 다음 단계로 갑니다.
- 새로 작성된 189개 파일을 고정된 파일 해시와 대조했습니다. 모두 일치합니다.
- 원래 Python int64 기준식을 Go로 옮겨 원래 228개와 수정 232개 기대값을
  확인했습니다. 저장된 독립 Go 기준 함수도 실제 실행했습니다.
- Laya 계획 96개를 자료로 읽고, 외부 피드백 32묶음을 원본 소스·선택 입력
  해시·후보 ID·새로 실행한 선택 입력 결과와 대조했습니다. 평가 입력은
  피드백 관측에 포함하지 않습니다. Python 실행과 모델 호출은 0회입니다.
- 모듈마다 시간 제한을 두고, 취소 시 자식 프로세스 그룹과 출력 파이프를 정리합니다.
  stdout/stderr는 각각 16MiB로 제한하고 종료·시간 초과 관측을 보존합니다.

## 소스와 공개 CI 결속

로컬 race/vet와 실제 재현이 통과했습니다.
[후보 Linux CI](https://github.com/kimjooyoon/gooo-metaprogramming-experiments/actions/runs/37122086981)와
[기본 재현 Linux CI](https://github.com/kimjooyoon/gooo-metaprogramming-experiments/actions/runs/37122087007)도
같은 소스에서 통과했습니다. 두 CI는 PR #19의 첫 실행 시도이며, 실행·PR·artifact
정보와 ZIP 해시를 독립적으로 확인했습니다. 모든 11개 도구 파일의 해시가 해당
Git 소스와 일치합니다. 소스 집합 해시는
`a391cb510ae6e29cec97f63a161fb19d28a47c247862eebbd33e8cf96731d41a`입니다.

- `local-exact-native.zip`, `local-report.json`: 로컬 프로세스 원문·시간·결과.
- `linux-candidate-exact.zip`, `linux-baseline-exact.zip`: 공개 CI artifact 원본.
  각각 `8ef22862…2969d9`, `d0f71cf3…194fd1`이며 전체 해시는 검증 파일에 있습니다.
- `ci-artifact-verification.json`: 실행·PR·첫 시도·해시·도구 소스 결속 확인.
- `local-source-binding.json`: 로컬 실행파일 해시와 정답 후보 유한 충족 수.
- `local-buildinfo.txt`: 실행파일의 빌드 정보. 첫 줄 로컬 경로만 가렸습니다.
  작업 폴더에 이전 미추적 캡처가 있어 `vcs.modified=true`이며, 실제 도구 소스
  11개는 모두 소스 커밋과 별도로 대조했습니다. 깨끗한 전체 체크아웃으로 보고하지 않습니다.

로컬 명령의 관측은 71.78초, user+system 75.44초, 최대 RSS 101,253,120바이트입니다.
이 수치는 재현 명령과 자식 실행의 관측 범위입니다. 모델 응답 지연이나 전체 호스트
CPU 이용률로 환산하지 않습니다. 빌드 캐시와 시스템 작업을 통제한 속도 비교도
수행하지 않았습니다.

이 작업으로 개발자가 기존 후보와 피드백을 현재 환경에서 다시 확인하는 경로를
복구했습니다. 실패 자료, 새 실행 자료, 의도 수와 충족 수를 함께 읽을 수 있습니다.
