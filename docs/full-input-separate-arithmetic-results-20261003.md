# Gooo judgment arithmetic: paired arm64/Linux results

## 한국어 요약

작은 모델이 조립 후보의 순서를 고를 때, 계산 환경의 미세한 차이가 다음에
시도할 경로를 바꾸는 문제가 있었습니다. 같은 눈금의 자로 부품을 재도록 하듯,
곱셈과 덧셈에서 반올림하는 시점을 모델의 실행 규칙에 명시했습니다.

18,432개 입력을 두 환경에서 대조한 결과, 기존 규칙에서는 후보 순서가
272건 달랐고 새 규칙에서는 모두 일치했습니다. 중간 계산값과 확률도 정확히
일치했습니다. 첫 선택의 완성도와 완성까지 필요한 추가 시도 수는 유지됐습니다.
이번 작업은 모델의 실행을 재현하기 위한 진전이며, 검증 범위는 이미 관측한
개발 입력과 Go 1.27.1의 두 환경입니다. SDK·컴파일러 적용은 다음 단계입니다.

## Controlled comparison

The [registered protocol](https://github.com/kimjooyoon/gooo-neural-decision-experiments/blob/250c935642b59f4f3122360c90286e5557af3da0/docs/full-input-separate-arithmetic-protocol-20261003.md)
was public before collection. Both platforms used source
`250c935642b59f4f3122360c90286e5557af3da0`, Go 1.27.1 and identical computational
source pins. Each invoked four lanes on every frozen input: legacy expanded,
legacy compact, explicit expanded and explicit compact.

- 36 conditions: four training arms × three numerical variants × three forms.
- 512 previously observed Korean/English development views per condition.
- 18,432 paired rows; **73,728 actual predictions per platform, 147,456 total**.
- **Zero optimizer updates**. All model weight bytes, input text, features,
  temperatures, scales and finite expectations are fixed.
- The comparison reader makes zero model predictions and independently recomputes
  36,864 complete feature arrays from the two retained collections.

| Cross-platform difference | Legacy arithmetic | `float32_separate_v1` |
| --- | ---: | ---: |
| Hidden activation rows | 18,221 | 0 |
| Logit rows | 16,236 | 0 |
| Probability rows | 15,524 | 0 |
| First selected masks | 0 | 0 |
| Complete candidate rankings | 272 | 0 |
| Partial-coverage / completion-position rows | 38 | 0 |
| Maximum absolute logit delta | 1.430511474609375e-6 | 0 |
| Maximum absolute probability delta | 4.470348358154297e-7 | 0 |

Expanded and compact lanes agree exactly within each arithmetic version on both
platforms. The explicit lanes also agree on every feature, hidden, logit and
probability bit digest. All complete rankings and finite outcomes agree. The
preregistered softmax difference bound was 1e-5; the observed difference is zero.

The local legacy run exactly reproduces the originally retained logits,
probabilities and rankings. On Linux, explicit and legacy observations agree.
On arm64, switching to explicit arithmetic changes 272 full rankings and 38
partial-coverage rows. First-path completion and additional attempts to a complete
path stay the same in all 36 conditions. Partial coverage moves in both directions:
for example, positioned-original FP32 original-input budget-two coverage changes
3,918→3,917 passed cases out of 8,192; its task-prefix counterpart changes
4,158→4,162. Both directions remain in the journals and summary tables.

## What changed in the runtime

`arithmetic_version: float32_separate_v1` makes product and accumulation rounding
explicit: `float32(sum + float32(x * weight))`. Ternary scaling is rounded before
bias addition. Iteration order, ReLU and softmax retain their original definitions.
The [Go floating-point specification](https://go.dev/ref/spec#Floating_point_operators)
allows fused operations and defines explicit conversions as rounding barriers.
The controlled change removes the observed platform difference in this cohort;
synthetic cancellation tests separately exercise the fused/separate distinction.

Expanded V3/V4 and compact three-choice loaders accept the declared version.
Unknown versions are rejected. Compaction and initial/feedback receipts preserve
the arithmetic identity. Omitted metadata continues to select the original
implementation, keeping old artifacts and observations reproducible.

The request workspace remains 3,200 bytes. Synthetic regression tests retain
zero allocations in the prediction kernel, atomic rejection and concurrent
immutable model use. Collection wall times were 1.829 seconds on darwin/arm64 and
3.657 seconds on linux/amd64. These single whole-collection observations include
file verification, all four lanes, JSON and gzip; standalone inference latency,
CPU utilization and peak memory were not measured in this follow-up.

## Evidence and use

- [Complete GitHub bundle](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/full-input-separate-arithmetic-20261003).
- [Hugging Face appendix](https://huggingface.co/asketeddy/gooo-shared-judgment-tiny-v1/tree/main/research/full-input-separate-20261003).
- [Successful Linux collection](https://github.com/kimjooyoon/gooo-neural-decision-experiments/actions/runs/37065925595/job/111033575750).
- [Earlier platform diagnosis](https://github.com/kimjooyoon/gooo-neural-decision-experiments/blob/main/docs/full-input-numerical-portability-followup-20261003.md).

The bundle contains all 36 four-lane journals from each platform, both collection
reports, the comparison, 24 separately named model metadata/weight pairs and
the computation sources. Every public file and decoded archive member has a
size and SHA-256 inventory. Gzip journals retain the complete inputs and all
intermediate arrays. The packer scans decoded text for private paths and tokens.

Use the matching research runtime and
`models/compact/<arm>/<variant>/model.json` from this appendix to select the new
contract. Expanded artifacts live under `models/expanded/`.
[SDK v0.2.15](https://github.com/kimjooyoon/gooo-decision-runtime/releases/tag/v0.2.15-experimental)
now supports these contracts; its [complete replay reports](../publication/full-input-sdk-replay-20261003/README.md)
record exact agreement on both platforms. The native compiler currently uses
SDK v0.2.14 and its earlier V3 contract. Its next integration stage carries
feature/arithmetic identity through actual generated-Go execution.

The original legacy CI comparator remains exact and continues to report its
known platform failure. The new arithmetic has its own paired acceptance check.
Further input forms and splits, representation collisions, unseen tasks,
resource measurements, and language discovery/completeness work remain open.
