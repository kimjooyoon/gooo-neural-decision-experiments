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
route through it, and the compiler checks the assembled body and generates Go.
A Go experiment runner then builds and executes the resulting program.

Like a workshop, the plan describes what the parts mean and how they may fit.
The model helps choose the next assembly; observed test failures guide another
attempt. A receipt connects the original intention to the choices and result.

| Component | Public home |
| --- | --- |
| Introductory Wiki in Korean | [Gooo: language, models, metrics and research](https://github.com/kimjooyoon/meta-ontology-go/wiki) |
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

문장 앞의 도입 표현에 민감한 문제를 확인한 뒤, 전체 입력을 유지하는 네 가지
새 모델을 학습했습니다. 기존 개발 입력에서 FP32의 첫 경로 충족 수는 대조군
113/512, 문장 조각의 빈도를 쓰는 조건에서 368/512였습니다. 표현을 다양하게
학습한 조건과 삼진 양자화에서는 후퇴한 결과도 있었습니다. 새 결과는 아래
연구 부록에 있으며, 기존 모델의 코드 생성·실행 관측은 해당 실험별로 남아 있습니다.

새 모델을 arm64와 Linux에서 대조한 결과, 첫 선택은 같았고 후순위까지 포함한
272개 후보 순위에 차이가 있었습니다. 이후 반올림 시점을 명시한 규칙으로
147,456회 호출을 대조했고, 18,432개 입력 쌍의 계산값과 전체 순위가 모두
일치했습니다. 새 계약은 SDK v0.2.15로 공개했고, 로컬과 Linux에서 각각
18,432개 입력·36,864회 판단으로 연구 결과를 재현했습니다. 이어서 SDK v0.2.15를
연결한 컴파일러로 400회 생성과 800회 실행을 완료했고, 9,600개 기대값이
모두 일치했습니다. 실제 모델 판단은 816회였으며, 모델을 끈 경로도 결정론적으로
완료했습니다. 아래에 새 관측과 기존 V3 모델 사용 예시를 각각 연결했습니다.

## What we want this to contribute

The [2026-10-03 research update](https://github.com/kimjooyoon/gooo-neural-decision-experiments/blob/main/docs/agile-language-research-20261003.ko.md)
narrows the next work to useful bounded language features: choosing informative
execution inputs, preserving operation order, and reusing small typed assemblies.
An additive Go SDK probe-ranking API explores the first step with zero model
calls or training updates. Its compiler integration now has a
[24-generation paired pilot](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/path-observation-loop-20261003):
the frozen compact bag-original FP32 judge made 24 actual predictions across the
model-enabled arms, and all configurations produced 48 compiled runs. A declared
Gooo reference activity supplied one new observation before construction. The
eight oracle-enabled generations matched 48/48 supplied expectations; controls
remain in the publication, with 84/144 matched across all 24 generations.
The original sparse selection case passed in every arm. This is one authored
task with two language views and two repetitions, with zero weight updates.
The Korean view needed extra search, and oracle-enabled generation added cost.
[Compiler PR 1160](https://github.com/kimjooyoon/meta-ontology-go/pull/1160) tracks
integration and deployment; the pilot pins compiler `2f02d244` and SDK v0.2.16.
Model artifacts and the existing task scores below remain attached to their
original observations.

An [additional paired compiler experiment](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/path-observation-reuse-20261003)
uses the same frozen judge with SDK v0.2.17 and compiler `87d8afae`:
**96 generations, 120 actual model predictions and 192 compiled executions**.
The compiler can retain candidate outputs and compare new reference observations
against them. Probe evaluations fall from 33 to 14 per oracle request, and all
24 fresh/reuse pairs preserve generated source, choices and runtime values.
Both oracle modes together meet 288/288 finite expectations; the full control
comparison is 396/576. This remains one authored task with Korean/English views,
six repetitions and zero training updates.

With the model enabled, observation-phase median falls from 0.603 to 0.333 ms,
while complete codegen changes from 9.213 to 9.372 ms. Both oracle modes have
18.83 MiB median peak RSS. Process CPU changes from 92.35% to 93.02% of one core;
whole-host utilization change is unmeasured. Reuse saves candidate evaluations;
the end-to-end latency result is slightly slower in this collection. The compiler
in that experiment still searches after a single candidate remains.
[PR 1162](https://github.com/kimjooyoon/meta-ontology-go/pull/1162) tracks this
opt-in integration. The existing model weights are preserved.

The [next direct-projection experiment](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/path-observation-resolution-20261003)
completes **120 generations and 240 compiled executions**, with **120 actual
model predictions in control arms**. Complete observation can now select the
single surviving candidate before model loading or search. Resolved arms make
zero predictions, meet 144/144 finite expectations and match all 24 corresponding
cached-search bodies and runtime results. All oracle arms meet 432/432; the full
comparison including sparse-case controls is 540/720. Compiler
[`9158c3cd`, PR 1164](https://github.com/kimjooyoon/meta-ontology-go/pull/1164)
adds this optional route with explicit source replay and skipped-work records.

For model-requested fresh processes, cached search vs direct projection has
9.491 vs 8.267 ms median codegen, 18.77 vs 17.29 MiB peak RSS, and 92.45% vs
88.01% CPU relative to one core. All samples are retained, including a slow
initial control. This is one task, two language views and six repetitions with
unchanged weights; host utilization change and broader speedup remain unmeasured.
The model supplies a preference when choices remain, and a resolved source
observation can complete this narrow construction without another prediction.

The language carries the plan, and the model supplies small local judgments.

An [instruction-order feature preflight](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/intent-order-preflight-20261003)
uses this frozen judge for 64 actual predictions. Four authored arithmetic pairs,
two languages and four wrapper forms produce 32 pairs with identical V4 features
and identical model predictions despite different requested operation orders.
An experimental 128-byte directed-clause sketch distinguishes those 32 pairs;
a repeated-clause counterexample still aliases and is retained. The feature
kernel measures 372–619 ns/op with zero heap allocations on M4 / Go 1.27.1.
It has no trained model head or new accuracy score, and this preflight includes
zero optimizer updates or native Gooo executions. The current weights and input
ABI remain attached to their original experiments.

Our aim is to make intent, construction and observed behavior travel together
as a program evolves. We measure complete finite behavior, partial coverage,
extra attempts, unresolved obligations, time and memory. These measurements
help decide which language features and model changes to develop next.

For the latest study, a first path is complete when it satisfies all 16 supplied
examples for that input. A result of 368/512 therefore describes that specific
authored task collection. Larger programs, broader types and reusable learned
constructions are the next language questions.

Model probabilities describe which permitted path the judge favors. Finite
coverage counts the supplied examples that pass. An unresolved observation
stays `UNKNOWN` in its own receipt dimension. Keeping these meanings visible
helps the system identify the next useful experiment.

## Choose an entry point

| What you want to do | Compatible starting point |
| --- | --- |
| Generate with SDK v0.2.14 or later | Pinned compact V3 bundle in the usage example below |
| Explore the new V3/V4 arithmetic contract in Go | [SDK v0.2.15](https://github.com/kimjooyoon/gooo-decision-runtime/releases/tag/v0.2.15-experimental) and [explicit-arithmetic models](research/full-input-separate-20261003/README.md) |
| Generate with the new V4 contract | Integration on main `fc0e99c4` through [merged PR 1159](https://github.com/kimjooyoon/meta-ontology-go/pull/1159); [native evidence](research/full-input-native-20261003/README.md) |
| Follow the current language work | [Language guide](https://github.com/kimjooyoon/meta-ontology-go/blob/dev/docs/language-direction.ko.md), [experiment repository](https://github.com/kimjooyoon/gooo-neural-decision-experiments) and the dated appendices here |

SDK release source `59c8d342da4475506b90954469aa201f85cadeb3` passed complete
replay on darwin/arm64 and linux/amd64: each made 36,864 predictions over 18,432
frozen inputs, matching intermediate values, probabilities and full rankings.
[Both original SDK reports](research/full-input-sdk-20261003/README.md) and
[Linux CI](https://github.com/kimjooyoon/gooo-decision-runtime/actions/runs/37070241916)
are public. This stage performed zero training updates. The native integration
observation below follows that library replay.

## New native generation and execution observation

On sixteen known bilingual tasks, four training arms × three precisions × two
layouts plus a disconnected arm completed **400 generations and 800 compiled
runs**. All **9,600 supplied expectations** and **192 representation pairs**
matched. An independent reader checked the original source/result bindings,
1,760 progress records and 432 feedback records. Total actual predictions were
816; the disconnected requests made zero model calls.

Bag-original FP32 completed the first path for 16/16 requests in this integration
slice. Its broader development result stays 368/512. Compact median prediction
time was 8.33 µs, whole codegen 10.00 ms and compiler peak RSS 17.78 MiB. Process
CPU was 78.84% of one core; host-wide utilization change was unmeasured. The
disconnected arm completed after 96 extra candidates, with 9.80 ms median codegen.
This task size shows reduced search attempts and similar process latency.

[Full native results, every variant and original records](research/full-input-native-20261003/README.md)
retain the timing conditions and known-task scope. All 400 receipts keep
`permission_boundary` unresolved. The observation used compiler `e461c1d` and
SDK v0.2.15, with zero training updates and no new default checkpoint. Compiler
integration merged to dev in PR 1158 and main in PR 1159 above.

## New full-input research exports — 2026-10-03

The [full-input appendix](research/full-input-initial-20261003/README.md) contains
four freshly initialized shared judges, each exported as FP32, PTQ and QAT in
expanded and compact layouts. Complete source-derived input and Korean/English
instructions are retained. The comparison changes position-based features versus
global fragment counts, and original versus varied training wording.

| FP32 arm | Original first-path complete /512 | Extra ranked attempts | Both KO/EN valid /256 |
| --- | ---: | ---: | ---: |
| positioned-original | 113 | 1,469 | 1 |
| positioned-varied | 266 | 528 | 75 |
| bag-original | 368 | 186 | 180 |
| bag-varied | 320 | 326 | 132 |

Bag-original FP32 reaches 358/512 and 368/512 on the two preregistered new wording
forms. Bag-varied falls to 173/512 on the new suffix. Bag-original PTQ and QAT
reach 94/512 and 287/512 on original inputs. Its global fragment representation
also maps two distinct operation orders to equal features; the appendix retains
that explicit counterexample. These are previously observed authored source tasks.

The Go audit reconciled all 6,400 MPS updates and made 25,824 real predictions
across export parity, compact parity, calibration and three development forms.
All twelve exports and complete raw evidence are public. Optimization took
42.18 seconds; average process CPU was 49.83% of one core and peak RSS was
1,809,678,336 bytes. CPU usage describes the optimizer process. Model inference
and native generation have their separate measurement above.

V4 compact artifacts use `triple_semantic_context_bag_v4_joint_v1` and the
[research Go runtime](https://github.com/kimjooyoon/gooo-neural-decision-experiments/commit/d23bd7e9e9fa3f197ffb5ee8d10872ad982e29ca).
The matching SDK v0.2.15 is published with complete numerical replay and the
native integration observation above. Remaining split/form comparisons and new
intentions are subsequent steps. Later historical sections retain the measurements
of their pinned earlier models.

The subsequent [Linux comparison](research/full-input-platform-20261003/README.md)
retained the same first selected mask on all 18,432 development rows, with 272
different complete candidate orders. Small score deltas changed partial-completion
curves, so the exact cross-platform comparison failed. Its complete observations
are public. The [explicit-arithmetic follow-up](research/full-input-separate-20261003/README.md)
then made 73,728 predictions per platform with unchanged weight bytes and zero
training updates. All 18,432 paired inputs now have identical hidden/logit/
probability bits and complete rankings under `float32_separate_v1`. Original
first-path completeness and extra ranked attempts stay unchanged; partial
coverage shifts in both directions compared with legacy arm64 arithmetic.
Converted metadata and complete four-lane journals are public in that appendix.
The SDK replay reproduces these observations. The native stage above subsequently
completed generation and execution with this arithmetic contract.

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

## Original V3 architecture, training and compatible files

The shared model reuses a **256 → 8 → 2** local network over three ordered
decisions: **2,072 trainable parameters**. The dense control has 18,656.
Both were freshly initialized using the same Go initializer recipe and trained
on Gooo-derived data. Each arm completed 800 FP32 and 800 QAT updates on local
MPS, totaling **3,200 optimizer updates**. PTQ exports come from the FP32 models.
That total belongs to the original shared/dense comparison. The later four-arm
full-input study above adds 6,400 updates and twelve exports in its own appendix.

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
| `research/full-input-initial-20261003/` | Four-arm study: full inputs, twelve exports and original observations | Research runtime and SDK v0.2.15 |
| `research/full-input-platform-20261003/` | Retained Linux replay and complete ranking diagnosis | Offline Go diagnostic reader |
| `research/full-input-separate-20261003/` | Explicit arithmetic, paired observations and converted metadata | Research runtime and SDK v0.2.15 |
| `research/full-input-native-20261003/` | 400 generations, 800 compiled runs, 192 matched pairs and independent receipt consumption | Compiler `e461c1d` with SDK v0.2.15 |
| `research/full-input-sdk-20261003/` | Complete local/Linux SDK replay reports | SDK release source `59c8d34` |

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
The subsequent full-input study above measures both language consistency and
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
text. The completed initial full-input comparison above varies training phrasing
and positional features while preserving full input and source binding.

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

Next work connects the published SDK to native generation and execution while
continuing full-input evaluations and operation-order representation work. Korean/English
intent alignment, more expressive Gooo construction, per-axis before/after
observations and serving costs guide the wider development.
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
- **Balog et al., DeepCoder (2016/2017):** learned program properties guide
  synthesis search. It informs our question of how much a small judgment can
  reduce construction attempts. [Paper](https://arxiv.org/abs/1611.01989).
- **Laya / ConvAI Innovations:** its structured decision interface was used in
  our early Gooo experiments. The present weights use fresh initialization and
  Gooo-specific training. [Project](https://huggingface.co/convaiinnovations/laya).
- **Ma et al., BitNet b1.58 (2024):** ternary-weight research motivates exploring
  low-storage models with explicit quality and runtime measurements.
  [Paper](https://arxiv.org/abs/2402.17764).
- **W3C PROV-O (2013):** entity, activity and agent relations inform how we trace
  inputs, construction and observed results. [Recommendation](https://www.w3.org/TR/prov-o/).
