# Full three-choice SDK comparison after a separate storage continuation

## What actually ran

The original producer `b8c515fcd5a5e623aae5fd3b93c3b8f69f9f4608` remains a
failed 768 MiB attempt with 8,779 immutable recorded sessions. The published
resource amendment permits 2 GiB and preserves that outcome. Clean public Go
source `59f921c520d369d58800ea5c9505f4ff60f1b307` independently audited the
original prefix, recorded the exact missing identity list before inference,
and executed only **2,485 missing development sessions**, with **9,142 actual
model predictions**. No original operational session was repeated.

The combined independent audit now proves **11,264 unique sessions**, **42,530
actual recorded model predictions**, **658,992 ordered candidate values**, all
11 calibration cells and all 11 development cells of 512 views. It rechecks
full inputs, source/model identities, every ordered arithmetic result, actual
failure contexts, observed probabilities, committed frontier and progress
chains. The separate audit and bilingual diagnosis make **zero model calls**.
Their status is `PASS_WITH_SEPARATE_STORAGE_AMENDMENT`; this is not a successful
run under the original cap. No new optimizer update or default promotion ran.

## Development results — identical finite contracts

Calibration selected `set-feedback/fp32` before any original development result.
That choice is preserved. Each row has 512 actual development functions, 16
ordered contract cases per function and at most eight authored mask candidates.

| Policy | Extra candidates | Actual model calls | Complete at 1 | Complete at 2 | Complete at 4 | Complete at 6 | Complete at 8 | Median / p95 SDK wall, ms |
|---|---:|---:|---:|---:|---:|---:|---:|---|
| Offline | 1,546 | 0 | 90 | 160 | 304 | 418 | 512 | 0.142 / 0.353 |
| Frozen independent reference | 1,156 | 4,706 | 116 | 250 | 379 | 456 | 512 | 0.217 / 0.580 |
| Set-feedback FP32, selected | 1,145 | 1,638 | 95 | 233 | 406 | 459 | 512 | 0.219 / 0.549 |
| Set-feedback PTQ ternary | 1,552 | 2,028 | 74 | 152 | 298 | 422 | 512 | 0.301 / 0.619 |
| Set-feedback QAT ternary | 1,342 | 1,827 | 82 | 198 | 354 | 444 | 512 | 0.245 / 0.609 |
| Set-initial FP32 | 1,322 | 1,801 | 89 | 201 | 351 | 447 | 512 | 0.243 / 0.594 |
| Set-initial PTQ ternary | 1,586 | 2,063 | 75 | 146 | 284 | 423 | 512 | 0.323 / 0.632 |
| Set-initial QAT ternary | 1,311 | 1,795 | 93 | 194 | 353 | 450 | 512 | 0.250 / 0.597 |
| Uniform-initial FP32 | 1,266 | 1,745 | 106 | 213 | 359 | 440 | 512 | 0.228 / 0.599 |
| Uniform-initial PTQ ternary | 1,550 | 2,034 | 80 | 153 | 281 | 440 | 512 | 0.324 / 0.597 |
| Uniform-initial QAT ternary | 1,347 | 1,821 | 85 | 193 | 357 | 437 | 512 | 0.242 / 0.619 |

The selected student uses **65.19% fewer model calls** than the independent
reference, but only **0.95% fewer extra candidates**. Its median is slightly
higher. Against uniform-initial FP32 it uses **9.56% fewer extra candidates**
and **6.13% fewer calls**. First-candidate completion is **18.55%**, below the
reference's **22.66%** and uniform student's **20.70%**. At four candidates,
selected completion is **79.30%**, versus reference **74.02%**, uniform
**70.12%** and offline **59.38%**. Test-driven continuation helps this finite
cohort without establishing first-attempt natural-language understanding.

All policies, including offline, finish at eight candidates because the known
space contains eight authored masks and has a passing candidate. This finite
completeness is not evidence that a model can generate arbitrary missing code,
handle unbounded IR, or prove general natural-language intent. The SDK timing
includes typed assembly, local contract evaluation and feedback. It excludes
native Gooo compilation and compiled Go execution. Original and tail timings
come from separate phases; these are observational costs, not randomized speed
claims or host-wide CPU/GPU utilization measurements.

PTQ and QAT preserve the nominal five-trit packed format, but add work here.
The selected arm's PTQ adds 35.55% and QAT 17.21% more extra candidates than its
FP32 variant. Packed matrices do not establish better end-to-end latency or
total resident memory. All negative variants remain public.

## Bilingual output diagnosis, separate from selection

A post-hoc zero-prediction diagnosis compares actual initial outputs on identical
Korean/English contracts: **5,632 pairs and 90,112 ordered case pairs** across
both splits and all policies. Different mask IDs can produce the same finite
outputs, so mask disagreement alone was not treated as functional failure.

For selected FP32 calibration, 256/256 mask pairs differ, but **21 pairs have
identical ordered outputs**; actual output disagreement is **235/256**. In
development, both mask and ordered-output disagreements are **256/256**, with
**3,879/4,096 unequal ordered case values**. Neither language completes its
first candidate in 161 pairs, exactly one does in 95 pairs, and both do in zero.
The frozen independent reference has 240/256 output-disagreement pairs; uniform
FP32 has 246/256; selected-arm QAT has 240/256. Offline has zero disagreements
and both first candidates complete in 45 pairs, reflecting its fixed choice.

This is direct evidence of weak bilingual initial judgment on this cohort.
The next model study should measure paired functional consistency and passing-set
probability in addition to continuation cost, preserve semantic ties, and keep
development data outside selection/training. No new training or selector change
was performed after seeing these results. Equal finite outputs are not universal
program equivalence, and this templated cohort does not measure paraphrases.

## Actual resource boundary

Tail collection wall: **11.662 s**; measured process CPU: **16.042 CPU-s**, or
**137.55% of one core** averaged over that phase. Go runtime and audit work can
use multiple cores. This cost includes original-prefix replay, planning and tail
recording; it is not pure model inference or a causal system-wide utilization
increase. Process lifetime peak RSS: **312,197,120 B**, about **297.7 MiB**,
separate from the tiny model's decoded weights and fixed inference workspace.
GPU utilization was not measured; this phase performs CPU Go inference.

Whole retained uncompressed evidence is **946,884,298 B**, about **903.0 MiB**,
including all earlier source/teacher/training phases, the failed SDK prefix and
the new tail. Original raw files remain local. Closed lossless public bundles
avoid duplicating original records or uploading the old weights again. Original
failed collector CPU/RSS remain unavailable. Source generation and full **640
native Gooo / 640 compiled Go** validation are still required and have not been
run in this new three-choice phase.

Evidence: `publication/own-three-choice-sdk-combined-audit-20261002.json`,
`publication/own-three-choice-sdk-bilingual-functional-audit-20261002.json`, and
the complete tail ZIP/closed byte manifest. The original stopped prefix is
preserved in its separate public ZIP and Hugging Face appendix.
