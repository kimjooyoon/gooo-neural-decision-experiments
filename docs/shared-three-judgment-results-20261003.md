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

# Own shared local Gooo judgment: complete bounded comparison

These are freshly initialized specialist models for choosing among eight legal
Gooo body paths. They consume complete source-bound Korean/English intent
contexts. They are not a general text-to-code model or a Transformers checkpoint.
Gooo retains source authority, legal typed alternatives, finite expectations and
deterministic continuation. No default model was replaced by this experiment.

## Actual experiment and lineage

The public research repository is
[gooo-neural-decision-experiments](https://github.com/kimjooyoon/gooo-neural-decision-experiments).
The compiler is [meta-ontology-go](https://github.com/kimjooyoon/meta-ontology-go).
Training source is `0f3249596b4dacdb34241e3b5e8b4cafac58b0df`, the independent
Go audit is `8398f1994d65b75af0483db3fefd637eec99ced4`, and the native collector
is `06f595f2b5bdb05da530f5aa698528704611a63a`.

Both models started from the same fresh Go initializer recipe; no Laya or other
pretrained/student weight was used. Each arm completed 800 FP32 and 800 QAT
updates on local MPS, for 3,200 actual updates. All six FP32/PTQ/QAT exports,
every update/epoch journal and the failed zero-update storage preflight are
retained. Dense control reproduces the earlier set-feedback model weight hashes.

The shared judge reuses a 256 -> 8 -> 2 local network over three ordered
decisions. Its trainable parameter count is 2,072 versus dense's 18,656.
Both export the existing 768 -> 24 -> 8 Go ABI: this experiment **does not reduce
resident execution tensors**. Go independently confirmed all shared matrix ties,
structural zeros, joint bit-score composition and all 288 parity predictions.
Maximum absolute Go/export error was 0.000001073, below the declared 0.00001.

The immutable source/teacher feature bank has 10,739 rows and 1,024 training,
256 calibration and 256 development program groups. Development contains 512
Korean/English views and was observed in earlier studies. These results are an
architecture comparison on that known cohort, not unseen-language generalization
or untouched holdout accuracy.

## Completeness, uncertainty and continuation

Each initial view has 16 independent finite expectations. Ranking continuation
below orders all eight masks by the one initial prediction and replays their
verified finite targets. It makes no adaptive feedback call; actual feedback
codegen is measured separately in the native phase. Budget eight eventually
finds a passing authored alternative in every cell, which follows from the
exhaustive finite search space and is not evidence of perfect model judgment.

| Export | Initial complete /512 | Initial cases /8192 | Extra ranked attempts | Complete by budget 4 /512 | EN/KO mask disagreements /256 | Same-mask both wrong /256 |
|---|---:|---:|---:|---:|---:|---:|
| Dense FP32 | 95 | 2265 | 1572 | 289 | 256 | 0 |
| Shared FP32 | 113 | 2592 | 1469 | 319 | 255 | 0 |
| Dense PTQ | 74 | 1967 | 1595 | 285 | 45 | 180 |
| Shared PTQ | 80 | 2052 | 1439 | 336 | 136 | 98 |
| Dense QAT | 82 | 2050 | 1584 | 293 | 240 | 12 |
| Shared QAT | 89 | 2191 | 1512 | 327 | 255 | 1 |

Shared FP32 raises initial complete functions from 18.55% to 22.07%, initial
case completeness from 27.65% to 31.64%, and reduces extra ranked attempts by
6.55%. Its summed development passing-set NLL **worsens** from 975.663 to
992.402, despite better calibration NLL (1.812 to 1.752). Thus confidence quality
and partial/complete behavior do not move together.

The largest FP32 gain is chained operands: 8 -> 20 initially complete views,
220 -> 149 extra ranked attempts. Comparison/assignment needs 185 -> 194 extra
attempts; nested branches 220 -> 224; successive assignments 113 -> 124. These
regressions and all per-family/per-language counts are retained in `go-audit.json`.
Bilingual disagreement remains severe. Agreement alone is not correctness:
dense PTQ agrees on many wrong masks. Shared composition is promising for bounded
continuation but has not solved Korean/English intent alignment.

## Immediate Gooo generation and real execution

All eight families, both languages, configuration 20 / goal 4 and six exports
produced **96 generations**, with 320 actual local model predictions. Every
generation was immediately followed by `gooo body-execute`; there was no
batch-wide generation barrier or model call after emission. Its two native
runs yield 192 program executions and 4,608 ordered values. All **2,304/2,304**
finite expectations pass; 768 use inputs absent from the current selection
suite. Those extra inputs are not claimed as a model-training holdout.

The actual capture compiler is `7a71f9871ae4b6cecf1f9e5e1170961a5e47ced8`.
The later source change only corrected the exact root-help test expectation.
The feature subsequently reached main in
[PR 1145](https://github.com/kimjooyoon/meta-ontology-go/pull/1145), commit
`464f387a88c535809d142be35bd54a595acece52`; captures are not relabeled to main.

All 96 shared Gooo-declared receipts were separately decoded using the real
compiler decoder. Exact original parent bytes and unrelated dimensions are
unchanged. Runtime model/provider calls are zero. `permission_boundary` remains
the first unresolved dimension in every result; native execution is not an OS
sandbox or proof of arbitrary host permissions.

## Resource observations

The four training loops total 33.84 seconds, excluding input loading, export and
audits. Loop CPU was 72.4–85.5% of one core; process lifetime peak RSS was
541.77 MiB. Sampled MPS tensor allocation peaked near 30.6 MiB and the MPS driver
pool near 1,040.7 MiB. These overlapping measurements are not summed; GPU busy
percentage and causal whole-host CPU increase were not measured.

Warm Go kernels measured 35.7–41.8 microseconds per prediction, including feature
projection, with zero heap allocations in valid warmed `PredictInto` calls.
FP32 weights/resident tensors are 74,624 bytes; ternary packed files are 3,854
bytes, resident tensors 18,752 bytes plus 8 scale bytes. Caller workspace is
3,200 bytes. Five trits per byte means **1.6 stored matrix bits**, not total
1.58-bit RAM.

Shared FP32 native codegen median was 30.92 ms (dense 33.93 ms), including
process start. Codegen CPU medians were 85.67% / 84.76% of one core. Shared
FP32 runtime pipeline median was 617.49 ms, including compilation and two
process runs; compiled child median RSS was 4.24 MiB, which is not model RSS.
Fixed interleaved order, 16 requests per variant, cache state and concurrent
local proof verification limit performance comparisons. Dense's cold-build p95
was 7.57 seconds. These measurements do not establish a causal speedup.

## Reuse and remaining work

Model files are under `models/{dense,shared-local}/{fp32,ptq_ternary,qat_ternary}`.
Use the pinned Go SDK's `jointdecision.LoadThree` / caller-owned arrays, or pass
the selected `model.json` to Gooo's `body-codegen --path-model` together with a
typed path plan. Full public fixture inputs, plans and runtime expectations are
in `evidence.zip`. No network/model service is needed during local inference.

The frozen preparation and source/teacher data remain in
[the original own-model publication](https://huggingface.co/asketeddy/gooo-three-choice-feedback-tiny-v1).
The archive contains the unchanged protocol, pre-optimizer storage amendment,
all six models, journals, Go/native evidence and source readers. A byte manifest
binds every member. The 768-MiB failure is preserved; new evidence has a 64-MiB
allowance within the separately declared 2-GiB retained-study limit.

Next experiments should examine the source/intent representation behind the
bilingual split, retain per-family regression metrics, and evaluate an actual
compact tied-weight Go ABI before claiming lower execution memory. No model
promotion follows from this bounded experiment alone.
