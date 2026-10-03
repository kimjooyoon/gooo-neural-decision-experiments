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
- Experimental research runtime and actual typed-interpreter search are available.
  **In-compiler native invocation of this new model is the next integration stage.**

The runtime is in the
[research repository at the publication revision](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/8a41bb0f825dfd3f950101b33491581492c740f6/internal/orderjudge).
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
candidate evaluation. Native compiler runs for this new architecture are zero.

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

The current development focus is reusing prepared candidates and connecting this
judge directly to Gooo's source-bind → rank → test → generate pipeline. Existing
V3/V4 models and their historical numerical comparisons remain preserved.

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
이어갑니다. 원본 과제·가중치·실패·비용을 공개했고, 새 모델의 컴파일러 내부 연결은
다음 단계입니다.
