# 소스·레시피·기대값 파일로 바로 실행하기

컴파일러의 `gooo body-path-run`을 사용하는 후속 예제입니다.
명령이 직접 생성·실행·기록을 담당합니다. 준비 프로그램은 128개 기대값과
활동 이름을 바꾼 한영 입력 파일만 구성합니다.

[사용법, 정확한 컴파일러 소스, 정상·미충족 원본과 비용](../../publication/body-path-file-cli-20261003).

연구 저장소 루트에서 Go 1.27.1과 해당 소스의 `gooo`로 실행합니다.

```sh
go run ./examples/file-body-run examples/whole-candidate-order file-body-inputs

gooo body-path-run --source file-body-inputs/ko-source.gooo --activity AssembleKorean \
  --path-plan file-body-inputs/ko-recipe.json --cases file-body-inputs/cases-128.json \
  --model publication/order-judge-initial-20261003/model.json \
  --repeat 2 --out file-body-model-results
```

`--model`을 생략하면 결정론적으로 조립합니다. 영어 입력은 `en-source.gooo`,
`en-recipe.json`, 활동 이름 `AssembleEnglish`입니다. 준비·결과 폴더는 각각 새 경로를
사용합니다. 준비 프로그램은 표준 Go int64 연산으로 `2*input+1` 기대값을 구성합니다.
입력 128개는 -62..61과 int64 양 끝의 네 값입니다. 고정 배열에 두 int64 필드로
배치하고 JSON 파일로 저장합니다. 프로그램 실행과 모델 호출은 `gooo`가 담당합니다.

현재 예시와 별도 Gooo oracle 관측의 측정 범위·실패는 링크한 원본 보고서에서 읽습니다.
