# Independent small Gooo model: paired judgment iteration

Own random initialization, 12728 parameters, five compiler-owned typed path
families and Korean/English intent. These are bounded path judgments, not a
general text or code generator. No Laya weights, fine-tuning, upstream inference
or inherited weights were used. Runtime, native orchestration, audits and
publication are Go; Python only performs offline MPS training/export.

## Protocol and evidence

The [preregistered protocol](https://github.com/kimjooyoon/gooo-neural-decision-experiments/blob/b0dd6fe96d76deceef7ef937f85f6f514b765ca6/docs/paired-compiler-context-preregistration-20261002.md)
pins the unchanged 2080 compiler-exported input views: 800 training, 80
calibration and 160 development bilingual pairs. The already inspected
development cohort is reused, with **zero new independent intentions**. No
untouched benchmark or universal correctness claim.

Consistency weights 0, 0.1 and 0.3 share initialization, minibatches and fixed
budgets. Each arm exports FP32, post-training ternary and quantization-aware
ternary variants. 840 optimizer updates used local MPS. The six optimization
loops totaled 1.6935 seconds; process-lifetime RSS reached 474726400 bytes,
sampled MPS tensor allocation 2033920 bytes and driver allocation 53166080
bytes. These are sampled allocations and optimizer-loop durations, not GPU
utilization or complete startup/export time.

All three zero-weight exports reproduce the previous compiler-context models
exactly, including metadata and weight hashes. Go/Python parity used 288 actual
predictions; maximum absolute difference was 2.2724271e-7, below 1e-6.

## Reused development results

Each row covers 320 views and 3840 independently reconstructed full-contract
arithmetic cases. Model ranks a declared path before any candidate tests.
Compiler-owned bounded TDD can assemble one further candidate.

| Consistency | Representation | Initially complete views | Extra candidates | EN/KO disagreements / 160 pairs |
|---|---|---:|---:|---:|
| 0 | FP32 | 240 | 80 | 80 |
| 0.1 | FP32 | 239 | 81 | 81 |
| 0.3 | FP32 | 239 | 81 | 81 |
| 0 | PTQ ternary | 177 | 143 | 15 |
| 0.1 | PTQ ternary | 185 | 135 | 7 |
| 0.3 | PTQ ternary | 185 | 135 | 7 |
| 0 | QAT ternary | 219 | 101 | 93 |
| 0.1 | QAT ternary | 221 | 99 | 99 |
| 0.3 | QAT ternary | 213 | 107 | 95 |

All nine finish **320/320 views and 3840/3840 finite cases** after at most one
extra candidate. This is finite-contract completeness, not all-input semantic
correctness. All nine were measured on 160 calibration views before 320
development views; total Go calls 1440 + 2880 + 288 parity = **4608**.

The PTQ improvement is eight fewer extra candidates (143 to 135, 5.6%), but
low language disagreement can reflect shared mistakes. Its reference, operand,
branch and execution-order judgments each often remain 16/32 per language.
FP32 did not improve; QAT has mixed changes. Preserve these negative results.

Calibration alone selected **js0-qat_ternary**, frozen before development:
54 extras and 36 disagreements beat tied-extra FP32's 38 disagreements.
On development it needs 101 extras versus FP32's 80. The selection rule does
not reliably generalize here. The selected artifact remains an explicit
experiment candidate; no compiler default was promoted.

## Actual deployed compiler dogfood

Pinned native main `b431ef7547a968e2532fa2dcf0891ee44ec2ee86`, Go 1.27.1,
SDK `v0.2.9-experimental`. Forty deterministic development views compare the
selected candidate, the previous compiler-context FP32 and disconnected
fallback: **120 native calls, 80 actual local predictions, 15 compiled Go
executions, 180 function invocations**. All source/context/model hashes,
candidate actuals and compiled function outputs pass independent audits.

Go prediction medians across the nine development arms are 9.46–11.33 µs.
Across all native children, median wall time is 6.4506 ms, max RSS spans
16711680–17727488 bytes (15.94–16.91 MiB), median CPU is 86.35% of **one core**.
This does not measure whole-host CPU increase or prove a causal speedup.
Offline makes zero model calls and completes the same finite contracts.

FP32 weights occupy 50912 bytes. Ternary weights occupy 2759 bytes; five trits
per byte mean 1.6 disk bits per matrix weight, close to the theoretical
log2(3). Go decodes int8 matrices: 12896 resident tensor bytes plus scales,
1248 bytes reusable workspace. There is no packed 1.58-bit arithmetic or
whole-process RAM claim.

## Failure retained and recovery

CI run 36891320795 rejected the first native observation because the new
consumer compared indented file bytes with the compiler's compact canonical
document digest. Native decision was PASS, one prediction was observed, and
zero compiled Go executions had occurred. The raw receipt and preexecution
are retained under `runs/paired-compiler-ci-rejected-20261002`; the incomplete
run is excluded as a completed native study. The consumer now reconstructs
the canonical digest, with no change to weights, model inputs or compiler.
Fresh 120-call native evidence is stored separately.

## How to use and how to grow

Use the model explicitly on source-bound declared paths:

```sh
gooo body-codegen --json --path-plan plan.json --activity ChoosePath source.gooo \
  --path-model models/js0/qat_ternary/model.json
```

The model path is optional. Compiler type checks, source equivalence and finite
case actuals own the outcome. A model confidence score is never a human review
gate or source authority. This model cannot fill arbitrary bodies by itself;
the compiler assembles detailed bodies from its legal IR alternatives.

Next study should represent definition/use links, branch predicate/arm meaning
and ordering dependencies as bounded compiler features rather than relying
mostly on byte n-grams. Keep a small shared intent encoder and per-decision
eligible-option heads. Measure complete assembly versus extra candidate cost,
valid partial composition and language disagreement separately. Preserve
source-bound PROV-O entities/activities for source, model, judgment, candidate
and test receipts; provenance explains observations, it does not grant model
write authority. Before changing the feature ABI, preregister paired controls,
new independent compositions and a fresh bilingual evaluation split.

Machine-readable scores, all nine own models, source snapshots, raw native
captures and the rejected run are published in GitHub and the Hugging Face
`research/paired-compiler-20261002` appendix. Existing model evidence remains
available at its immutable revision.
