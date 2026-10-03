---
license: mit
language:
  - en
  - ko
tags:
  - gooo
  - golang
  - program-synthesis
  - candidate-ranking
  - experimental
---
# Gooo whole-candidate order judge

A small, independently initialized model for choosing which permitted **Gooo
program body to try first**. It reads a complete Korean or English instruction
and the ordered operations of each candidate. Gooo supplies a typed assembly
space; this judge ranks complete constructions and a bounded search checks
finite examples.

Think of the compiler as a workshop with eight assembled pieces on the bench.
The model suggests which piece to try; actual measurements determine whether it
fits. Equal pieces can reuse an earlier measurement.

## Artifact and integration status

- One 128×32 bilinear matrix: **4,096 parameters, 16,384 FP32 bytes**.
- Full UTF-8 instruction up to 512 bytes, positional byte-bigram/trigram features.
- Eight complete candidates, each described by two ordered integer operations.
- One root-order and two operand-order decisions; add/subtract/multiply,
  local/input/constant leaves, model constants -16..16.
- `model.json` and `weights.bin` load with the custom Go `orderjudge.Load` API.
- The 128-byte prediction workspace excludes feature arrays, preparation,
  artifact loading, program objects and receipts.
- CPU fit and local Go inference. This architecture currently has FP32 weights.
- Current installed compiler: main `fc854ef49be02644006c956c652e60cc8d1d06bf`,
  with `body-path-run` for direct source/recipe/case files and immediate execution.
  Known English/Korean arithmetic arms passed 1,024/1,024 finite expectations
  in eight generations, four actual predictions and 16 compiled executions.
  Conditional assembly and original unmet outcomes are also preserved separately.
  [Current installation evidence and complete costs](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/body-path-file-cli-20261003).
