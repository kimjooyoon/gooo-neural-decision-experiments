# A small judge that sees the whole candidate

Measured 2026-10-03 on Apple M4, 10 CPU cores, 16 GiB RAM, macOS arm64,
Go 1.27.1. The model matches complete Korean/English instructions against the
actual ordered operations in each permitted Gooo body. It has **4,096 parameters,
16 KiB FP32 weights**, and a 128-byte prediction workspace. Input arrays, prepared
programs, artifact loading and result documents require additional memory.

The admitted body is `let v=input; two integer updates; return v`. Four arithmetic
families, two source orders, two requested orders and two languages form the
authored cohort. This model is independently initialized; it uses no Laya weights
or teacher outputs. Laya inspired the structured-choice interface. Whole-candidate
scoring also follows the question raised by
[Joy's decomposition study](https://arxiv.org/abs/2609.33971): separately useful
local probabilities need not compose into reliable whole-program confidence.

## Fixed fit and first observations

The [protocol](../../docs/order-judge-protocol-20261003.md) precedes the first fit.
64 training requests, 400 full-batch updates, fixed learning rate 2 and L2 0.0001.
The initial collector is `5f0caf53f18747d44f79734a2768c4dbe238a947`; source-bound
plan exports come from clean compiler `ba00850f57db382f4b6a040e7a9ed14f0f6a98b4`.
Training took **89.61 ms on CPU**; GPU updates were zero.

Each request uses three selection inputs and eight evaluation inputs, including
five selection-disjoint inputs. Completeness here means all eight finite outputs
match the authored reference. The 96 development requests use new templates,
presentation changes or new constants; families are already represented in fit.

| Group | Requests | First complete: deterministic / model | Budget-8 body attempts: deterministic / model | Budget-8 complete, both |
| --- | ---: | ---: | ---: | ---: |
| Training | 64 | 32 / 62 | 96 / 68 | 64 |
| New template | 32 | 16 / 26 | 48 / 48 | 32 |
| Renamed local + commuted operands | 32 | 16 / 30 | 48 / 36 | 32 |
| New constants | 32 | 16 / 26 | 48 / 52 | 32 |

Across the three development contrasts, first completion is **48/96 versus
82/96**. All-mask attempts fall only from **144 to 136**, and the new-constant
group gets worse. Scores are uncalibrated ranking weights. They do not express
the probability that arbitrary natural-language intent is fully implemented.

The initial fit makes 320 post-fit predictions, including exact artifact-reload
comparisons. Timed prediction-only calls have median **1.875 µs**, range
1.791–2.333 µs. Features and artifact loading are outside that timer. The whole
collection/fit/evaluation process takes 1.94 s wall time and 1.60 s CPU time,
roughly **82.5% of one core on average**, with 43.39 MiB maximum RSS reported by
`time -l`. These whole-process numbers include compiler child invocations and
are neither inference-only memory nor peak system CPU utilization.

The initial learned search counts replay precomputed candidate outputs. A real
bounded-search follow-up below checks them by freshly executing typed programs.

## Actual search and equal-body reuse

The [follow-up protocol](../../docs/order-judge-runtime-followup-20261003.md) keeps
the exact weights and runs at source `76094eddbe0dcb4bf8470d9f5de5e054c87e9844`.
**960 actual searches, 480 actual model predictions**, plus 320 existing-SDK
baseline searches, reconstruct the original outcomes and full rankings exactly.
There are zero training updates. Source descriptors, features, all 1,280 candidate
bodies and their eight outputs per body are reconstructed from the exported plans.

Within this profile, equal 48-byte descriptors identify the same pure integer
computation. Skip already evaluated equivalent candidates, while retaining each
structural mask and the representative used. Compare both ranking policies:

| Development group | Deterministic, ordinary / reuse | Model, ordinary / reuse |
| --- | ---: | ---: |
| New template | 48 / 48 | 48 / 38 |
| Presentation | 48 / 48 | 36 / 34 |
| New constants | 48 / 48 | 52 / 38 |
| **Total actual bodies evaluated** | **144 / 144** | **136 / 110** |

All four budget-8 arms complete **96/96 requests and 768/768 finite evaluation
cases**. The 110 versus 144 comparison is **23.6% fewer body evaluations** on
this already observed cohort. It combines learned ranking with exact reuse.
The reuse experiment was designed after seeing the original result and is a
development follow-up, not new holdout evidence.

Fewer evaluations have not yet made total search faster. Across all 160 requests,
median budget-8 wrapper search is 322.04 µs deterministic, 340.83 µs learned and
339.25 µs learned with reuse. Preparation validates all eight candidates and
artifact hashing adds work. This timer excludes final Gooo/Go generation and
native execution. The complete replay process takes 0.91 s wall / 0.59 s CPU
and reports 47.66 MiB maximum RSS. Detailed timers remain in `runtime.zip`.

## Use and reproduce

These artifacts currently run in the research Go package `internal/orderjudge`.
The compiler's source-plan export is the integration point. Native invocation
of this new model inside the compiler is the next stage; native runs in this
publication are **zero**. The released shared-judge model has its separate native
integration and evidence.

From the research repository with Go 1.27.1:

```sh
cd publication/order-judge-initial-20261003
shasum -a 256 -c SHA256SUMS
unzip -q initial.zip -d /tmp/gooo-order-initial
cd ../..
go run ./cmd/order-judge-replay \
  --input /tmp/gooo-order-initial --output /tmp/gooo-order-replay
```

Both output directories must be fresh. The replay loads the original model,
reconstructs every input and prediction, compares real search to frozen outcomes,
and writes all attempts, equivalence skips and functional results. `Search` accepts
a nil model for deterministic operation. It requires a deadline and bounded cases.

## Artifact inventory and limits

- `initial.zip`: all 160 authored sources/recipes/source-bound exports, complete
  dataset and features, 400 loss records, model, predictions, manifest and summary.
- `runtime.zip`: every actual search, prediction, finite case, alias and timer,
  plus clean build identity and group counts.
- `model.json`, `weights.bin`: directly loadable model, exact same bytes as the fit.
- `*-summary.json`, `*-process.txt`: convenient group totals and original resource logs.
- `SHA256SUMS`: digest of every published artifact in this directory, including this page.

The model accepts full valid UTF-8 intent up to 512 bytes and constants -16..16
in its numeric feature projection. Three binary structural decisions produce
eight masks; only one root order and two operand orders are admitted. Long text,
branching, multiple locals, nested operations and broad natural-language coverage
require new scoped work. No 1.58-bit export was measured for this architecture.
The original fit, including regressions, remains unchanged.
