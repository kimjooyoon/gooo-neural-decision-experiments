# Native feedback over varying typed coordinates

## Plan authored before measurement

Adopt the explicit SDK 0.2.7 `SearchFeedbackBatchesUnfixed` API through native
Gooo's optional `--path-feedback-unfixed` flag. Default model feedback, source
binding, deterministic disconnected operation and verification contracts remain
available. The compiler shares the SDK batch state machine and its cancellation
accounting. The flag requires a model, positive batch step and feedback rounds.

After exact source CI/provenance verification, build a clean Go 1.27.1 native
binary. Reuse six existing views: three interacting compound-body templates,
mask-zero intentions, contradictory eight-case contracts, Korean and English.
Use four frozen own models and both legacy/opt-in modes: 48 compiler calls and
24 legacy-first pairs. This is a narrow integration pilot, not 48 independent
experiments or an untouched language-accuracy benchmark. First-shot correctness
is not an acceptance criterion. Partial completeness and failed cases are retained.

Every native raw capture and process-resource sidecar is saved before inspection.
Bind the compiler revision, original Gooo source, typed document, finite suite,
model identity, progress/failure/feedback receipt links and selected native Go
function AST. Derive constant-coordinate skips from committed masks. Execute each
distinct actual emitted native Go program once on the rederived union of finite
and separate inputs, including int64 boundaries, and compare against the independent
integer-state oracle. Offline audit repeats these checks without inference or
compiler/Go subprocesses; observed clocks are preserved rather than replayed.

Keep native startup, model loading, search, whole-child wall/CPU time and max RSS
in their respective scopes. Fixed arm order and a small reused sample do not
establish causal speedup, host CPU growth or generalized ordering equivalence.
The caller's source-pinned PASS hint remains unauthenticated context; exact GitHub
CI and GoProof are recorded separately. No model training, GPU work or upstream
Laya HTTP calls are planned. The own models' earlier broad regressions remain
valid. Publish the evidence as a compressed allowlisted appendix with hashes;
preserve previous model weights, card, failures and raw studies.

## Recorded feature pilot