- The released Go SDK v0.2.20 adds preparation reuse to the existing direct
  in-compiler generation route.
  The [native collection](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/order-judge-native-20261003)
  uses compiler commit `9d570ab1` in [PR #1176](https://github.com/kimjooyoon/meta-ontology-go/pull/1176).
- [Main PR #1179](https://github.com/kimjooyoon/meta-ontology-go/pull/1179) is merged
  as `8117fbaefac490a28e78c956bf1d40aed1372608`, with SDK v0.2.20 preparation reuse.
  The clean installed compiler
  reproduced eight existing Korean/English requests in 16 generations, 8 model
  calls and 32 compiled executions: 128/128 finite expectations passed.
  [Installation and CI evidence](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/order-prepared-installed-main-20261003).

The runtime is in the
[Go SDK v0.2.20](https://github.com/kimjooyoon/gooo-decision-runtime/tree/v0.2.20-experimental/orderjudge),
extracted from the separately pinned
[research implementation](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/8a41bb0f825dfd3f950101b33491581492c740f6/internal/orderjudge).
The compiler can export its source-bound expanded plan with `body-context --include-plan`.
The separately published [shared judge](https://huggingface.co/asketeddy/gooo-shared-judgment-tiny-v1)
has earlier native integration and different feature/weight contracts.

## Training and lineage

The matrix starts at zero and is trained in Go on **64 authored Gooo requests**,
with 400 full-batch updates, learning rate 2 and L2 0.0001. Training labels are
sets of candidates meeting separately authored arithmetic references on three
selection inputs. Held development requests do not enter the fit. The first fit
and all regressions are retained. CPU fit time was **89.61 ms** on an Apple M4.
No GPU updates were used for this matrix.

There are no Laya checkpoint weights, teacher outputs, private user messages or
repository secrets in this model. Authored English/Korean arithmetic templates,
typed Gooo plans, constants and finite reference outputs form the dataset.
The complete 160-request dataset, original sources, features, 400 loss records,
all predictions and manifests are in the
[public evidence bundle](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/8a41bb0f825dfd3f950101b33491581492c740f6/publication/order-judge-initial-20261003).

## Results and costs

Each development request has eight finite evaluation cases, including five inputs
not used for candidate selection. Families are represented in training; these
are small controlled development contrasts.

| Development contrast | Requests | First complete, deterministic / learned | Body evaluations with budget 8, deterministic / learned / learned + equal-body reuse |
| --- | ---: | ---: | ---: |
| New instruction template | 32 | 16 / 26 | 48 / 48 / 38 |
| Renamed local + commuted operands | 32 | 16 / 30 | 48 / 36 / 34 |
| New constants | 32 | 16 / 26 | 48 / 52 / 38 |
| Total | 96 | **48 / 82** | **144 / 136 / 110** |

All budget-8 arms complete 96/96 requests and 768/768 finite evaluation cases.
Deterministic equal-body reuse also uses 144 evaluations. The reuse follow-up
was designed after viewing the original observations and uses unchanged weights.
It is explicitly a development experiment on an already observed cohort.

The initial collector made 320 post-fit predictions. A separate replay made
**480 actual predictions and 960 actual bounded searches**, plus 320 baseline
SDK searches. It reconstructed the first observations exactly and recorded every
candidate evaluation. A later SDK replay reproduces the complete records on
Linux amd64 and macOS arm64, with only the two timing fields excluded.

On M4/macOS arm64, Go1.27.1:

- Prediction alone: median **1.875 µs** on the 160 initial timed inputs.
- Budget-8 complete search wrapper: median **322.04 µs** deterministic,
  **340.83 µs** learned, **339.25 µs** learned with reuse.
- Whole original source-collection/fit/evaluation process: 1.94 s wall time,
  1.60 s CPU time (about 82.5% of one core on average), 43.39 MiB maximum RSS.
- Whole follow-up replay: 0.91 s wall, 0.59 s CPU, 47.66 MiB maximum RSS.

Prediction timing excludes feature extraction and model loading. Process memory
includes the collector and associated work. These measurements show fewer body
evaluations, while preparation and bookkeeping still make learned search slower
in this small cohort. No peak system CPU utilization claim is made.

### Direct compiler follow-up

With the same weights and zero further training, the compiler completed **256
generations, 128 actual predictions and 512 immediately compiled executions** on
the 64 existing new-template/new-constants requests. Budget-1 complete requests
are 32/64 deterministic and 52/64 learned (28/32 English, 24/32 Korean). At budget
8 both routes complete 64/64 requests and 512/512 listed expectations. The model
with exact descriptor reuse evaluates 76 bodies versus the ordinary deterministic
route's 96. All outcomes and full predictions reproduce the frozen study.

Median budget-8 generation wall time is 8.579 ms deterministic versus 9.256 ms
with the model; maximum RSS medians are 17.56 versus 18.60 MiB. OS-reported
per-command CPU usage corresponds to 88.14% versus 92.50% of one core. Model
prediction alone takes 2.75 µs. Each generation starts a fresh compiler process;
native compilation and two runs take another roughly 310 ms. These are process
measurements with a fixed observed development cohort. The
[raw evidence and cost breakdown](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/order-judge-native-20261003)
retain all partial outcomes: 1,706/2,048 finite cases across both budgets/routes.

Within the supported profile, pass `--path-model /path/to/model.json` to
`gooo body-codegen --json --activity Compose --path-plan recipe.json source.gooo`.
Place this model's `weights.bin` beside the metadata. Omit `--path-model` for
deterministic generation. Explicit seeds, model-level batching and feedback are unsupported
by this first model route and produce recorded errors before prediction.

The [runnable Korean example](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/examples/whole-candidate-order)
includes the Gooo source, recipe, finite cases and commands for generation and
compiled execution. It asks for multiplication before addition while the original
body uses the opposite order.

## Run the evidence replay

Use Go1.27.1 and a fresh pair of temporary output directories:

```sh
git clone https://github.com/kimjooyoon/gooo-neural-decision-experiments.git
cd gooo-neural-decision-experiments
git checkout 8a41bb0f825dfd3f950101b33491581492c740f6
unzip -q publication/order-judge-initial-20261003/initial.zip -d /tmp/gooo-order-initial
go run ./cmd/order-judge-replay \
  --input /tmp/gooo-order-initial --output /tmp/gooo-order-replay
```

The archive holds exactly the model published here. The replay loads its weights,
reconstructs features and source candidates, checks every frozen prediction,
executes bounded searches and writes all attempts. `orderjudge.Search` accepts a
nil model for deterministic operation. Its caller supplies a deadline, a typed
plan, finite cases and an attempt budget. Compiler source binding remains the
compiler caller's responsibility.

## Reading the limits

The feature projection is narrow and lossy. Long instructions, branching,
nested operations, multiple locals, other output types and broader domains need
separate scoped experiments. Finite completeness is tied to the listed inputs;
ranking probabilities are uncalibrated. There is no fresh general-language
benchmark or measured 1.58-bit version for this architecture.

The Go SDK now provides `orderprepared.NewRuntime` and `Runtime.Prepare` as two
explicit preparation steps, followed by repeated `Prepared.Search` calls. The
runtime captures a private immutable model; each prepared plan holds eight checked
programs and their fixed features. Each search predicts and checks its current
finite cases. The SDK has no global cache. Compiler integration retains at most
one complete plan per generator and binds source again before reuse.

On 64 already observed requests, 3,840 SDK searches and 1,920 model predictions
matched the original complete selection records, with zero further training.
Budget-8 model search medians were 356.98 µs for the original wrapper, 244.04 µs
for capture + preparation + search per call, and 4.83 µs for retained preparation.
Three observed live-heap deltas for 64 prepared plans and one model were about
1.7 MB. The [full SDK observations](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/order-prepared-sdk-20261003)
include the initial memory-measurement correction and Linux replay. These numbers
measure SDK intervals; compiler generation and native execution have separate costs.
Existing V3/V4 models and their historical numerical comparisons remain preserved.

The [compiler API follow-up](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/order-prepared-native-20261003)
completed 512 generations, 512 fresh predictions and 1,024 immediate native runs.
At budget 8, second-call Generate medians are 1.017 ms for a fresh owner and
0.693 ms for a retained owner; including recipe expansion and constructor gives
2.225 versus 1.627 ms. Both retain the original 64/64 complete requests. Budget-1
outcomes remain 52/64, with every original failure preserved. This compares API
ownership patterns within one compiler revision, with the original weights.
The compiler change is in [PR #1178](https://github.com/kimjooyoon/meta-ontology-go/pull/1178),
merged to dev as `9fc2634720f0a43ca7d72bac46aacba5c77822ce` after canonical CI.
[Main promotion #1179](https://github.com/kimjooyoon/meta-ontology-go/pull/1179)
also passed its checks and source-bound proof, and is merged and installed.
The installed revision and independent execution observations are linked above.

The [retained worker follow-up](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/prepared-worker-native-20261003)
ran 96 generations, 48 model predictions and 192 native executions, matching
768/768 declared expectations. Responses arrived before input EOF. All 16
sequential second model calls reused preparation; alternating two plans in
four-request batches produced no reuse in that collection. Only the latest plan
is retained. These are known examples, with one or four worker slots; batch
intervals include native execution of preceding responses and do not measure
parallel speedup.

For easier integration, [PR #1180](https://github.com/kimjooyoon/meta-ontology-go/pull/1180)
adds `gooo body-path-stream` using the same runner as `gooo-body-worker` and is
merged to dev as `e9d1fd0f87fc1aa95756fd55293421051f6f3650` after canonical CI.
An explicit `--model` retains this model in one process; omission uses
deterministic construction. Local dogfooding passed four generations, eight
compiled runs and 32/32 finite expectations. [Main promotion #1181](https://github.com/kimjooyoon/meta-ontology-go/pull/1181)
passed CI and independent proof verification, and is merged as `cb2892cb583e693190bd68f75eb4df223819f053`.
The clean installed main reproduced four generations, two model predictions,
eight compiled runs and 32/32 expectations, including second-request preparation
reuse. [Installation evidence and runnable instructions](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/body-stream-installed-main-20261003).

The compiler now reuses an original checked projection within recipe
expansion, retaining current-source and fallback checks. With these same weights,
512 additional generations and 1,024 native executions preserved the original
selection, ranking and finite outcomes, including partial results. A historical
timing comparison showed no wall improvement. Four paired process trials then
measured 8,192 fresh decodes: median decoder interval 0.3714→0.2921 ms and
allocated bytes per decode 728,580→571,800. All 4,096 document/plan pairs matched;
751 timing pairs were slower. These are compiler preparation costs; weights and
training are unchanged. [Every observation and scope](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/source-recipe-projection-reuse-20261003).

[Main #1183](https://github.com/kimjooyoon/meta-ontology-go/pull/1183) passed canonical
CI and independent proof verification and is merged and installed as
`3bee55daf8063b0daee3d543e57a528975bc63ee`. Existing English/Korean tasks, the
integrated stream and the Go example completed 24 generations, 12 model calls,
48 compiled executions and 192/192 finite expectations. All source/selection
comparisons matched. [Installation evidence](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/recipe-projection-installed-main-20261003).

### Run generation and execution together

The [Go example](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/examples/whole-candidate-order)
reads a Gooo source, short recipe and finite execution cases, sends one request
to the integrated stream, executes its response immediately, then sends the next
request with input still open. `go run ./cmd/order-example --model ... --out ...`
uses this model; omitting `--model` uses deterministic construction. It saves
every response, generation and runtime report, including finite failures.

Local and Linux collections each completed four generations, two actual model
calls and eight compiled executions with 32/32 expectations. Local model
response times were 51.39 ms initially and 2.06 ms on the repeat; native command
times were 655.81/309.06 ms. The first response can include worker setup. Linux
native times were 5,351.99/182.90 ms; that long first interval is retained.
These are two known sequential requests per route, with uncontrolled cache and
platform differences. [All observations, CPU/RSS and stage costs](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/order-example-immediate-20261003).
The new example's race, vet and actual Linux execution checks passed. No model
weights or training changed.

### Retain one executable and execute current inputs

The compiler [dev #1184](https://github.com/kimjooyoon/meta-ontology-go/pull/1184)
adds `body-path-stream --execute` and per-request `execution_cases`. Each newly
constructed body immediately enters native observation in the same process.
One owned executable is retained; current source/plan/parent replay, executable
hash checks and two current-input executions are performed on every request.
`--retain-native` selects this route in the Go example. Model omission remains
deterministic. Native construction can use bounded parallel workers; one
cancellable gate owns the native workspace and serializes execution.

On the original 64 known EN/KO development sources at budgets 1/8, 512 additional
generations/predictions and 1,024 compiled executions matched every original
ranking, source and finite result: 3,720/4,096 expectations, including partial
failures. There were 384 actual builds and 128 artifact reuses, with zero training
updates. On macOS arm64, warm execution observation median was 293.60→28.89 ms;
all 128 paired calls were faster. Current-child CPU median was 214.98→9.76 ms;
maximum single-current-child RSS median was 86,171,648→14,483,456 bytes. Prior
build history is excluded from reuse costs. These are current child observations,
separate from model tensor storage and parent/whole-host memory or utilization.
The first owned call still builds; its execution median was 299.50 ms.

The immediate Go example completed four generations, two model calls, eight
native runs and 32/32 expectations across model/deterministic routes. Repeat
responses including execution were 24.41/24.90 ms; first responses were
967.38/433.50 ms. Startup/cache/platform differences remain present. The first
collector failure exposed typed-live-scope versus decoded JSON ordering in parent
comparison; both sides now pass the same exact-number decoder before comparison.
The failure, raw observations and fixed protocol are
[public](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/retained-native-execution-20261003).
An exact-source Linux follow-up also passed: four matching generated bodies,
two predictions, eight native runs and 32/32 expectations. Repeat responses
including execution were 36.10/38.31 ms for model/deterministic routes. The
dedicated job and immutable artifact are linked in the report; this does not
change historical full-workflow arithmetic failures.

Dev #1184 and [main #1185](https://github.com/kimjooyoon/meta-ontology-go/pull/1185)
passed six canonical checks and independently verified immutable proofs, and are
merged. At that stage, clean main `8c6ec01c4931186460244f3a2975013edda62325` was locally installed.
Four generations, two predictions and eight compiled runs matched 32/32 finite
expectations and preceding generated sources. Both repeat calls reused their
executable; responses including execution were 30.72/29.52 ms. The first model
response of 1,267.64 ms is retained. Changed expectations produced a new 1/2
result and an omitted expectation was rejected before generation. Installation,
exact proof archives and all observations are linked in the report. Weights
remain unchanged.

### Direct file construction and current installation

Dev #1186 and [main #1187](https://github.com/kimjooyoon/meta-ontology-go/pull/1187)
passed their own six canonical checks and independently verified immutable proofs.
Clean main `fc854ef4` is installed with Go1.27.1. Source files, a named activity,
full typed plans or short recipes and finite expectations can now be passed
directly to one compiler command. It saves generated Go, original inputs, each
generation/runtime record and an incremental summary before the next request.

With that compiler in PATH and Go1.27.1 available, from the research repository:

```sh
gooo body-path-run --source examples/whole-candidate-order/source.gooo \
  --activity Compose --path-plan examples/whole-candidate-order/recipe.json \
  --cases examples/whole-candidate-order/cases.json \
  --model publication/order-judge-initial-20261003/model.json \
  --repeat 2 --out order-file-results
```

Use a fresh output path. `--go-bin` can select the local Go1.27.1 binary. Omit
`--model` for deterministic construction. The installed compiler ran this exact
source/recipe/case example: two generations, two actual predictions, four native
runs and 16/16 supplied expectations. The larger named EN/KO 128-input collection
separately completed eight generations/four predictions/16 runs and 1,024/1,024
expectations, with four executable reuses and unchanged generated sources.
Model-on warm response including construction/execution was 33.092ms Korean and
32.415ms English; first responses were 496.269/474.290ms. Current-child CPU across
all four warm arms was 8.829..10.957ms; maximum single-current-child RSS was
14,237,696..14,417,920 bytes. These costs exclude the earlier retained build and
do not measure model-only memory or whole-host CPU utilization. Startup/cache
conditions remain uncontrolled.

The initial sparse-example fixture retained two valid selection candidates and
missed all supplied separate runtime expectations. A source Gooo oracle added a
discriminating observation and resolved one candidate; selection/probe-disjoint
runtime inputs then passed 4/4 across two requests. Actual predictions in the
resolved oracle arm were zero, including when this model was loaded. Original
failures, unchanged weights and exact denominators remain in the
[public records](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/body-path-file-cli-20261003).
CI feedback requires a supporting model profile and explicit step/round budgets;
this whole-candidate model currently uses ordinary bounded search and rejects
feedback/batching options.

The installed completeness-delta reader still rejects owned runtime v2 records.
A [separate correction #1188](https://github.com/kimjooyoon/meta-ontology-go/pull/1188)
passed local race/vet and read 12 historical comparisons, then used this unchanged
model for two new generations/four native runs and 256/256 finite expectations.
Its canonical CI, proof and installation are pending. Changed resource units and
4→3 denominators retain both observations without a numerical improvement.
[Reader correction sources, failures and original records](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/owned-runtime-comparison-20261003).

## Research acknowledgments

[ConvAI Innovations' Laya](https://huggingface.co/convaiinnovations/laya) motivated
the structured-choice interface. [Saman Sarker Joy's decomposition study](https://arxiv.org/abs/2609.33971)
raised useful questions about composing local judgments. [DeepCoder](https://arxiv.org/abs/1611.01989)
and [DreamCoder](https://arxiv.org/abs/2006.08381) connect learned guidance with
program search and reusable structure. Gooo studies a small source-bound assembly
space with explicit finite observations and cheap local inference.

### 한국어 요약

Gooo가 만든 여덟 가지 코드 후보에서 무엇을 먼저 시도할지 고르는 16 KiB 모델입니다.
64개 한영 요청으로 짧게 학습했고, 개발 집합 96건의 첫 선택 완성은 48건에서 82건으로
늘었습니다. 같은 기능의 중복 후보를 다시 실행하지 않으면 실제 후보 평가가 144회에서
110회로 줄었습니다. 전체 탐색 시간은 조금 늘어서 준비·기록 비용을 줄이는 작업을
이어갑니다. 컴파일러 내부 연결도 완료해 추가 학습 없이 256회 생성과 512회 컴파일된
실행을 관측했습니다. 기존 개발 과제 64건에서 첫 시도 완성은 32→52건, 최대 여덟 번
시도하면 양쪽 모두 64건을 완성했습니다. 실제 생성 중앙값은 8.58→9.26ms였고,
원본 과제·가중치·실패·비용을 함께 공개했습니다.
