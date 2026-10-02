---
license: mit
language:
- en
- ko
tags:
- gooo
- metaprogramming
- golang
- tiny-model
- ternary
---

# A small local judge for Gooo program construction

Updated 2026-10-03. This model ranks **eight permitted Gooo body paths** built
from three binary decisions. The project explores a language and a small model
working together: Gooo provides the construction plan, the model suggests a
route through it, and the compiler checks and executes the assembled program.

Like a workshop, the plan describes what the parts mean and how they may fit.
The model helps choose the next assembly; observed test failures guide another
attempt. A receipt connects the original intention to the choices and result.

| Component | Public home |
| --- | --- |
| Gooo language, compiler and execution | [meta-ontology-go](https://github.com/kimjooyoon/meta-ontology-go) |
| Go inference and typed search | [gooo-decision-runtime](https://github.com/kimjooyoon/gooo-decision-runtime) |
| Training, raw evidence and comparisons | [gooo-neural-decision-experiments](https://github.com/kimjooyoon/gooo-neural-decision-experiments) |
| Project direction in Korean | [언어·모델·실험 안내](https://github.com/kimjooyoon/meta-ontology-go/blob/dev/docs/language-direction.ko.md) |

## 한국어 요약

Gooo 선언을 설계도, 작은 모델을 조립 순서를 고르는 장치로 생각하면 됩니다.
현재 모델은 세 가지 이진 판단으로 이루어진 여덟 경로에 순위를 매깁니다.
컴파일러는 조건식·할당·분기 등을 조립하고, 실패한 입출력 예시를 다음 시도의
문맥으로 전달합니다. 생성된 Go의 빌드·실행 결과와 남은 의문도 기록합니다.

개발은 Laya의 구조화된 선택 실험에서 출발해, Gooo 자료로 새로 학습한
2,072개 파라미터 모델과 Go 실행기로 이어졌습니다. 현재 공개물에는 FP32와
삼진 가중치, 학습 기록, 실제 코드 생성·실행 관측이 함께 들어 있습니다.
목표는 적은 메모리로 의도와 생성 경로를 연결하고, 완전성을 여러 항목으로
관찰하며 다음 작업을 이어갈 수 있게 하는 것입니다.

최근에는 문장 앞의 도입 표현만 달라도 선택이 크게 달라지는 문제를 확인했습니다.
기존 입력의 첫 경로 충족률은 113/512이고, 지정한 표현을 제거한 실험 조건에서는
480/512였습니다. 아래 표들은 이 조건들을 구분합니다. 다음 연구에서는 전체
입력을 유지하면서 표현 변화와 한·영 의도에 대한 일관성을 개선하려 합니다.

## How the model is used

The compiler derives context from an original Gooo body, its declared typed
alternatives, and complete Korean/English intentions. The model ranks complete
paths involving conditions, assignments, references, nested branches and
arithmetic. Finite expectations evaluate a proposed body. Later predictions may
include actual failed-case feedback and rank the remaining choices.

The model makes small decisions inside a supplied plan. The compiler supplies
the source binding, allowed alternatives, attempt budget and deterministic
continuation. Disconnected or unsupported inputs keep their complete text and
continue with zero model predictions. Generated Go then has a separate replay,
build and execution stage.

The current artifacts target authored Integer → Integer tasks. Broader source
discovery, larger programs, reusable abstractions and Korean/English meaning
alignment are active development areas.

## Architecture, training and files

The shared model reuses a **256 → 8 → 2** local network over three ordered
decisions: **2,072 trainable parameters**. The dense control has 18,656.
Both were freshly initialized using the same Go initializer recipe and trained
on Gooo-derived data. Each arm completed 800 FP32 and 800 QAT updates on local
MPS, totaling **3,200 optimizer updates**. PTQ exports come from the FP32 models.

The data contains 1,024 training, 256 calibration and 256 development program
groups. The development set has 512 Korean/English views and had been observed
in earlier studies. The source/teacher feature bank has 10,739 initial and
feedback rows. Quality numbers below describe this known development cohort.

| Files in this Hub repository | Purpose | Go loader |
| --- | --- | --- |
| `research/compact-runtime-20261003/models/{fp32,ptq_ternary,qat_ternary}/` | Current compact shared representation | `jointdecision.LoadSharedThree` |
| `models/shared-local/{fp32,ptq_ternary,qat_ternary}/` | Original expanded shared exports | `jointdecision.LoadThree` |
| `models/dense/{fp32,ptq_ternary,qat_ternary}/` | Matched dense controls | `jointdecision.LoadThree` |
| `evidence.zip`, `go-audit.json`, `manifest.json` | Original training and native study evidence | Go audit tools in the research repository |
| `research/compact-runtime-20261003/` | Exact representation parity and kernel measurements | SDK v0.2.14-experimental |
| `research/compact-native-20261003/` | Actual paired Gooo generation and compiled execution | Compiler and independent receipt reader |

Compact files use `gooo/tiny-shared-three-choice-path-model/v1` metadata and
three tensors: 8×256 input weights, eight biases and 2×8 output weights.
The total feature input is 768 values; a caller workspace holds 24 hidden
values and eight mask scores. Go loads `model.json` and its adjacent weights.

With a compatible Gooo binary and an authored three-choice source/plan:

```sh
hf download asketeddy/gooo-shared-judgment-tiny-v1 \
  --revision 985999a89caba6a31cc7147f66ba29a5ce76a1d9 \
  --include 'research/compact-runtime-20261003/models/fp32/*' \
  --local-dir ./gooo-models

gooo body-codegen --json --activity ChoosePath \
  --path-plan plan.json \
  --path-model gooo-models/research/compact-runtime-20261003/models/fp32/model.json \
  --path-step-attempts 1 --path-feedback-rounds 7 source.gooo
```

`source.gooo` and `plan.json` above are caller inputs. Complete measured examples
and expectations are in the native evidence archive. The
[compiler integration guide](https://github.com/kimjooyoon/meta-ontology-go/blob/dev/docs/three-choice-path-model.md)
describes the input contract. Offline PyTorch performs training/export; the Go
SDK performs local inference and path search.

## Decision quality

Each development view has 16 finite expectations. This table ranks all eight
masks from one initial prediction and evaluates their stored finite targets.
Adaptive failure feedback is measured separately in actual codegen.

| Export | First choice complete /512 | Initial cases matched /8192 | Extra ranked attempts | Complete by budget 4 /512 | EN/KO mask disagreements /256 | Same-mask both wrong /256 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Dense FP32 | 95 | 2265 | 1572 | 289 | 256 | 0 |
| Shared FP32 | 113 | 2592 | 1469 | 319 | 255 | 0 |
| Dense PTQ | 74 | 1967 | 1595 | 285 | 45 | 180 |
| Shared PTQ | 80 | 2052 | 1439 | 336 | 136 | 98 |
| Dense QAT | 82 | 2050 | 1584 | 293 | 240 | 12 |
| Shared QAT | 89 | 2191 | 1512 | 327 | 255 | 1 |

Shared FP32 improves first-choice completion **18.55% → 22.07%**, finite-case
matches **27.65% → 31.64%**, and extra ranked attempts **1,572 → 1,469**.
Its development passing-set negative log likelihood worsens **975.663 → 992.402**.
Comparison/assignment, nested branches and successive assignments need more
extra attempts. The full per-family counts remain in `go-audit.json`.

Paired-language disagreement remains **255/256** for shared FP32. Agreement also
needs a correctness check: dense PTQ gives the same wrong mask on 180 pairs.
The next representation experiments will measure both language consistency and
membership in the valid-path set. Full-budget success here depends on the
authored finite search space containing a passing alternative.

## Input sensitivity diagnosis

A subsequent fixed-weight study varies five authored instruction forms while
keeping the original Gooo source, eight paths and finite expectations constant.
The collector made **92,160 actual Go predictions** across all 3,072 corpus views
and six exports. An independent reader replayed all 92,160 predictions with zero
numerical difference locally and recomputed every condition's finite and bilingual
summary. These replay calls are recorded separately from collection.

The [Linux CI replay](https://github.com/kimjooyoon/gooo-neural-decision-experiments/actions/runs/37049184049)
also reproduced all chosen masks and 30 condition summaries. Its largest
floating-point difference was 0.000001430511474609375, within the declared
0.00001 tolerance. The run's 13 jobs completed successfully.

For shared FP32, the 512 development views give:

| Complete input form | First-choice complete /512 | Extra ranked attempts | EN/KO disagreements /256 |
| --- | ---: | ---: | ---: |
| Original development prefix | 113 | 1,469 | 255 |
| Bare authored instruction | 480 | 38 | 32 |
| Calibration prefix | 92 | 1,372 | 250 |
| Development phrase as suffix | 233 | 662 | 180 |

The bare form matches the format of the original training instructions. Its
result describes a controlled intervention on an already observed cohort;
original-input quality remains 113/512. Production Gooo keeps the complete caller
text. The next comparison will vary training phrasing and positional features
while preserving full input and source binding.

The source v3 intent encoder uses four relative-position buckets for byte
fragments. Added prefixes alter the fragments, their buckets and normalization.
The study identifies sensitivity to this combined change; further experiments
will separate those factors. All six models, all forms and regressions are in
[the report](https://github.com/kimjooyoon/gooo-neural-decision-experiments/blob/main/docs/bilingual-wrapper-audit-results-20261003.md)
and the Hub appendix `research/bilingual-wrapper-20261003/`. Its 30-member archive
includes every full constructed input and prediction, the exact source dataset,
six small existing models and the independent reader. The appendix adds zero
training updates. The model weights at their existing paths remain unchanged.

## Actual Gooo generation and execution

The latest compact study used 16 bilingual views, three model variants and two
representations, for **96 generations** and **298 real model predictions**.
Of those predictions, **202** used finite-failure feedback. Each generation was
immediately followed by a native build and two executions: **192 compiled runs**.

All **2,304 supplied finite expectations** passed, producing 4,608 ordered
values. Of these expectations, 768 use inputs absent from the current selection
suite; their relationship to model training remains part of the dataset scope.
All **48 unseeded expanded/compact pairs** retain matching generated source,
search/feedback semantics and ordered outputs.

An independent compiler reader consumed all 96 runtime receipts, 596 progress
records and 202 feedback records. The first unresolved completeness dimension
is `permission_boundary` throughout. Gooo keeps each observation axis and its
remaining questions alongside the finite functional results.

This compact study uses compiler `7db19b6bc9a2909aa059f1265d39a539c3573a57`,
SDK v0.2.14-experimental and Go 1.27.1 on local darwin/arm64. The earlier
architecture study separately made 320 predictions in 96 generations.
[Earlier quality/native report](https://github.com/kimjooyoon/gooo-neural-decision-experiments/blob/main/docs/shared-three-judgment-results-20261003.md)
and [compact native report](https://github.com/kimjooyoon/gooo-neural-decision-experiments/blob/main/docs/compact-shared-native-results-20261003.md)
retain both experiments with their own identities.

## Storage, speed and CPU observations

| Compact variant | Weight bytes | Resident tensor bytes | Native-study prediction median | Fresh-process codegen median |
| --- | ---: | ---: | ---: | ---: |
| FP32 | 8,288 | 8,288 | 23.709 µs | 28.927 ms |
| PTQ ternary | 446 | 2,096 + 8 scale bytes | 25.000 µs | 27.765 ms |
| QAT ternary | 446 | 2,096 + 8 scale bytes | 30.208 µs | 29.511 ms |

Caller workspace is **3,200 bytes**. Five trits fit in one byte, giving 1.6
stored matrix bits per weight; biases and scales have their own storage.
The runtime decodes matrix values into int8 arrays and computes with int8/float32.
Valid warmed kernel calls allocate zero heap objects in the measured contract.

The separate kernel audit checks all 10,739 frozen states across three variants:
**64,434 actual predictions**, with bit-for-bit agreement in features, hidden
values, logits, probabilities and chosen mask. Compact conversion preserves the
existing shared model; this step performs zero optimizer updates.

Native timings have 16 generations per representation/variant. Prediction timers
exclude loading, startup, code generation and builds. QAT's complete codegen
median increased from 29.237 to 29.511 ms after compact conversion. Cold-cache
outliers remain recorded, including the first 6.45-second native build.

Codegen process CPU/wall-time medians are **83.7–84.6% of one core**. Generated
program RSS medians are **4.20–4.23 MiB**, covering the compiled arithmetic child.
Inference-process RSS and whole-host CPU increase were unmeasured in this study.
Retained serving and additional hardware need their own controlled measurements.

## Reproducibility and development

Every public evidence bundle has a closed byte inventory. The compact native
archive has 684 members, 3,040,850 compressed bytes and 12,474,521 decoded bytes.
Original failed preflights, negative variants, complete inputs and measurement
outliers are retained. Public packaging checks private paths and credential
patterns, including decoded embedded parent receipts.

Training source: `0f3249596b4dacdb34241e3b5e8b4cafac58b0df`.
Original independent Go audit: `8398f1994d65b75af0483db3fefd637eec99ced4`.
Compact native collector: `7e9be4489d15663cbac8f3e71bc02d2e4649dfe5`.
Immutable compact model edition: `985999a89caba6a31cc7147f66ba29a5ce76a1d9`.
Immutable native appendix edition: `37db3a1f8a06669081a370ee1e7b23a64cbc00fb`.

Next work prioritizes Korean/English intent alignment, more expressive Gooo
construction, per-axis before/after observations, and measured serving costs.
Repeated successful structures may later become reusable language abstractions.
The dated research protocols define each experiment's budgets and acceptance
criteria; the compiler retains deterministic continuation between model versions.

## Research acknowledgments

- **Solar-Lezama et al., SKETCH (2006):** partial programs completed under a
  specification inform our view of bounded construction.
  [Combinatorial Sketching for Finite Programs](https://people.csail.mit.edu/asolar/papers/asplos06-final.pdf).
- **Ellis et al., DreamCoder (2020/2021):** neural program search and learned
  reusable abstractions inform the longer-term language/model direction.
  [Paper](https://arxiv.org/abs/2006.08381).
- **Laya / ConvAI Innovations:** its structured decision interface was used in
  our early Gooo experiments. The present weights use fresh initialization and
  Gooo-specific training. [Project](https://huggingface.co/convaiinnovations/laya).
- **Ma et al., BitNet b1.58 (2024):** ternary-weight research motivates exploring
  low-storage models with explicit quality and runtime measurements.
  [Paper](https://arxiv.org/abs/2402.17764).
- **W3C PROV-O (2013):** entity, activity and agent relations inform how we trace
  inputs, construction and observed results. [Recommendation](https://www.w3.org/TR/prov-o/).
