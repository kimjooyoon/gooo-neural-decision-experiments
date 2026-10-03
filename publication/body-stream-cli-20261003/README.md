# Integrated Gooo stream command: local evidence

Compiler revision `f6697dda897481f111113f24ed899604f8e3260a`, clean Go1.27.1
Darwin arm64 build; [PR #1180](https://github.com/kimjooyoon/meta-ontology-go/pull/1180).
This publication records local verification before CI and promotion finish.

`gooo body-path-stream` and the existing standalone worker share the same bounded
runner and command implementation. Users can keep the compiler process open and
send one source recipe or typed plan per line. The optional local model loads
once. Omit `--model` to construct deterministically. The compiler documentation
includes a runnable recipe pipeline and explains individual rejections, output
order, exit codes and cancellation.

The public [two-request example](../../examples/whole-candidate-order) was run
through the new command with the original 16 KiB model and then without a model.
All four generated results were built and executed twice, matching **32/32 finite
expectations across eight native executions**. The second model request reused
candidate preparation. Weights and training data did not change. These are known
examples and entry-point checks, not a new accuracy or speed experiment.

The documentation's deterministic recipe pipeline also completed both requests.
Focused race tests cover both CLI dispatch and the stream, including responses
before input EOF, recovery after a rejected record, cancellation during an input
wait, usage and model setup errors. Full `go vet ./...` and the billing semantic
check passed. Full canonical CI is reported by the linked PR; these local checks
are not its replacement.

Each `*-runtime.json` contains independent compiled execution and source identity.
The JSONL files preserve original request responses; setup JSON, usage output,
build metadata and test logs are included. Verify with `shasum -a 256 -c SHA256SUMS`.

## 한국어 사용법

이 기능이 포함된 컴파일러로 연구 저장소 루트에서 실행합니다.

```sh
gooo body-path-stream --workers 1 \
  --model publication/order-judge-initial-20261003/model.json \
  < examples/whole-candidate-order/worker-requests.jsonl > /tmp/gooo-stream.jsonl
jq '{id:.correlation_id,status,preparation:.response.report.body_paths.whole_candidate_preparation}' /tmp/gooo-stream.jsonl
jq -s '.[1].response' /tmp/gooo-stream.jsonl > /tmp/gooo-order-generation.json
```

생성 파일은 예제의 `body-execute` 명령에 바로 전달할 수 있습니다. 모델 경로 옵션을
생략하면 결정론적으로 처리합니다. 결과의 `status`를 요청마다 확인해야 합니다.
여러 워커를 쓰면 완료 순서가 달라질 수 있어 `sequence`와 `correlation_id`로 연결합니다.
EOF 이전에도 응답하므로 다음 프로그램이 완성된 바디부터 실행할 수 있습니다.

The build-info first line replaces the local binary path with its basename;
version, dependencies, source revision and clean-tree flag are preserved.
