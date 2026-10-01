# Feedback over decisions that still vary

## Plan authored before measurement

The earlier interacting-body native study observed 97 of 438 feedback predictions
on coordinates constant in all remaining paths. This iteration adds the explicit
`Session.ReconsiderUnfixed` API. Existing `Reconsider`, disconnected execution,
cases, intentions and models retain their contracts. It ranks only varying
coordinates, then every candidate still passes type checks and finite tests.

The session counts committed masks in `[16][2]uint16`, exactly 64 bytes, without
retaining old attempt logs. Each option occurs in half the declared binary space;
even with 16 decisions a count cannot exceed 32,768. Once all masks with the other
option have been attempted, this coordinate is constant among unattempted masks.
Interrupted candidates consume no count; type rejections are committed attempts.
Original model identity, CI hint syntax, cancellation and round limits are checked
before a skip. A hashed receipt lists the derived constant option and remaining
mask count, without fabricating a model prediction. Common log factors become
zero; floating-point near ties can differ from the earlier ranking on other data.

Measure the same 72 existing development views and four own tiny models under
both APIs: 576 SDK sessions, 288 pairs. Each pair runs legacy first, then the
opt-in method, using the same loaded model and prepared immutable plan. Maximum
four candidates, one attempt per batch, three feedback rounds. This fixed order
is not a randomized timing experiment. Loaded-model prepared-session wall time
excludes model loading, plan preparation, native compiler launch and Go execution.

The legacy SDK arm must be compared against all 288 recorded native SDK 2.5
feedback results, including candidate sequence, emitted function AST and call
counts. Every selected SDK Go function from both arms is actually executed once
on the native study's 16-input union, including int64 boundaries, and reconciled
against the independent integer-state oracle. Deduplicated executions are shared
observations, not additional function invocations. Raw captures precede inspection.
Offline replay checks hashes, original model/cases/progress/failure links, constant
coordinates and actual-Go values without inference or subprocess execution.

This experiment exercises own-model bounded bilingual construction. It adds no
language capability, new model training, upstream Laya calls or new native compiler
deployment. Main continues to use SDK 2.5 and default `Reconsider`. Partial and
contradictory outcomes remain evidence; first-shot accuracy is not acceptance.
Earlier broad model regressions remain valid. Views, paired policies and repeated
tests are not independent new experiments or an untouched language benchmark.

## Actual recorded result

Runner source: `e5d115f79944765d7a675f2f99664d556333ef2a`. Raw report SHA256:
`41b0e37c2ba99fa44d2173a25e4c2d7174e21115b9dff0a29d3c512664bf839e`.

All 576 SDK sessions completed: 1,931 actual predictions in total, split into
1,014 legacy and 917 opt-in predictions. Initial predictions are 576 in each arm;
feedback falls from 438 to 341. Thus 97 calls are skipped: 22.15% of feedback and
9.57% of total predictions in the opt-in arm. All 288 pairs retain candidate
sequence and final body/finite outcomes. The legacy SDK matches all 288 original
native references. Twelve selected Go programs were actually executed, with 192
function evaluations against the independent oracle. This iteration adds zero
native compiler invocations, training steps, GPU work or upstream Laya requests.

| Model | Predictions, legacy → opt-in | Skipped | Attempts in each arm | Session p50 ms, legacy → opt-in |
|---|---:|---:|---:|---:|
| Parent FP32 | 246 → 221 | 25 | 147 | 0.05423 → 0.05063 |
| Feedback FP32 | 256 → 232 | 24 | 152 | 0.05640 → 0.05494 |
| Feedback PTQ | 256 → 232 | 24 | 152 | 0.05788 → 0.05665 |
| Feedback QAT | 256 → 232 | 24 | 152 | 0.05842 → 0.05960 |

Each row uses 72 repeated views per arm. QAT's median increases by approximately
0.00119 ms despite fewer predictions. These descriptive loaded-model SDK session
times include TDD, receipts and rendering, exclude native startup/model loading
and are not causal speedup evidence. Outliers are retained. Host CPU utilization
and GPU utilization were not measured. Fixed counter storage is 64 bytes; this is
not total session/process memory. The reproducible
[descriptive metrics](../publication/unfixed-feedback-descriptive-metrics-20261001.json)
also retain all per-arm p95 values and their calculation scope.

The [public SDK 0.2.6 release](https://github.com/kimjooyoon/gooo-decision-runtime/releases/tag/v0.2.6-experimental)
is source-bound to this research implementation. SDK source
`4d59a9c5346d07c05b5074659b53f898df8f728c` passed
[formatting, vet, unit and race CI](https://github.com/kimjooyoon/gooo-decision-runtime/actions/runs/36823126075).
Its 24-file extraction manifest is
`843a5b1ea47969c188e1724b2ef42c7b591c6f0081adb4df90cf7b64e5cf3c49`.
Initial SDK validation correctly rejected the old manifest pin; the extraction
pin was updated to the actual new manifest, then all checks passed. A mistyped
runner SHA was rejected before any prediction, directory creation or Go execution.

## Public evidence

The [immutable Hugging Face appendix](https://huggingface.co/asketeddy/gooo-feedback-path-tiny-v1/tree/cd99086ea108737fd2bcd1ad2d0085755e7ace1b/research/unfixed-feedback-sdk-20261001)
has 593 allowlisted raw entries in a 1,165,542-byte compressed archive. All seven
publication files and every archive entry passed anonymous byte verification
(eight HTTP requests; no credentials). The original 25-file core and previous
nine-file compound appendix were independently verified at the same revision.
These verification steps execute zero predictions, native calls or Go processes.
The core model card, weights and previous negative studies remain unchanged.

Native main remains SDK 2.5; this release adds an explicit SDK API and measured
development evidence. Compiler adoption needs a source-bound integration of the
new opt-in method. No general ordering equivalence or first-shot accuracy is
inferred from this small reused cohort.
