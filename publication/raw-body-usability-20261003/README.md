# 여러 줄로 함수 바디를 작성하기

2026-10-03. [컴파일러 PR #1190](https://github.com/kimjooyoon/meta-ontology-go/pull/1190),
측정 소스 [`ad87b510`](https://github.com/kimjooyoon/meta-ontology-go/commit/ad87b5104c464312ec05a6c9d08a84d3caabf95b).
개발 #1190과 [main #1191](https://github.com/kimjooyoon/meta-ontology-go/pull/1191)의
필수 여섯 검사·불변 증거를 독립 확인해 병합했습니다. 깨끗한 main
`eb8477d51f58131e0dc3b7126c547bde2c5d0e5f`를 Go1.27.1로 빌드해 설치했습니다.

## 줄바꿈과 따옴표를 직접 쓰기

Gooo 바디를 긴 `\n` 문자열로 작성하는 부담을 줄이는 작은 문법입니다.
새 소스에서 빌드한 컴파일러로 다음과 같이 작성할 수 있습니다.

```gooo
activity ClampBelowZero(Integer) -> Integer computes `
let value = input
value = input
let accepted = value >= 0
if accepted {
    return value
} else {
    return 0
}
`
```

[전체 파일과 명령](https://github.com/kimjooyoon/meta-ontology-go/blob/ad87b5104c464312ec05a6c9d08a84d3caabf95b/docs/language/body-codegen.md).
백틱은 문자열을 닫고 안의 역슬래시·줄바꿈을 그대로 보존합니다. 내부 Text 값은
바디의 Go 문자열 규칙으로 읽습니다. 기존 이중 따옴표는 종전의 이스케이프 규칙을
사용합니다. 정규 Gooo 포맷은 해석된 내용을 유지한 quoted 문자열로 출력합니다.

## 실제 자체 모델로 확인한 범위

기존 한영 `AssembleKorean`·`AssembleEnglish` 파일의 마지막 `computes` 값을 같은
해석값의 raw 문자열로 바꿨습니다. 조립 계획·128개 정수 기대값·16,384바이트 가중치는
그대로입니다. 모델·결정론 네 조건을 각각 두 번 조립하고 즉시 실행했습니다.

| 관측 | 결과 |
| --- | ---: |
| 생성 | 8회 |
| 실제 모델 판단 | 4회 |
| 컴파일한 프로그램 실행 | 16회 |
| 유한 기대값 | 1,024/1,024 |
| 앞선 quoted 소스 설치본과 생성 Go 비교 | 8/8 일치 |
| 가중치 갱신 | 0회 |

모델의 최초 응답은 한영 각각 487.173·477.383ms, 반복 응답은 32.719·31.766ms였습니다.
결정론 반복은 27.684·30.805ms였습니다. 생성·실행을 포함하며 최초 모델 준비와
파일 저장·출력을 제외합니다. 알려진 정수 연산 가족과 한 장비의 관측입니다.
128개 입력의 반복을 새로운 128가지 실험으로 집계하지 않습니다.

## 원래 실패와 검사

- 처음 TDD는 백틱을 인식하지 못해 실패했습니다.
- 첫 수정 검사는 공백 구성이 다른 두 바디의 생성 코드를 비교했습니다. 원본의
  빈 줄을 그대로 보존하는 차이를 확인했습니다. 같은 해석값을 canonical quoted로
  바꿔 생성·의미 항목을 대조하도록 수정했습니다. 다른 공백 구성은 생성 형식과
  소스 해시를 바꿀 수 있습니다.
- 파싱·포맷 검사에서 필수 package/namespace를 빠뜨린 테스트도 보존했습니다.
- 초기 요약은 wire 응답에서 파일 드라이버의 `response_ms`를 찾아 null을 냈습니다.
  실제 저장한 `summary.json` 행에서 시간을 읽도록 고쳤고 초기 사본도 남겼습니다.
- 전체 syntax/bodycodegen race와 전체 vet가 통과했습니다. CR·LF·CRLF, 한글,
  문자 그대로의 역슬래시·내부 따옴표, 잘못된 UTF-8 한 바이트의 위치, 닫히지 않은
  문자열과 포맷 고정점을 검사했습니다.

`raw-computes-public.zip`은 원본 입력·생성 Go·생성/실행 기록·요약을 담습니다.
각 검사와 초기 실패, 초기 요약·수정된 요약을 함께 공개합니다.

## 설치한 main에서 다시 사용하기

설치한 `gooo`로 다음 명령을 실행할 수 있습니다.

```sh
gooo body-codegen --activity ClampBelowZero \
  examples/body-codegen/raw-computes.gooo.fixture
```

원본 파일에는 package·namespace·Integer 선언까지 들어 있습니다. 지역 변수,
대입과 조건을 줄바꿈으로 작성하고 Go 함수를 생성합니다. 설치본에서 이 바디를
별도로 컴파일·실행해 int64 양 끝값을 포함한 8/8 기대값을 확인했습니다.
잘못된 UTF-8을 넣은 raw 바디는 6:1–6:2에서 생성 전에 거부했습니다.
기존 quoted 바디·타입 경로 예제와 기본 명령 다섯 개도 확인했습니다.

같은 모델·결정론 네 조건을 main에서 다시 조립해 새로 8회 생성·실제 판단 4회·
실제 실행 16회·1,024/1,024 기대값과 생성 Go 여덟 파일의 일치를 확인했습니다.
반복 응답은 모델 한영 82.580·81.439ms, 결정론 93.576·76.820ms였습니다.
최초 모델 응답 957.629·826.189ms도 남겼습니다.

이번 반복 응답은 위 개발 소스 관측의 27.684–32.719ms보다 느렸습니다.
반복의 현재 자식 CPU 합은 27.163–34.796ms, 최대 단일 자식 RSS는
14,172,160–14,385,152바이트입니다. 모델 자체·부모 컴파일러·호스트 전체 자원은
별도로 관측해야 합니다. 입력/출력과 과제는 일치했으며, 캐시·스케줄링·다른 작업이
통제되지 않은 한 장비의 시간 관측을 그대로 보존합니다.
한국어 모델 반복의 바디 생성은 3.591ms였고 `go version` 자식만 21.568ms였습니다.
후보 판단 이후의 검증·실행 비용을 따로 살펴볼 근거입니다.

저장된 생성→실행 기록은 `PARENT_RUNTIME_CONTINUATION`, 실행 두 기록은
`SAME_MEASUREMENT_SCOPE`로 읽었습니다. 현재 자식 자원의 분모 4→3은 단위도
바뀌므로 숫자 개선을 계산하지 않습니다. 추가 학습은 0회입니다.

`installed-main-native.zip`은 새 설치본의 원본·생성 Go·결과·비교·clamp 테스트를
담습니다. 기본 구조 생성의 로컬 출력 위치만 공개 stdout에서 가렸고 경로를 담은
manifest는 제외했습니다. 원본 파일은 로컬에 보존했습니다.
`installation.json`, `installed-summary.json`, `installed-costs.json`과 개발/main
증거 검증 결과도 공개합니다.
