# Own Gooo semantic composition model: actual training and dogfood

This study trains six own random-init models under the
[frozen protocol](fresh-composition-semantic-v3-preregistration-20261002.md).
There are no inherited Laya, pretrained or earlier own-model weights. Runtime,
collection, oracle, audits and publication are Go; Python performs offline MPS
optimization/export. The optional model ranks compiler-owned typed fragments.
Disconnected generation follows the deterministic bounded continuation path.

## Training and inputs

Both feature arms share an initial random state (seed 20261022), paired batch
shuffle seed 20261023, 3,072 train decision views, 768 calibration views and 768
development views. Six families combine two interacting choices each. The
9,216-row source-bound collection spans Korean/English, configurations and four
goals; these variants are not independent intentions. Complete 16-case targets,
int64 wraparound, duplicate inputs and finite equivalence ties remain intact.
All byte-identical conflicting-input targets remain in the data.

FP32 and QAT each perform 240 updates per arm: 960 total. PTQ has zero optimizer
updates. The four MPS loops sum to 1.716 seconds, excluding feature preparation,
startup/export and Go evaluation. Sampled tensor allocation peaked at 4,208,896
bytes; driver allocation at 53,166,080 bytes; process lifetime RSS at 548,208,640
bytes. GPU utilization and a host CPU increase were not measured.

Each model has 12,728 parameters, 256 inputs, 48 hidden values and eight labels.
Go/Python parity on 192 held-out exported-model observations has maximum absolute
error 2.683e-7. CI also checks published weights and parity without GPU training.

## Finite completeness and continuation cost

Every cell uses all 384 bilingual development function views and all 16 ordered
cases. Feedback consumes only actually committed failures and a caller hint
bound to successful compiler main CI; the hint grants no semantic authority.
The calibration selector is frozen before development session predictions.

| Model | Initially complete / 384 | Extra attempts | Actual predictions | Complete at budgets 1 / 2 / 3 / 4 |
|---|---:|---:|---:|---|
| Disconnected | 104 | 548 | 0 | 104 / 202 / 298 / 384 |
| v2 FP32 | 110 | 524 | 1,487 | 110 / 213 / 305 / 384 |
| v2 PTQ | 121 | 504 | 1,464 | 121 / 214 / 313 / 384 |
| v2 QAT | 125 | 491 | 1,450 | 125 / 220 / 316 / 384 |
| Source v3 FP32 | 163 | 397 | 1,360 | 163 / 259 / 333 / 384 |
| Source v3 PTQ | 127 | 465 | 1,440 | 127 / 231 / 329 / 384 |
| Source v3 QAT | 115 | 470 | 1,479 | 115 / 241 / 326 / 384 |

Calibration selected **v3 QAT**, with 365 extra attempts versus v3 FP32's 377.
On development v3 FP32 is better (397 versus 470); this negative selection
comparison remains recorded. Development cannot replace the frozen candidate.
Relative to v2 FP32, v3 FP32 reduces extra attempts by 24.2%; selected v3 QAT by
10.3%. These are descriptive finite-cohort comparisons with one matched seed.

All arms finish their finite contract by four candidates. The model changes
priority and cost; complete enumeration is not evidence of general language
understanding. Initial bilingual disagreement remains high, including 138/192
calibration pairs and 166/192 development pairs for selected v3 QAT. Agreement
and functional completeness have separate denominators.

Across calibration/development, parity and warm/allocation probes, the Go study
makes 29,106 actual predictions. Of those, 12,012 are explicit performance probes
and 192 are parity checks. The independent audit rechecks 5,376 function records
with zero new inference or optimizer updates.

## Direct Gooo compiler dogfood

Clean native main `f4813dc6251037767c8cff7295ccfdab2b044ff2`, Go 1.27.1 and
SDK v0.2.11 run the fixed 48-view config-40 subset in three policies: selected
v3 QAT, new v2 FP32 reference and disconnected. These are **144 actual native
calls, 370 model predictions, 144 separately compiled Go executions and 2,304
actual function invocations**. All complete 16-case contracts pass. Native
candidate observations match the earlier SDK session observations.

| Native policy | Initially complete / 48 | Extra attempts | Predictions | Native child median wall | Median max RSS |
|---|---:|---:|---:|---:|---:|
| Disconnected | 12 | 72 | 0 | about 10.0 ms | about 17.8 MB |
| v2 FP32 reference | 13 | 68 | 187 | about 10.7 ms | about 18.3 MB |
| Selected v3 QAT | 14 | 60 | 183 | about 10.8 ms | about 18.0 MB |

The selected native child CPU median is about 84% of **one core** over its wall
time. This is not host utilization or its increase. Fixed policy order, cold
process startup and independent Go builds do not establish a causal wall-speed
gain. Selected warm inference measured 9.675 microseconds with zero per-call
heap allocations; whole-process time remains separate.

Ternary weights occupy 2,759 disk bytes, decode to 12,896 resident tensor bytes
plus eight scale bytes, and use a 1,248-byte workspace. Five-trit packing is 1.6
storage bits per matrix weight; arithmetic uses decoded int8 matrices and FP32
biases/scales. This is not whole-process 1.58-bit RAM.

The first native aggregator did not populate bilingual pair fields and emitted
zeros there. Raw captures are preserved. The independent audit reconstructs 24
language pairs per policy, appends corrected statistics and validates every
source/input/model/hash chain and captured Go value. Initial language masks
disagree in 18/24 selected pairs versus 14/24 reference pairs. Future aggregation
is fixed.
Cancellation kills child process groups; regression tests cover held pipes and
atomic output overflow. No deadlock was observed in this bounded run.

## Use and further growth

Use the public six models explicitly with the current Gooo compiler's
`--path-model`; its original source binding creates the proper model input.
The experiment remains optional and does not promote a default model.

Next studies should freeze new material before outcomes: richer source features
that distinguish the retained conflicting inputs; joint two-choice ranking that
preserves correlated finite masks; Korean/English agreement measured against
functional contracts; and QAT selection measured across several seeds. Each
should compare complete-function cost, actual predictions and runtime memory.
Existing test observations must not silently become untouched evaluation data.

[Public weights and trainer](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/models/fresh-composition-v1)
and [Go measurement source](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/tools/fresh-composition-study)
are available. The
[standalone Hugging Face edition](https://huggingface.co/asketeddy/gooo-semantic-composition-tiny-v1/tree/1102d2c1c157cd72054cc9fd16f81a61043cc660)
contains all six variants, their negative comparisons, six PROV-O derivations
and 327 raw/source archive members. Anonymous immutable verification passes
for all 35 payloads plus the manifest. The archive compresses 132,429,624 bytes
to 9,682,825 bytes. [Publication-source CI](https://github.com/kimjooyoon/gooo-neural-decision-experiments/actions/runs/36924760889)
passes all eight jobs; the [finalization receipt](../publication/semantic-composition-iteration-finalization-20261002.json)
and [growth roadmap](own-small-model-growth-roadmap-20261002.md) retain exact
pins and the scope of further studies.
