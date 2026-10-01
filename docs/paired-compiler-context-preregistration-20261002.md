# Paired compiler-context judgment study v1

Preregistered before optimization/evaluation. Use the existing native canonical
compiler context v2, unchanged feature ABI `split_context_intent_ngrams_v2`,
256 features / 48 hidden units / 8 labels / 12728 parameters. Runtime and study
orchestration are Go. Python is offline local MPS optimization/export only.

## Data and authority

Pin validated 2080-row curriculum SHA
`8c2b4fb884e001f7a58fa84ce862f61abe0f99b26855064d0011f631d67cecb3`.
Reuse 1040 bilingual pairs (800 train, 80 calibration, 160 development), 640
program groups, five typed path families, original finite soft-target ties.
All initial inputs remain exact compiler-exported source facts + natural suffix;
no expected answers, test outcomes or CI outcomes are added to initial input.
The development cohort has already been inspected. This is a reused diagnostic
cohort, zero new independent intentions and no untouched benchmark claim.

## Matched training

Three arms: `js0`, `js01`, `js03` (consistency weights 0, 0.1, 0.3). Same random
initial state and minibatch order, seeds 20261012 for FP32 and 20261013 for QAT,
20 epochs per optimized representation. Each arm: FP32 140 updates, QAT from
its own FP32 checkpoint 140 updates, PTQ export without optimization. Maximum
840 updates total. Compare zero weight with the prior compiler-context control;
record exact exported-weight agreement or any nondeterminism, never assume it.

Loss is finite soft-target NLL plus the weighted Jensen-Shannon divergence of
English/Korean accepted-option distributions. Ties retain probability mass on
both valid labels; agreement is not rewarded by inventing a unique intention.
Checkpoint choice uses the minimum calibration objective. Export temperature
0.5/1/2/4 uses calibration finite NLL only, as before. No development score
selects checkpoints or a runtime candidate.

## Go evidence and candidate selection

For every nine exported models: 160 calibration views, 320 development views,
32 Go/Python parity calls (96 per arm). Total planned Go predictions: calibration
1440 + development 2880 + parity 288 = 4608. Reconstruct finite candidates and
full independent arithmetic cases. Report initial intention agreement, accepted
set coverage, full finite completeness after max one additional candidate,
additional candidate cost, language disagreement, representation and latency.

Choose one candidate using calibration **only**: minimum added full-contract
candidates, then minimum calibration bilingual disagreement, then finite NLL,
then lexicographic arm/variant ID. Completeness and source/ABI/parity failures
exclude a candidate. Preserve all nine variants and negative comparisons.
No automatic default model promotion: a selected candidate is an explicit
experiment artifact. A constant bilingual prediction may improve agreement yet
harm intended path coverage; report both to identify this failure mode.

## Native dogfood

Run selected candidate and disconnected fallback on the same deterministic
40-view native subset (first eight development views per family), plus the
existing compiler-context FP32 as a reference: 120 actual native calls, 80 model
predictions. First view per family/arm also executes compiled Go (15 executions,
180 function invocations). Bind canonical input hashes, original source and
model hashes, candidate attempts and independent actuals. Native compiler main
pin `b431ef7547a968e2532fa2dcf0891ee44ec2ee86`, Go 1.27.1, SDK .9. No network
provider, online optimizer or source authority from model confidence.

Publish source and protocol before training, fresh output directories, bounded
child processes and append-only captures. Publish own weights, resources, Go
parity, full breakdowns and raw native evidence. Five trits/byte is 1.6 disk bits;
ternary runtime remains decoded int8. CPU/RSS measurements are per process, not
whole-host utilization. Retain disconnected deterministic continuation and all
representation declines. This study does not change compiler context semantics.