Runner `3749fdc6c1f218c36d944a7899bb1125061c6906` passed all four jobs in
[source CI](https://github.com/kimjooyoon/gooo-neural-decision-experiments/actions/runs/36826245347).
The measured clean feature compiler is `d869be3219a7798b6cc3b13c31efa8a6d899ae22`,
SDK 0.2.7, Go 1.27.1; binary SHA256
`9940f3e85d2942774f24134a6a893971c04d2b6823ef221f0199046e33578110`.
Before execution all six canonical native checks and the exact PR/head/base/run
GoProof/provenance receipt were verified from
[CI 36825091552 attempt 1](https://github.com/kimjooyoon/meta-ontology-go/actions/runs/36825091552).

All 48 actual compiler calls completed, forming 24 paired observations. Actual
own-model predictions total 264: 144 legacy versus 120 opt-in. Initial predictions
are 48 in each arm; feedback is 96 versus 72. Thus 24 fixed-coordinate predictions
are omitted: 25% of feedback and 16.67% of total predictions in this small sample.
Every candidate sequence and final body/finite result remains unchanged. All
eight-case contradictory outcomes remain seven passes, one failure, or 87.5%.
These are 336/384 repeated policy-case outcomes, not independent test examples.

Three distinct actual native Go programs were executed on 14 union inputs,
including int64 limits: 42 actual function invocations. All match the independent
integer-state oracle. Each model/mode reuses those actual values; shared values
are not counted as additional invocations. Report and offline audit SHA256 are
both `4dba2dab42c17af2c069a442b18053fc804bc50a32f5b4276799ddf3f96d3962`;
preexecution SHA256 is `b7fd3eedddabac4e37864b05a77d6f65e08dd60b1f626bb83134ab1126e8bdd0`.
The verifier's initial SHA prefix parsing error was fixed and regression tested
before any pilot call; it did not change compiler behavior or the partial result.

| Model | Predictions, legacy → opt-in | Whole-child wall p50 ms, legacy → opt-in | CPU time p50 ms, legacy → opt-in |
|---|---:|---:|---:|
| Parent FP32 | 36 → 30 | 6.2643 → 6.0013 | 5.4385 → 5.3165 |
| Feedback FP32 | 36 → 30 | 5.9635 → 5.9113 | 5.2560 → 5.2300 |
| Feedback PTQ | 36 → 30 | 6.0229 → 5.8995 | 5.2940 → 5.1390 |
| Feedback QAT | 36 → 30 | 6.0111 → 6.0038 | 5.2275 → 5.3020 |

Each cell uses only six repeated observations. QAT's median CPU time increases
despite fewer predictions, and opt-in median RSS is slightly higher in all four
rows. Median child RSS is 17.12–17.28 MiB; CPU is 86.63–88.89% of one core over
child wall time. These are whole-process observations, not host CPU utilization
increases or causal effects. Native bounded-search medians are approximately
0.328–0.341 ms. All outliers/p95 values and narrower native clocks are retained
in [descriptive metrics](../publication/unfixed-native-feature-descriptive-metrics-20261001.json).
The additional counter array is 64 bytes; that does not describe total process RAM.

[PR 1124](https://github.com/kimjooyoon/meta-ontology-go/pull/1124) was squash merged
into dev `6c637f8b164e129a7aab1f24ef608d8c54cb4e75`. The initial merge-commit request
was rejected by dev's linear-history rule; the normal supported squash succeeded.
[Main promotion PR 1125](https://github.com/kimjooyoon/meta-ontology-go/pull/1125)
contains the exact dev tree and sole current-main parent. Main is not yet deployed
at this feature measurement. Its later checks and deployment are separate evidence.

## Public feature evidence

The [immutable Hugging Face appendix](https://huggingface.co/asketeddy/gooo-feedback-path-tiny-v1/tree/97d3be1f05ca7e2f61e79ca084f075b63e0efd1b/research/unfixed-native-feature-pilot-20261001)
contains 104 allowlisted raw entries compressed to 369,474 bytes. All seven
publication files and every archive entry passed anonymous byte verification.
At the same revision the 25-file original core, nine-file compound appendix and
seven-file SDK appendix were independently reverified: 48 files, 52 HTTP GETs,
zero credentials, inference or native/Go subprocesses during verification.
The [source/publication receipt](../publication/unfixed-native-feature-source-publication-20261001.json)
binds exact successful native/runner/SDK CI and proof digests to immutable HF bytes.

The additional implementation-successor workflow failed with the existing
`AXIS_MISMATCH: compatibility is broader than implementation-only` in
[run 36825091588](https://github.com/kimjooyoon/meta-ontology-go/actions/runs/36825091588).
It is outside the six required checks. The failed certificate is retained; the
pilot does not establish all-input implementation-only compatibility or relax
branch policy. Earlier broad checkpoint regressions remain valid.

## Next main measurement, planned before execution

After normal protected snapshot promotion, rebuild the clean actual main revision
and repeat the same 48-call pilot. The runner accepts explicit PASS, FAIL or
UNKNOWN through `--family-ci-status`; a pending post-push run is recorded UNKNOWN,
with its real source SHA. Source-bound promotion proof and eventual main push CI
are retained separately; the hint never authorizes generation or merge. The
offline audit checks the recorded hint and exact feedback prefix, and cannot
silently relabel PASS as UNKNOWN. Invalid hints fail before binary/output access.
No model, corpus or first-shot acceptance threshold changes are planned. Paired
candidate/body differences, if any, are retained. Feature and main views are
repeated deployment observations rather than additional independent experiments.

The [observed CI costs](../publication/unfixed-native-feature-ci-observed-costs-20261001.json)
use GitHub job timestamps. Native feature CI queued 86 seconds before its first
job; all job spans lasted 849 seconds, while the six canonical checks spanned
707 seconds. Their summed job durations were 1,451 seconds because jobs overlap;
that sum is not workflow elapsed time or measured CPU/billing. Runner-source CI
spanned 292 seconds. These full-repository checks have a different workload from
local candidate construction. Local finite feedback therefore stays in the
bounded codegen loop, with source CI evidence used as additional context.

## Actual main deployment and repeated pilot

After all six canonical checks and exact promotion GoProof passed,
[PR 1125](https://github.com/kimjooyoon/meta-ontology-go/pull/1125) was normally
protected-merged at 07:04:03 UTC. Actual main is
`6f69eb116336b4728db6f94a992130ea56003a48`, tree
`51c2eccb0b65bd08e46b1b1f47fecdc6d1c3997b`, exactly the live dev tree. Main
requires exactly six statuses, strict checks/admin enforcement and zero reviews;
no Guardian or override was used. A clean Go 1.27.1/SDK 0.2.7 binary was rebuilt,
SHA256 `77e330a2a865dc80c061928863f3fe82c5649e2db6981782338c3d0bebf8b858`.

Runner `b093d1b7a855dba34051b0b4ff959ba8be45fdf0` passed all four jobs in
[source CI 36828009562](https://github.com/kimjooyoon/gooo-neural-decision-experiments/actions/runs/36828009562).
The repeated main pilot actually made 48 compiler calls and 264 predictions,
144 legacy versus 120 opt-in, again skipping 24. All 24 pairs preserve candidate
sequence, body and 87.5% contradictory-case completeness. Three actual emitted
Go programs were executed on 14 inputs, 42 function invocations, and matched
the independent integer-state oracle. Combined feature/main execution counts
are 96 native calls, 528 predictions, six Go processes and 84 function invocations;
they reuse the same views rather than forming independent new experiments.

The main raw hint remains UNKNOWN because its post-push CI was pending when
executed. That status is part of the exact feedback prefix and is not relabeled
after subsequent CI. Report/audit SHA256 is
`6bf4030ee7fe769fd3befe8b5bee66bb6fe00fe272037153f4197d59cc6818f6`.

| Model | Whole-child wall p50 ms, legacy → opt-in | CPU time p50 ms, legacy → opt-in | RSS p50 MiB, legacy → opt-in |
|---|---:|---:|---:|
| Parent FP32 | 6.0803 → 6.0545 | 5.3285 → 5.2700 | 17.2188 → 17.2656 |
| Feedback FP32 | 6.0513 → 6.1033 | 5.2720 → 5.3300 | 17.2656 → 17.3203 |
| Feedback PTQ | 5.8852 → 5.9094 | 5.1845 → 5.2345 | 17.0938 → 17.1406 |
| Feedback QAT | 5.9920 → 5.9190 | 5.2545 → 5.1940 | 17.1016 → 17.1797 |

Each row has six observations per mode. FP32/PTQ median wall and CPU time rise;
every median RSS rises. Main one-core child CPU medians span 87.44–88.67%, and
bounded-search medians span 0.330–0.352 ms, including a slightly higher parent
opt-in search median. None is host CPU growth or a causal speedup conclusion.
[Full metrics and outliers](../publication/unfixed-native-main-descriptive-metrics-20261001.json)
preserve the exact scope and calculation rules.

The [immutable main HF appendix](https://huggingface.co/asketeddy/gooo-feedback-path-tiny-v1/tree/5e5718256beb43667483154cc0e4464981a66fb4/research/unfixed-native-main-pilot-20261001)
contains 104 raw entries compressed to 369,676 bytes. All seven files/entries
passed anonymous byte verification. Original core (25 files), compound (nine),
SDK (seven) and feature (seven) appendices were reverified at the same revision:
55 files and 60 HTTP GETs, zero credentials or inference/subprocess calls.
The [main source/publication receipt](../publication/unfixed-native-main-source-publication-20261001.json)
binds exact promotion proof and immutable bytes. Subsequent main push CI and
evidence-source CI are recorded separately without changing the raw hint/captures.

## Final source verification

All six required checks passed for actual main `6f69eb116336b4728db6f94a992130ea56003a48`
in [post-push CI 36828207770 attempt 1](https://github.com/kimjooyoon/meta-ontology-go/actions/runs/36828207770).
The downloaded GoProof and append-only receipt independently passed verification,
including exact push/head/base/ref/run/attempt and all six successful source jobs.
Proof SHA256 is `8ffaaa104416276eeb34581e11117fffcae393fa94141154180dfac2814b3286`;
receipt SHA256 is `33ea47d671b774c297e4bdf95cdc9c798bf05f19f93c64dcc6d739d3df856ac4`.
The original captured UNKNOWN remains unchanged.

Complete main evidence, strict frozen replay, raw values, model/process descriptive
metrics and compressed publication passed all four jobs at source
`019e89a1c6121489c64ec6e87dc2bff741047f08` in
[research CI 36828837293](https://github.com/kimjooyoon/gooo-neural-decision-experiments/actions/runs/36828837293).
Complete feature evidence passed all four jobs at
`37d9129852bb75f583660b67887215d46839699d` in
[CI 36827097109](https://github.com/kimjooyoon/gooo-neural-decision-experiments/actions/runs/36827097109).
Final metadata adds these verification outcomes, explicit HTTP-fetch count scope
(automatic redirects not instrumented), and a
[next construction-cost plan](native-construction-cost-next.md). It changes no
code, model/corpus or captures. The proposed retained-model native worker is not
implemented and has no measured speedup claim.
