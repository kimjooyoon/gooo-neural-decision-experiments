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

Updated 2026-10-04. This model ranks **eight permitted Gooo body paths** built
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

### 최근 재현 검사

과거 계산 방식의 macOS·Linux 기록을 각각 고정 원본과 대조하는 Go 검증기를
추가했습니다. 새 실행은 환경마다 18,432개 개발 기록과 24개 compact 형식 모델 파일을
모두 재현했습니다. 원래 두 환경에서 달랐던 요약 101개 항목과 후보 순위 272개는
양쪽 값과 입력 해시를 함께 보존합니다. 검증기는 모델을 호출하지 않으며 새 감사
실행의 Go 추론 호출은 환경마다 25,824회입니다. 원래 가중치와 기대값은 유지합니다.

[Linux 검사 37126899513](https://github.com/kimjooyoon/gooo-neural-decision-experiments/actions/runs/37126899513)의
19개 작업이 통과한 뒤 연구 PR #1을 병합했습니다. 새 산술의 플랫폼 간 검사는
계속 별도로 실행합니다. [새 원본과 재현 방법](research/legacy-platform-replay-20261003/README.md)을
공개했습니다. 당시 설치된 컴파일러 main `ed2c2cac`는 SDK v0.2.20-experimental을
사용했습니다. 최근 파일 연결과 설치 상태는 아래 갱신에, 이전 연결 실험의
버전·결과는 각 날짜의 기록에 남아 있습니다.

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

## Connecting model files to Gooo — 2026-10-04

The [released Go SDK v0.2.21](https://github.com/kimjooyoon/gooo-decision-runtime/releases/tag/v0.2.21-experimental)
uses bounded model reads and checks the opened regular-file identity and extent.
On Unix, nonblocking/no-follow open prevents the observed replacement-FIFO
writer wait. The SDK's existing non-symlink model rule is retained. Use
`hf download --local-dir` to obtain regular metadata and adjacent weights.
The exact compact FP32 pair at Hub revision
`7c2781cbbcf81edba89a50d0423276c042b082de` was downloaded and checked:
metadata SHA256 `6478334e1e181d0854865d1da54b9c2203eaa0e488f2b05617c92e27f08b755f`,
weights SHA256 `43c503b224a78f23725101e58cdef1ec4647831329eea4de80f0a259a0253d31`.

Compiler [dev PR1206](https://github.com/kimjooyoon/meta-ontology-go/pull/1206)
passed its exact-head CI and independent source proof. Candidate `2420ad19`
completed 128 model-path swaps with zero writer releases/timeouts; 34 regular
completions kept 4,352/4,352 expectations. The standalone shared-model worker
kept 512/512 in concurrent Korean/English requests and joined both processes.
[Original failures, counts and process CPU scope](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/0e24fffeae993af22032f6694d8a4a118be8eabf/publication/legacy-model-open-20261004).
Main promotion [PR1207](https://github.com/kimjooyoon/meta-ontology-go/pull/1207)
passed its own canonical CI37167954453, independently verified source proof and
live promotion authorization, then normally merged as `4f6c7566`. Both CLI and
worker are freshly installed from that clean main with Go1.27.1 / SDK v0.2.21.
Fresh installed swaps completed 128 requests with zero writer releases/timeouts;
35 regular completions kept 4,480/4,480. Separate ordinary runs kept 1,024/1,024
and concurrent shared-model worker requests kept 512/512 with joined EOF.
The older d1bfd273 / SDK v0.2.20 wait remains recorded as the baseline.

The research comparator now binds each recorded source inventory to its own
immutable Git revision. Complete local/Linux replays each made 73,728 actual
predictions over 18,432 paired inputs. Explicit arithmetic kept all values,
rankings and finite curves; fourteen source changes stay visible. Legacy
arithmetic keeps 272 ranking and 38 finite-outcome differences between platforms.
Research CI37168510833 completed SUCCESS, and its arithmetic archive was
independently downloaded, hash-checked and fully consumed.
[Source deltas and original differences](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/arithmetic-source-delta-20261004).

### A direct file-based example

With Go1.27.1, a compatible compiler and the compiler repository as working
directory, the existing compound source, plan and expectations can run directly:

```sh
hf download asketeddy/gooo-shared-judgment-tiny-v1 \
  --revision 7c2781cbbcf81edba89a50d0423276c042b082de \
  --include 'research/compact-runtime-20261003/models/fp32/*' \
  --local-dir ./gooo-models

gooo body-path-run \
  --source examples/body-codegen/typed-path-compound.gooo.fixture \
  --activity Combined \
  --path-plan examples/body-codegen/typed-path-compound-plan.json \
  --cases examples/body-codegen/typed-path-runtime-cases.json \
  --model gooo-models/research/compact-runtime-20261003/models/fp32/model.json \
  --repeat 2 --timing --out compound-model-results

gooo body-path-run --verify-timing --out compound-model-results
```

Choose a fresh output path. Omitting `--model` uses deterministic construction.
Read assembled Go in `run-N-generated.go`, current case outputs in
`run-N-runtime.json`, and request status plus the finite numerator/denominator in
`summary.json`. Setup failures and unobserved cases have their own recorded state.
The clean candidate made two actual shared-model predictions and four native
runs, keeping 6/6 expectations in this existing example. First/repeated responses
were 884.413250/70.435292ms, covering construction and execution while excluding
initial model preparation and saving/output. Weights and training remain unchanged.
Fresh main `4f6c7566` also kept 6/6 in two constructions, two predictions and
four native runs. Its original first/repeated responses are
1,188.190292/75.380833ms, and the saved-file timing verifier passed. The earlier
candidate timing stays attached to its own run. Current installed concurrent
worker CPU averaged 72.69% of one core during a 1.796446292s process interval;
this includes setup, native children and EOF joining. Host CPU and model-only
RAM remain unobserved.
[Commands, saved outputs and int64 readback](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/direct-body-quickstart-20261004)
and [Korean input/error guide](https://github.com/kimjooyoon/meta-ontology-go/wiki/Reading-Input-Files)
give the full scope.

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

### Smaller source-derived assembly recipes

The [source-recipe pilot](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/82afdff/publication/source-recipes-20261003)
uses the unchanged `compact/bag-original/fp32` full-input export in actual native
construction. Gooo derives the typed base from its source body; a small recipe
names the permitted structural choices. Compiler
[`3a52232d`, PR 1168](https://github.com/kimjooyoon/meta-ontology-go/pull/1168)
adds that input form and merged to dev as `79e7e20c`. Its main promotion is
tracked in [PR 1169](https://github.com/kimjooyoon/meta-ontology-go/pull/1169).

Across 24 generations and 48 compiled runs, all 12 recipe/full-document pairs
have equal expanded plans and emitted code. Compact authored JSON falls from
1,538 to 569 bytes (63.0%). Six real joint predictions each judge three choices;
there are no training updates. Oracle/resolution arms satisfy 72/72 independent
runtime expectations; sparse single-example controls satisfy 12/72, making the
complete comparison 84/144. One authored subtraction task and repeated requests
define this finite scope.

Own-model search has median whole-process generation time 8.143 ms for the full
document and 8.598 ms for the recipe, including expansion. Peak RSS medians are
17.33 and 17.64 MiB; CPU relative to one core is 86.78% and 86.53%. Each cell has
three samples. Whole-host utilization change and broader task accuracy are
unmeasured. The pilot reduces authored representation while retaining the added
processing cost. Initial collector accounting failure and all controls are public.

The language carries the plan, and the model supplies small local judgments.

The [constant-body follow-up](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/de2b2e8/publication/constant-recipes-20261003)
uses the same frozen model with SDK v0.2.18 and compiler `b742ba75` in
[PR 1170](https://github.com/kimjooyoon/meta-ontology-go/pull/1170). It repairs a
language gap: bodies that never read their declared input can now use typed
recipe construction. Six generations and twelve compiled runs met 36/36 finite
expectations, including both integer endpoints, with three actual predictions
and zero training updates. Three repeats per arm cover one authored body.

For that constant body, deterministic generation took a median 9.209 ms and
17.42 MiB peak RSS; model-connected generation took 9.731 ms and 19.31 MiB.
Process CPU medians were 90.56% and 97.96% of one core; host utilization change
was unmeasured. Isolated prediction median was 7.708 microseconds. Model ranking
increased evaluated candidates from five to seven, so this task favors the
deterministic route. The first proposed body failed in both arms; finite search
reached the same successful source. All failed attempts and process records are
retained. The result informs where a small model helps and where ordinary
language machinery is already sufficient.

### Condition chains using the same model

The [condition-chain pilot](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/855ae1c318160a8134494bfb4650ea557220fe91/publication/condition-chain-recipes-20261003)
extends source recipes to `if ... else if ... else`. Compiler development source
`bf51d65a` lowers that form into existing typed nested branches, retaining
condition order, local scope and source binding. Its branch is public; main
deployment is tracked separately in the [compiler wiki](https://github.com/kimjooyoon/meta-ontology-go/wiki/Current-Status).

One authored clamp, one bilingual intention and three repeats per arm give six
generations and twelve compiled runs. All 42 finite expectations match, including
24 selection-disjoint expectations and both int64 endpoints. The frozen own
model makes three actual joint predictions; training updates are zero. Both arms
choose the first candidate and emit equal source. Seven of eight declared
combinations remain unattempted per request.

Deterministic/model generation medians are 7.958/9.227 ms, process peak RSS
17.45/17.91 MiB and one-core-relative CPU 90.64/91.58%. Prediction median is
7.917 microseconds. The first deterministic process takes 367.183 ms; all samples
are public and its cause was not isolated. Cache state and whole-host CPU change
were not instrumented. This already complete body favors deterministic assembly.
The useful result is that a common source form now reaches both construction
routes using the existing small runtime and unchanged weights.

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

The [native order follow-up](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/eaeca70c8f6b4068089804d387faf31feea43ead/publication/native-order-20261003)
uses main compiler `729482ed` with four operation pairs, Korean/English views and
both requested orders. Across 96 generations, 192 compiled runs and 64 actual
predictions, first-attempt functional completion is 8/16 requests for deterministic
construction and 6/16 for each frozen positioned/bag model. With up to eight
candidate attempts, all three arms complete 16/16 requests and 128/128 finite
expectations each. All controls together meet 554/768 expectations. Training
updates are zero, and the two models also have different learned weights.

The positioned model distinguishes all eight reversed-instruction pairs in its
features and probability distributions, while its first masks remain unchanged.
Bounded-search generation medians are 8.129 ms deterministic, 9.352 ms positioned
and 8.861 ms bag; total candidate evaluations are 24, 53 and 46. The first
deterministic process took 574.741 ms and is retained. These are one sample per
authored request, with process costs and limits in the raw publication.

A [source-order counterexample](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/7400a97d4b14f76fabefdfefa82f82a1ef297faa/publication/root-order-source-alias-20261003)
reverses the source assignments while keeping the instruction fixed. The two
correct source-relative choices are opposite, but the local root-order model
inputs are identical. Both calls choose reverse: native outcomes are 8/8 and 0/8.
The shared local judge lacks a distinction needed for this pair. These observations
leave the released weights unchanged and retain bounded deterministic TDD as the
working completion path.

The [source-order preflight](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/f663ed3061fdb000e74a1346623d86e92bb38aaa/publication/source-order-preflight-20261003)
now describes two source-derived assignment operations in **48 bytes per
alternative**. Compiler `ba00850f` exposes its validated plan through the explicit
`body-context --include-plan` option in [PR1174](https://github.com/kimjooyoon/meta-ontology-go/pull/1174).
Across 48 exports from four authored families, the new description distinguishes
24/24 source-reversal pairs that have identical existing V3 local root inputs.
Renaming locals and commuting operands each preserve 16/16 description pairs.
Korean/English intent changes preserve 24/24 source-description pairs; this last
check measures the source channel, not language understanding.

Five M4/Go1.27.1 samples measure 28.02–29.31 ns/op and zero allocations for the
descriptor alone. The complete context exports take a median 6.119 ms per fresh
process; parsing, binding and validation remain separate costs. These observations
add no model predictions, native runs or training updates. The new representation
is isolated from released V3/V4 weights, and its trained-model quality is still
unmeasured. It covers two simple integer updates; nested expressions and branches
require further work. The initial collector digest-format failure is retained and
covered by a regression. The next learning study must evaluate actual assembled
bodies and additional attempts with the same deterministic-search budget.

Our aim is to make intent, construction and observed behavior travel together
as a program evolves. We measure complete finite behavior, partial coverage,
extra attempts, unresolved obligations, time and memory. These measurements
help decide which language features and model changes to develop next.

For the full-input learning study, a first path is complete when it satisfies all 16 supplied
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
