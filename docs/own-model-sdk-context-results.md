# Own-model Gooo context SDK assembly

자체 소형 모델을 Gooo의 제한된 코드 경로 판단에 연결했습니다. 이 단계는
Go SDK에서 타입 계획을 검증한 뒤 실제 Gooo 바디를 조립한 실험입니다.
컴파일러 main의 새 규격 채택이나 범용 자연어 코드 생성 결과는 아닙니다.

## Published component and actual execution

[Go SDK v0.2.9-experimental](https://github.com/kimjooyoon/gooo-decision-runtime/releases/tag/v0.2.9-experimental)
supports explicit split-v2 structural model metadata and a Go-only typed path
example. Source is `2a730c83f1a43f303fe526801890571cc87e9fcb`; source CI, local
unit/race tests and all 29 extracted-file byte checks pass. The previous manifest
is preserved. The example is additional SDK-owned source, separate from the
extracted inference/path kernel.

Protocol and four finite documents were published at research source
`dd378950e1a69c9792d9fb8947ac53a0ca251125` before the local calls. A clean Go
1.27.1 build has binary SHA256
`b27ae79dd8373a4c618e9e8d37a3f227a341e7515e9cbb551a0849adbd3e1ef1`.
Sixteen actual SDK child invocations use the own split-v2 FP32/PTQ/QAT exports
or disconnected control, each in English/Korean and sparse/full contracts.
Twelve model predictions and twelve applied context serializations. No upstream
Laya use and no new optimizer steps. All raw JSON captures are retained.

The example validates the original typed plan before serializing activity/result,
decision kind/address and legal alternative facts into a fixed 512-byte buffer
alongside the original intent. Delimiter ambiguity and overflow fail explicitly.
This is a small caller-supplied typed-plan adapter, not automatic parsing of a
Gooo source file. Its facts differ from current training text. Setup creates a
string; the 1,248-byte prediction workspace and zero-allocation kernel contract
remain separate. Validated bindings and legal paths are compiler-owned.

## Finite completion and negative evidence

All selected results pass 24/24 supplied finite cases. Eight full-contract calls
select `return (input - 2)`, using three additional candidate attempts in total.
Three sparse calls select `return (2 - input)` instead; both return zero at input
two. Those wrong-intent choices are preserved. This deliberately tiny two-path
example reuses one subtraction intention, so it establishes compatibility and
bounded assembly, not effectiveness or generalization.

Full contracts are separate invocations with new initial rankings. Within each
invocation, evaluating a second candidate performs no additional prediction.
An offline audit independently recomputes each forward/reverse candidate's
integer arithmetic, expected values and pass booleans: 30 attempted case
evaluations and 24 selected finite cases. That audit performs no new inference
or subprocesses. Prediction observations across 12 calls range from 11.792 to
20.250 microseconds; one capture and differing contexts do not establish a
speedup. Host utilization and whole-child RSS were not measured here.

SDK child invocations are compiled Go consumers, not execution of emitted Go.
Native compiler calls: zero. Emitted-Go processes: zero. The native main still
uses SDK 0.2.8 and rejects v2 before inference. Original-file source binding,
compiler-produced context and native SDK adoption remain subsequent work.

## Reproduce

From a checkout of the published SDK, supply local public model weights and one
of the four documents under `studies/own-model-sdk-context-v1` in this research
repository:

```sh
go run ./examples/gooo-path --plan en-full.json --model model.json
go run ./examples/gooo-path --plan en-full.json
```

The second command is deterministic disconnected execution. Keep the exact
metadata/weights, document and source hashes with each result. The dedicated
research CI job rebuilds the pinned SDK and repeats all 16 cells, preserving
its own captures as an artifact. Local counts exclude CI and unit-test calls.

The next independent model direction is source-bound Gooo typed context plus
separately supervised Korean/English intent evidence. Finite-test ambiguity must
remain visible. Partial case completion, extra attempts, abstentions, calls and
resource cost remain primary observations; first-shot accuracy is descriptive.
