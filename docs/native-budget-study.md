# Bounded bilingual construction on native main

The [preregistered study](native-budget-preregistration.md) executed **1,944 native
constructions** on clean main `37fb287a9c888bd0262194f524dde16426eca3a9` with Go
1.27.1 and SDK 0.2.7. It reused 72 contract/language views over twelve existing
intention groups and three templates. Nine arms and three budgets are repeated
observations, not independent new tasks or a new language holdout. No model was
trained, promoted or derived from Laya weights in this phase.
The four checkpoints retain our independently initialized 12,728-parameter
weights; their earlier broader evaluation results remain unchanged.

There were 162 worker processes, each with one invalid-source rejection followed
by twelve sequential valid requests. All 162 rejections made zero predictions.
The valid constructions committed 3,005 candidates and made **4,041 actual model
predictions: 3,456 initial and 585 feedback**. Every captured native result was
source/document/case/model bound, type/replay verified and matched to a selected
typed function AST. Twelve distinct actual emitted Go programs were independently
compiled and executed over sixteen finite/separate inputs, including int64
boundaries: **192 actual function invocations**, reused and reconciled for all
1,944 observations. Offline replay produced identical report bytes.

## Two-candidate budget

Each row below has 72 observations, 384 finite cases and 548 separate-input case
views. Separate-input views reuse the actual emitted-Go executions; they are not
548 independent function invocations or natural-language samples.

| Arm | Finite cases | Separate inputs | Intended masks | Mean budget-position completion | Actual predictions |
|---|---:|---:|---:|---:|---:|
| Disconnected | 58.33% | 60.58% | 36 / 72 | 45.96% | 0 |
| Parent FP32, initial ranking | 93.75% | 96.72% | 70 / 72 | 93.23% | 144 |
| Parent FP32, feedback | 91.93% | 95.44% | 69 / 72 | 92.53% | 196 |
| Existing new FP32, initial ranking | 93.75% | 100% | 72 / 72 | 87.85% | 144 |
| Existing new FP32, feedback | 93.75% | 100% | 72 / 72 | 87.85% | 208 |
| PTQ, initial or feedback | 93.75% | 96.72% | 67 / 72 | 87.24% | 144 / 208 |
| QAT, initial ranking | 89.84% | 95.80% | 69 / 72 | 85.85% | 144 |
| QAT, feedback | 93.75% | 100% | 72 / 72 | 87.85% | 208 |

**93.75% is the finite legal maximum for this contract mixture.** Complete and
sparse views pass all declared cases at that maximum. Each contradictory view
has seven passes out of eight. It is a different denominator from the earlier
contradictory-only 87.5% retained pilot, not an increase in its score. The maximum
is independently enumerated after construction over four authorized typed paths;
no maximum, intention mask or separate input was supplied to selection. Reaching
it is not all-input correctness or general language understanding.

The parent reaches higher average early progress, while the existing new FP32
has stronger final intention/separate-input observations here. These criteria do
not establish a universally best checkpoint. Earlier broader held-out regressions
remain published and prevent an automatic default-model promotion.

## Feedback and expression differences

Among 864 matched feedback/initial pairs, 22 changed candidate sequence. Three
improved finite and separate-input outcomes; they are the **same Korean
`operand_branch`, mask-three intention**, repeated across complete, sparse and
contradictory contracts with QAT at budget two. This is one intention group,
not three independent repairs. One parent-FP32 English complete-contract pair
became worse at budget two despite two additional predictions and the same two
candidate attempts. It lost seven finite and seven separate-input passes.

Feedback adds 64 predictions to new FP32 at budget two without changing its
outcomes in this cohort. The same context can help one frozen model and hurt
another. An always-on feedback default is therefore unsupported by this evidence.
The next bounded policy comparison should account for marginal construction gain
and prediction cost, preserve negative cases and use a new group-separated
curriculum if any training is performed.

There are 972 matched Korean/English arm/budget/contract pairs. Eighty emit
different selected bodies: 48 at budget one, 18 at two and 14 at four. At four,
all arms reach the same finite maximum, yet sparse contracts can still leave
language/intention differences visible. This is compatible with accepting known
expression limits while measuring them. It is not a new estimate of multilingual
generalization. First-shot correctness is an observation, not acceptance.

## Timing and memory

New FP32 initial-only, budget two, has request median **0.916 ms**, median native
construction time **0.519 ms**, whole-child CPU **1.433 ms per
valid construction**, median child peak RSS **20.39 MiB** and maximum **20.91 MiB**.
Across modeled arms/budgets, request medians are approximately **0.879–1.031 ms**;
whole-child CPU is **1.399–1.533 ms per valid construction**, median peak RSS is
**20.08–20.81 MiB**, and maximum peak RSS is **21.17 MiB**. Average child CPU is
about **102–106% of one core**. Host-wide utilization increase is unmeasured.

Request round-trip starts after the constructor record and excludes startup and
one-time model loading. Whole-child measurements include startup, constructor,
one source rejection and twelve valid requests. Controller work and generated-Go
compilation/execution are excluded from child resource figures. The first
disconnected worker's constructor-arrival time is **421.064 ms**, retained as an
outlier with uninstrumented cause. No causal wall-speedup is inferred from fixed
arm order. RSS describes thirteen small records per worker, not arbitrary
maximum-size or long-stream memory. Quantized storage remains 1.6 bits per weight
on disk with decoded int8 computation; model-array sizes do not equal process RSS.

## Evidence and selection rule

`runs/native-budget-main-20261001` contains all raw NDJSON, resource sidecars,
preexecution pins, independently executed Go values and the reproducible report.
Original UNKNOWN CI context remains unauthenticated caller input and is not
retroactively rewritten. Exact-tree main adoption proof and actual main runtime
measurements are separate from any subsequent push-CI observation.

The source's full-budget audit still requires the independently authored finite
maximum. The new partial-budget audit verifies actual best-so-far values instead.
Same-arm candidate and committed-curve prefixes match across budgets, and
initial/feedback bodies match at budget one. Progress curves exclude initial and
feedback-only checkpoints. Budget-position means carry the last score forward
after early completion; actual attempt counts remain separate. This padding
creates neither extra executions nor a time-weighted area.

For this existing cohort, one local initial-ranking hint followed by bounded
deterministic tests is a useful small-cost construction mode. Feedback remains
an explicit alternative with measured benefits and failures. No global policy or
checkpoint is changed from these reused views alone.
