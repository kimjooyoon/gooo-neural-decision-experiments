# Gooo neural decision experiments

## Three-choice Gooo construction — implementation and authored preparation

Released [Go SDK v0.2.13-experimental](https://github.com/kimjooyoon/gooo-decision-runtime/releases/tag/v0.2.13-experimental)
and [compiler main 774eabb](https://github.com/kimjooyoon/meta-ontology-go/commit/774eabb226f88a317c523ce4efa24a08032066a9)
add a separate source-bound 768/24/8 model interface for ranking eight complete
three-choice bodies. Actual failed-case feedback can rerank the remaining masks;
disconnected, unsupported or oversized inputs continue deterministically with
zero prediction calls. [Native adoption evidence](docs/own-three-choice-native-adoption-20261002.md)
records the three successful exact-source CI runs and clean executable builds.
[Implementation evidence](docs/own-three-choice-sdk-abi-20261002.md)
uses controlled weights and does not claim trained quality.

The [new authored fixtures](docs/own-three-choice-authored-fixtures-20261002.md)
cover eight combinations of conditions, assignments, references, nested branches
and execution order. 3,072 Korean/English views retain all passing-mask ties;
393,216 typed/oracle comparisons and separate compiled-Go execution pass locally.
The [frozen study](docs/own-three-choice-completeness-preregistration-20261002.md)
keeps actual native corpus export/freeze, teacher observation, fresh training,
student/native measurements and public weight verification as separate phases.
The first source-export attempt is retained with a collector correction;
the full native corpus is not yet complete. Teacher observation, training and
trained native measurement have not started; no default model is promoted.
The existing nine v2 models below remain the latest trained edition.

## Own feedback Gooo model v2 — latest experiment

[Nine fresh own models and actual failure feedback](docs/own-joint-feedback-results-20261002.md)
compare uniform initial learning, passing-set initial learning and passing-set
learning from observed teacher failures. All share a fresh initial state; 3,600
actual MPS updates produce FP32/PTQ/QAT variants. These small Gooo structural
judgment models use no inherited Laya or other pretrained weights.

The feedback FP32 arm reduces additional candidates 488→432 (-11.48%) against
its matched uniform control in 384 development views. Native main then generates
240 actual Gooo bodies, independently executes 240 emitted Go programs and
passes all 3,840 ordered finite checks. Native extra candidates decline 57→53
in the matched 48-view subset. Four-candidate completion includes deterministic
enumeration; it is not universal semantic accuracy. Calibration still selects
the frozen v1 independent model, and quantization regressions remain retained.

Warm prediction is about 10.8 µs with zero heap allocations. Feedback-policy
codegen median is 10.792 ms and measured child peak RSS median is 17.55 MiB.
Lower candidate counts have not established a compiler wall-time speedup.

The [public immutable v2 edition](https://huggingface.co/asketeddy/gooo-joint-feedback-tiny-v2/tree/b4e9a3e50b50e893abc52a36f49eacf032aded50)
contains nine models, PROV-O lineage, frozen protocols and 614 raw evidence
members. [Anonymous publication verification](publication/own-joint-feedback-public-verification-20261002.json)
checks all 48 payloads against the pinned local manifest. The v2 CI job audits
published evidence and executes new SDK sessions and native generations.
[The recorded source CI](https://github.com/kimjooyoon/gooo-neural-decision-experiments/actions/runs/36952987501)
passes all ten jobs. Its [permanent raw CI appendix](https://huggingface.co/asketeddy/gooo-joint-feedback-tiny-v2/tree/60b28e80057dbe7f376e936487a9fe0db9b46683/research/ci-own-feedback-v2-36952987501)
preserves 519 raw records and 15 reports, with
[anonymous Go verification](publication/own-joint-feedback-ci-public-verification-20261002.json)
of all 19 published files. CI replays do not count as newly authored intentions.

## Own joint Gooo model v1 — preceding baseline

[Six fresh own models and direct Gooo execution](docs/joint-composition-model-results-20261002.md)
now compare independent choices with one joint four-mask prediction. There are
480 actual MPS optimizer updates, no inherited pretrained/Laya/earlier own-model
weights, and six FP32/PTQ/QAT exports. Complete source-bound Korean/English
intent ranks legal compiler-owned fragments; actual failure feedback can rerank
remaining paths. Disconnected or unsupported input continues deterministically.

On 384 development function views, joint FP32 reduces predictions 1,447→777
but increases extra assembly attempts 448→455. All seven policies complete their
16-case finite contracts by four candidates. Calibration selects independent
FP32; no default model is promoted. Warm joint prediction is about 10.7 µs with
zero per-call heap allocations. Ternary files occupy 2,590 bytes and execute as
decoded int8 tensors, separately from whole-process memory.

Gooo [main 363a3d8](https://github.com/kimjooyoon/meta-ontology-go/commit/363a3d8aa365c35dd634c241248b444de0050973)
uses released SDK v0.2.12 and supports an explicit joint path model after source
binding. Actual main dogfood makes 192 compiler calls and 470 predictions,
independently compiles 192 emitted outputs and passes 3,072 ordered Go function
invocations. Joint prediction reduces calls in that native subset too, but
median compiler-child latency remains about 10.9 ms. Global host CPU delta is
unmeasured; process CPU/RSS observations are retained with their scope.

The [complete immutable public model edition](https://huggingface.co/asketeddy/gooo-joint-path-tiny-v1/tree/5fb63c8092b4f1a2b1d0eb99b6d17f42cd6aac66)
contains all six models, both frozen protocol documents, negative comparisons,
PROV-O chains and raw source/curriculum/SDK/native evidence. Anonymous Go
verification passes for 39 payloads, the manifest and 479 regular ZIP members.
The new CI job retrieves this pinned edition, reconstructs original evidence
and performs new direct native generations and compiled-Go executions.

Install both compiler executables with Go 1.27.1, then provide source and its
typed legal path plan:

```sh
GOTOOLCHAIN=go1.27.1 go install github.com/kimjooyoon/meta-ontology-go/cmd/gooo@363a3d8aa365c35dd634c241248b444de0050973
GOTOOLCHAIN=go1.27.1 go install github.com/kimjooyoon/meta-ontology-go/cmd/gooo-body-worker@363a3d8aa365c35dd634c241248b444de0050973
gooo body-codegen --json --path-plan plan.json --path-model joint/models/fp32/model.json --path-step-attempts 1 --path-feedback-rounds 3 --path-feedback-unfixed --activity ChoosePath source.gooo
```

Joint v1 supports exactly two binary choices. More decisions use deterministic
continuation; future own-model work expands variable/assignment/condition
composition and measures completeness, total attempts and inference cost.

## Own source-aware small model

The [actual fresh-model training and native dogfood](docs/fresh-composition-model-results-20261002.md)
now adds 960 MPS optimizer updates, six own random-init exports and 144 native
compiler calls with 2,304 independently executed generated-Go invocations.
Source v3 FP32 reduces development continuation attempts 524→397; the frozen
calibration-selected v3 QAT needs 470. All negative comparisons remain recorded.
Warm selected inference is about 9.7 microseconds, with zero per-call heap
allocation. Published ternary disk packing decodes to int8 runtime matrices.
The [standalone own-model edition](https://huggingface.co/asketeddy/gooo-semantic-composition-tiny-v1/tree/1102d2c1c157cd72054cc9fd16f81a61043cc660)
publishes all six variants, PROV-O model derivations and 327 archived evidence
members. [Anonymous byte verification](publication/semantic-composition-tiny-public-verification-20261002.json)
passes for 35 payloads plus the manifest; all eight publication-source CI jobs
pass. The [growth roadmap](docs/own-small-model-growth-roadmap-20261002.md)
prioritizes new frozen source, joint-path and Korean/English judgment studies.

The [source ABI and fresh curriculum preparation](docs/semantic-source-v3-preparation-results-20261002.md)
connects our optional small model directly to typed Gooo source in protected
native main. Six fresh two-choice compositions provide 9,216 audited decision
rows across v2/v3 and Korean/English. Both source and curriculum packages are
public and anonymously byte-verified; the first capture rejection is retained.
Preparation adds zero optimizer updates or weights. The frozen next training
stage measures completeness and continuation cost, retaining source compression
conflicts and finite target ties.

## Continued partial construction

The [opt-in varying-coordinate feedback study](docs/unfixed-feedback-study.md)
uses own frozen tiny models in 576 actual Go SDK sessions. Across 288 pairs,
`ReconsiderUnfixed` skips 97 fixed-coordinate predictions (1,014 → 917) with
the same candidate sequences and final bodies/finite results. Twelve actual Go
programs execute 192 independent-oracle checks. The counter costs 64 fixed bytes
per session. This is SDK development evidence; native main keeps SDK 2.5/default
feedback. Earlier model regressions and finite expression limits remain recorded.

The [interacting four-path study](docs/compound-path-study.md) adds three new
body compositions involving local references, assignment, subtraction, if
branches and execution order. Actual main codegen ran 648 times and generated
twelve distinct executed Go programs. Observed failure context changed sixteen
candidate sequences among 288 pairs; final code and finite/separate outcomes
match. QAT saved two candidate attempts but used 112 extra predictions, while
the parent spent one extra attempt. At that native revision, the audit recorded
97 predictions for already fixed coordinates as a cost opportunity. Sparse ambiguity and
commuting updates retain distinct functional and structural denominators.
[The public compound appendix](https://huggingface.co/asketeddy/gooo-feedback-path-tiny-v1/tree/c902d14550eda5de08ff7a5bbd8507e60f02c31f/research/compound-path-main-20261001)
preserves 674 raw records in a 1,729,483-byte archive, with anonymous byte checks
and [successful source CI](publication/compound-path-source-publication-20261001.json).

Korean/English Gooo intentions can now rank bounded structural paths once and
continue finite construction in batches. First-shot accuracy is not the goal:
partial results, failed cases, type rejections, unattempted paths and cost remain
inspectable. Models are optional; the disconnected order is deterministic.

The optional [observed feedback judgment](docs/feedback-path-judgment.md)
can now ask the same frozen model again after a partial batch. It supplies a
bounded failure summary and optional caller CI hint, and changes only remaining
path priority. Go SDK `v0.2.5-experimental` exports this bounded API;
default sessions still rank once. The native feedback feature was deployed to
main in [compiler PR 1119](https://github.com/kimjooyoon/meta-ontology-go/pull/1119),
with [source-bound post-main CI](publication/native-feedback-main-push-20261001.json).
This is an experiment with an existing model, not feedback training.

The [continued-judgment repair study](docs/continued-judgment-results.md) preserves
the failing long-input comparison and shows four actual native calls with 24
local predictions. Optional context overflow now records zero-call declines and
continues the same 64 candidates, retaining 6/7 completeness in both languages.
One deduplicated generated Go execution agrees with native and typed values.
The [public HF repair appendix](https://huggingface.co/asketeddy/gooo-typed-path-tiny-v1/tree/15efbf6c49e15b59ac4de36c185df01bc723c4e3/research/continued-judgment-20261001)
adds synthetic evidence without changing weights. The native SDK upgrade is
merged to dev in [compiler PR 1120](https://github.com/kimjooyoon/meta-ontology-go/pull/1120),
with [verified six-check evidence](publication/context-decline-dev-20261001.json).
[Main promotion PR 1121](https://github.com/kimjooyoon/meta-ontology-go/pull/1121)
is merged at `4dced73b26dde567cb7129f3e4ba5733850196d0`. The clean main
compiler separately preserves long-context partial construction and disconnected
deterministic replay; feature measurements remain separate from main observations.
The [main deployment receipt](publication/context-decline-main-20261001.json) and
[verified main-push six-check proof](publication/context-decline-main-push-20261001.json)
record the protected merge, exact source and actual main smokes.

The [first finite-feedback GPU tuning](docs/feedback-path-training-results.md)
completed 760 optimizer steps on our own 12,728-parameter model. It preserves
ambiguous finite targets and original intentions. The six model arms and offline
control show a regression against the parent, which remains recorded. The new
models have also been used inside actual Gooo codegen on both feature and main
sources, with hashed captures and independently executed generated Go. See the
[experimental model card](docs/model-card-feedback-path-v1.md).
The [public new-model edition](https://huggingface.co/asketeddy/gooo-feedback-path-tiny-v1/tree/4a52615b4889966c3f7e6916b456a554e132b38d)
contains all three checkpoints and synthetic evidence. Go verified all 25 files
anonymously at the immutable revision; the [public-byte receipt](publication/feedback-path-model-public-verification-20261001.json)
is separate from inference and training. [Observed continuation curves](docs/feedback-continuation-curves.md)
track completeness at candidate batch boundaries rather than only the final outcome.

The [five-family native study](docs/native-feedback-families-study.md) now adds
1,080 actual native calls and twenty actual generated Go programs. Sparse tests
expose remaining behavior even when every selected finite case passes. All 480
two-option feedback pairs preserve outcomes while spending 244 extra predictions.
SDK v0.2.5 now skips another ranking call when one declared path remains;
[compiler PR 1123](https://github.com/kimjooyoon/meta-ontology-go/pull/1123) merged
this continuation into main. A separate 1,080-call main matrix preserves all
paired code, finite/separate-input outcomes and candidate counts while reducing
actual predictions from 1,204 to 960. Its original caller CI hints remain
`UNKNOWN`; actual source checks are recorded separately.
[Main evidence on Hugging Face](https://huggingface.co/asketeddy/gooo-feedback-path-tiny-v1/tree/e720276887cca90817d174494e1d6238551da0b9/research/native-family-optimized-main-20261001)
contains 1,105 raw records in an 878,049-byte archive. The seven-file appendix,
25-file core and both earlier appendices were anonymously verified at that
immutable revision. Model weights remain unchanged; the new checkpoints'
regressions against the parent remain visible.
The [main integration receipt](publication/sole-path-native-main-integration-20261001.json)
binds the protected merge, successful post-push six-check Go proof, main captures,
successful research source CI and anonymous HF verification. The auxiliary
predecessor certificate mismatch remains recorded.

The [actual native integration study](docs/native-feedback-results.md) measured
28 native generations and 246 local predictions with three existing models.
All 12 model pairs emitted identical Go and finite completeness. Native process
wall medians were 8.816→9.021 ms and maximum RSS medians 20,865,024→20,971,520
bytes. Captured Go was separately executed twice and agreed with both
interpreters. These are one-intent, baseline-first observations; host CPU
utilization and new training were not measured.
The [public HF native evidence appendix](https://huggingface.co/asketeddy/gooo-typed-path-tiny-v1/tree/dc87b6c8f6784accd60bd93535c14e2d0b20f470/research/native-feedback-integration-20261001)
is anonymously digest-verified; the earlier nine weight and metadata files,
model card and previous evidence appendix remain unchanged.
The [next study design](docs/continued-judgment-next-study.md) specifies source
and intention-family splits, retained partial outcomes and resource denominators
for future feedback training; it does not claim a newly trained model.

The [actual feedback pilot](docs/feedback-path-pilot-results.md) made 246 small-model
predictions, including 102 reconsideration predictions, and 32 native generations.
All 16 pairs emitted the same Go and final case scores. One pair needed fewer
candidates, two needed more, and there was no aggregate functional gain. Raw
[captures and independent audit](runs/feedback-path-pilot-fixed-20261001/) retain
those observations; existing weights remain unchanged.

[Session API and limits](docs/incremental-typed-paths.md),
[Korean judgment and partial-construction policy](docs/judgment-and-partial-construction.md),
[actual 288-call native study](runs/incremental-native-20261001/), and
[cost summary](runs/incremental-native-20261001/summary.json) show 32 same-result
pairs, 1,152→144 model judgments, and 6,340→1,268 candidate attempts. Sixteen
inconsistent contracts retain six of seven satisfied cases. This compares eight
restart requests with one continued request on one compound intention; it is
not new training, general language accuracy, or 32 independent experiment ideas.
The optional native flag is `--path-step-attempts 8`, backed by public Go SDK
`v0.2.2-experimental`. [HF edition and scope](docs/hf-typed-path-v1-incremental.md)
preserve previous weights and publications.

The [main deployment receipt](publication/incremental-typed-path-main-20261001.json)
records protected main `307159f041644a3aa56dfd325c345f5325aec902`, exact six-check
PR proof and four main smokes: real model execution, deterministic offline
replay, and preserved 6/7 partial output. The
[experimental Go SDK release](https://github.com/kimjooyoon/gooo-decision-runtime/releases/tag/v0.2.2-experimental)
is public. [Laya cancellation repair](publication/laya-cancellation-repair-20261001.json)
separately fixes a buffered-response race with 100 repeated mock tests per case;
it does not add live model or training measurements.

[Actual post-main CI](publication/incremental-typed-path-main-push-20261001.json)
also passed all six checks; its downloaded proof and provenance receipt passed
the Go verifier. This is separately bound to the merged main source.

Public experimental Gooo-specific natural-language to typed IR decisions.
Runtime, orchestration, data generation and compiler bridges are written in Go.
Python is used for offline PyTorch/MPS training and checkpoint export. The
optional Laya comparison connects from Go to its existing upstream PyTorch
service; the tiny-model runtime has no Python dependency.

First pilot: eight bounded binary-operation decisions, English/Korean synthetic
instructions, template-grouped train/calibration/test splits. This is a small
classifier, not a full natural-language compiler. Laya is the research baseline;
the first tiny model is independently initialized, not copied Laya weights.

## Current compiler dogfood model

The [compiler/PROV-O model v2](https://huggingface.co/asketeddy/gooo-compiler-prov-tiny-v2)
fine tunes our own first model using Gooo declaration and PROV-O context views.
The Go 1.27.1 compiler integration is merged into `meta-ontology-go`'s `dev`
branch in [PR 1110](https://github.com/kimjooyoon/meta-ontology-go/pull/1110) and
promoted to `main` in [PR 1111](https://github.com/kimjooyoon/meta-ontology-go/pull/1111).
The model has actually been used in native Gooo body generation, with a separate
deterministic disconnected baseline. The post-main machine CI proof passed.

## Bilingual structural path model and bounded TDD

The new [typed path model](https://huggingface.co/asketeddy/gooo-typed-path-tiny-v1)
publishes three training comparisons and nine small model bundles for local
references, assignment targets, operand order, branch layout and execution order.
Conservative confidence selection abstains on the recorded development cohort;
the explicit TDD mode uses model scores to prioritize typed candidates.

On the reserved bilingual probe, FP32 ranking reduces candidate evaluations from
2,880 to 2,468 (14.31%). All four arms, including deterministic search without a
model, pass 15,360/15,360 unseen-input cases. Median search is 76.834 µs with FP32
versus 69.584 µs without a model; candidate savings did not yield a measured
wall-time speedup. These are five closed two-option families, 640 instructions
and 1,920 context views, not arbitrary language-to-code tasks.

See [the structural workflow](docs/typed-path-model.md),
[HF model card](docs/hf-typed-path-v1-model-card.md) and
[raw bounded TDD evidence](runs/typed-path-reserved-probe-tdd-20261001/).
Native in-process structural inference is now merged to compiler `main` in
[PR 1113](https://github.com/kimjooyoon/meta-ontology-go/pull/1113), using the public
[Go SDK v0.2.0-experimental](https://github.com/kimjooyoon/gooo-decision-runtime/tree/v0.2.0-experimental).
`body-codegen --path-plan [--path-model]` binds the fallback to the actual source,
ranks each decision once, checks finite candidates and emits the selected body.
Omitting the model retains deterministic finite search. Development is direct,
without subagents.

The [fresh native compound probe](runs/native-typed-path-compound-20261001/)
records 32 native calls, 72 in-process predictions and zero external calls over
one compound intent. Eight-attempt arms pass all 160 independent arithmetic
observations; four-attempt arms retain failures, including the Korean model
arms' 6/60 success. These are budget/language/model views, not 32 distinct ideas.
Deliberately inconsistent finite cases retain 66.67% functional completeness.
See the [native measurement scopes](docs/hf-typed-path-v1-native-direct.md),
[actual main smoke and promotion receipt](publication/native-typed-path-main-promotion-20261001.json)
and [anonymous HF verification](publication/typed-path-native-main-public-verification.json).
The nine model weight bundles are unchanged in this integration update.

## Prepared conditional paths and completeness

The Go-only [SDK v0.2.1-experimental](https://github.com/kimjooyoon/gooo-decision-runtime/tree/v0.2.1-experimental)
owns one immutable validated plan and fallback. Compiler
[PR 1114](https://github.com/kimjooyoon/meta-ontology-go/pull/1114) is merged to
`dev`; [PR 1115](https://github.com/kimjooyoon/meta-ontology-go/pull/1115) is merged
to `main` after six exact machine checks and verified source-bound proof.
Repeated native preflight preparation goes from four
to one; each combined candidate retains its type/scope checks and native replay.

The [paired raw study](runs/prepared-native-conditional-20261001/) uses six
interacting Boolean/Integer/conditional decisions, English/Korean instructions,
four arms and budgets 8/64. Five balanced repetitions produce 160 actual native
generations and 720 fresh local predictions. All 80 baseline/prepared pairs have
identical emitted code and search results. Nine independent arithmetic inputs
pass 645/720 per version; full-budget results pass 360/360, with authored internal
structure agreement separately 220/240. Fifty partial calls remain recorded.

On this one local workload, median native stages are 3.202/1.562 ms and complete
child wall times are 9.839/8.448 ms. This is not a universal speedup or an estimate
of general natural-language accuracy. No weights are trained or selected on the
new fixture. [Measurement scope and public model-card addition](docs/hf-typed-path-v1-prepared-native.md)
disclose peak child RSS, one-core CPU/wall, the first fixture failure and the
functional/structural distinction. The fixed compact HF publication adds 11
evidence files and preserves all nine existing model bundles byte for byte.
The [promotion and main smoke receipt](publication/prepared-typed-path-main-20261001.json)
records the exact source/tree and three actual clean main invocations. Two
model-free invocations have identical emitted code and search results; one
model-backed invocation makes six fresh local predictions. All three retain
7/7 declared finite cases. These smokes are excluded from the primary study.

For the earlier structural integration, actual post-main
[CI 36787975612](https://github.com/kimjooyoon/meta-ontology-go/actions/runs/36787975612)
also passed; its downloaded proof and append-only receipt passed the Go verifier.
The [post-main evidence receipt](publication/native-typed-path-post-main-ci-20261001.json)
binds that result and the research repository's four successful jobs to their
actual source revisions.

On the new synthetic held-out set, FP32 and QAT score 768/768 and PTQ 643/768.
These are 256 original instructions in three views, not 768 independent tasks.
Two observed compiler intents were added to training; their native raw-operation
repair score improves from 0/6 to 6/6 across three variants. Final generated code
passes 36/36 finite cases both before and after training because local TDD already
repairs candidate selection. The disconnected baseline passes 12/12 with no
model calls. This demonstrates repair and integration, not new unseen-task
codegen accuracy or reduced candidate-evaluation work.

See the [model card](docs/hf-compiler-prov-v2-model-card.md),
[training and raw execution evidence](runs/compiler-prov-v3-mps-20261001/),
[development feedback design](docs/compiler-model-dogfood.md) and
[fixed HF bundle](publication/hf-compiler-prov-v2/).
The previous model/release results below remain frozen historical observations.

The subsequent [128-program body comparison](runs/compiler-prov-v3-bodyplan-20261001/comparison.json)
finds direct functional case success of 74.22%→84.84% for FP32, 53.05%→49.84%
for PTQ and 71.56%→60.00% for QAT. **Use FP32 for this measured broader workflow.**
Training-only TDD search reaches 100% on these finite cases for all variants and
the deterministic baseline; it does not erase initial-choice regressions.
See the [reviewed HF card](docs/hf-compiler-prov-v2-reviewed-model-card.md).

Compare float32, post-training ternary quantization and ternary-aware training.
The ternary alphabet has theoretical log2(3)=1.585 bits; base-3 packing of five
weights per byte uses 1.6 bits/weight plus scales, biases and metadata. Training
uses floating point master weights. Go inference must report actual resident
memory and latency separately from packed-file size.

Only generated public synthetic material, source and allowlisted model artifacts
will be published. Private repository data, credentials, local paths and device
identifiers are outside the export bundle.

Status: the first MPS pilot exports FP32, PTQ ternary, and QAT ternary bundles.
The independent Go audit matches all 96 saved parity rows within `1e-4` and
scores the 256-row held-out test split. Full results and input hashes are in
`runs/pilot-mps-20260930-v1/go-audit.json`.

The three model variants and the 17-file synthetic-data/evidence bundle are
public on [Hugging Face](https://huggingface.co/asketeddy/gooo-ir-operator-tiny-v1).
Commit `1d1741fe6b88d5121e7fdba45fd90cefdd9a6c91` was anonymously fetched and
verified against every allowlisted SHA-256. The publication receipt is
`publication/hf-publish-v1.json`; the displayed model card source is
`HF-MODEL-CARD.md`. This is an independent tiny-model pilot; no Laya fine-tune
or production compiler integration is claimed.

Standalone Go executables with all three model variants are available in the
[v0.2.0-experimental release](https://github.com/kimjooyoon/gooo-neural-decision-experiments/releases/tag/v0.2.0-experimental)
for macOS ARM64, Linux AMD64 and Linux ARM64. It adds `gooo-body-compose` to the
single-request and persistent-stream tools. Its source is pinned to commit
`72813219c891285a5c6af43406cb0688d3a8b4c8`; two independent builds were
byte-identical and all four uploaded assets passed fresh anonymous download
verification. Darwin smoke checks executed all nine CLI/model combinations;
Linux executables were inspected and integrity-checked, not run on this host.
See [build verification](publication/release-independent-verification-v0.2.json),
[public asset verification](review/release-v0.2-publication-20260930/verification-receipt.json),
and the [small body example](examples/adjust-balance/README.md).

The earlier two-tool
[v0.1.0-experimental release](https://github.com/kimjooyoon/gooo-neural-decision-experiments/releases/tag/v0.1.0-experimental)
remains available
for macOS ARM64, Linux AMD64 and Linux ARM64. The release is pinned to commit
`6d306d3aae547cb6f5bffa503303171f4145f4c0`. Two independent builds produced
byte-identical archives; all four uploaded assets were anonymously downloaded
and verified against their SHA-256 checksums. See the separate
[release publication receipt](publication/github-release-v0.1.0-experimental.json)
and [build verification](publication/release-independent-verification-v1.json).

| Model | Test correct / 256 | Packed weights | Resident tensor arrays | Hot prediction, M4 |
| --- | ---: | ---: | ---: | ---: |
| FP32 | 256 | 50,912 B | 50,912 B | 7.86 us |
| PTQ ternary | 245 | 2,759 B | 12,896 B | 9.34 us |
| QAT ternary | 242 | 2,759 B | 12,896 B | 9.30 us |

All hot measurements used zero allocations. Ternary reduced resident tensor
arrays by about four times but was about 19% slower in this run. It uses int8
decoded matrices, not a packed 1.58-bit inference kernel. See the model card
and raw audit for calibration, template-level limits, cold startup variability
and memory accounting.

## Go inference runtime

The [bounded public SDK kernel measurement](publication/go-runtime-sdk-v0.1-benchmark/README.md)
records latency, allocations, process CPU, and lifetime peak RSS with explicit
measurement scopes. The [first native body-fill capture](publication/native-tiny-body-fill-2519/PUBLICATION.md)
preserves six real trained-model invocations, four deterministic fallbacks, two
incorrect model-applied proposals, and 36/36 corrected finite generated-Go case
executions. It also records the revision's module-import CI failure; those
observations are not rewritten by later adapter fixes.

The small, separately versioned [Go runtime module](https://github.com/kimjooyoon/gooo-decision-runtime)
is available as `v0.1.0-experimental` for compiler dependencies; its
[public import verification](publication/go-runtime-sdk-v0.1-public-consumer/README.md)
uses the three existing bundles and no local replacement.
The [public Go library](docs/go-library.md) exposes the same bounded model and
typed binary IR API for imports from other Go modules. Models remain separate
files; each concurrent call owns its workspace. This API identifies local tiny
predictions by variant and weights digest and makes no Laya-provider claim.

`cmd/gooo-decision` loads the strict `model.json` plus sibling `weights.bin`
bundle, validates its digest and tensor layout, then accepts one JSON request on
stdin. The request contains an instruction and two typed identifiers. The
response contains the ordered eight-label probabilities and either a closed,
typed binary IR node or an abstention. The model cannot return source text.

Input must be valid UTF-8 and contain 1–512 bytes; the runtime rejects longer
inputs. Feature extraction lowercases ASCII bytes, hashes byte bigrams and
trigrams with FNV-1a, and L2-normalizes the 256 counts. One reusable workspace
uses 1,248 bytes per worker; the fixed eight-logit plus eight-probability
prediction arrays use 64 bytes inside an 80-byte `Prediction` struct.

FP32 inference stores its matrices and biases in 50,912 bytes of float32 tensor
storage. Ternary inference decodes the two packed matrices into one contiguous
int8 array and keeps 56 float32 biases in one array: 12,672 matrix bytes plus
224 bias bytes, or 12,896 tensor-array bytes total. The two float32 matrix
scales use another 8 bytes and are applied after each matrix dot product.
Tensor-array totals exclude model metadata and Go object headers. This is an
int8-decoded runtime representation; the benchmark does not claim a packed
low-bit inference kernel or faster ternary inference. Packed bundle size,
resident tensor storage, per-worker scratch, and latency are measured
separately.

Run the focused runtime tests and comparative microbenchmark with Go 1.27:

```sh
go test ./internal/decision ./cmd/gooo-decision
go test ./internal/decision -run '^$' -bench '^BenchmarkPredictInto$' -benchmem
go run ./cmd/gooo-model-audit --models runs/pilot-mps-20260930-v1/models --parity runs/pilot-mps-20260930-v1/go-parity.json --dataset data/synthetic-ops-v1/dataset.jsonl --output /tmp/gooo-audit-replay.json
```

The audit checks all 96 saved feature/logit/probability rows within absolute
tolerance `1e-4`, then scores only the 256 frozen test examples. It records
planned and observed counts, label/language/template breakdowns, calibration,
typed-IR/source-expression checks, model/input hashes, and per-worker memory.
Adding `--decision-bin PATH --cold-dir DIR` also captures three raw cold CLI
runs per model variant with process wall time and peak RSS.

## Persistent bounded execution

`cmd/gooo-decision-stream` loads one model and accepts NDJSON requests. One to
eight workers share the read-only weights; each worker reuses its own fixed
workspace. Job and result channels are bounded by worker count. Results are
emitted as they finish, identified by correlation ID and input sequence. The
runner closes cancellable input/output to interrupt blocked I/O and joins its
watcher on normal completion. See [stream protocol and resource boundaries](docs/streaming-inference.md).

The zero-allocation benchmark covers `PredictInto` only. JSON parsing, typed
response construction, encoding and channels allocate separately. A rejected
record does not prevent later records from completing; a model abstention is
visible inside the transport result and supplies no IR.

The [six-condition local stream measurement](runs/stream-end-to-end-v2-20260930/README.md)
completed 24,576 records, repeating 256 frozen test rows. FP32 processed a
4,096-record batch in 77.7 ms with one worker or 54.2 ms with four. The latter
used 176.7% average CPU on a one-core basis and 11.6 MiB child peak RSS. This
measures batch throughput, not individual request latency or host CPU increase.
CI also exercises the persistent stream without Python, GPU or providers.

## Runtime hardening after the pilot

The v1 published audit remains unchanged. Follow-up v2/v3 records cover raw
invalid UTF-8 rejection, Go keywords/blank identifiers, nonfinite intermediate
values and atomic output on failed inference. The v3 audit again matched all
96 parity rows and observed all 256 test rows per model with unchanged scores.
The original v2 source is separately archived. See
[hardening evidence](runs/runtime-hardening-v3-20260930/report.md).

Controlled study status and the completeness dimensions are in
[next-experiments.md](docs/next-experiments.md).

## Multi-node body experiment

The [typed body-plan path](docs/multi-node-body-plans.md) now supports authored
expression arenas, typed operation holes, scoped locals, assignments, nested
branches and returns. `cmd/gooo-body-compose` fills multiple holes using explicit
model choices or deterministic fallbacks. A bounded training-only search is
optional. The [Go study runner](tools/evaluate-bodyplans/README.md) validates
native Gooo generation and independently compiles/executes emitted Go, with
explicit planned, observed and unknown counts. This extension is separate from
the immutable v1 operator-model results; the v0.2 executable release includes
the new body composer while preserving the original model weights.

The [frozen 128-scenario study](runs/body-plan-v1-laya-7d626b9-20260930/README.md)
completed 1,280 native Gooo generation cells and 45,200 independent Go case
executions, with zero unknowns. It made 222 actual Laya calls and 666 tiny-model
predictions. Initial heldout behavior was 74.2% for FP32 and 73.8% for Laya;
training-only finite search reached 100% for every arm, including model-free
search. Model proposals reduced observed search work; final correctness is not
exclusive model contribution. Laya HTTP median/p95 was 40.9/46.8 ms on CPU.
See the raw run and independent audits for the comparison and resource limits.

The [slot-backed interpreter](docs/body-plan-memory-layout.md) resolves names
once into scoped array indices. One-plan M4 measurements improved the median
evaluation time from 239.7 to 113.9 ns, with 544 B/5 heap allocations per call
reduced to zero. Compilation still allocates and stores bounded slot tables;
the benchmark does not measure full code generation or inference.

The study also exposed a native compiler mismatch for inferred Integer locals.
The fix is [merged into Gooo main](https://github.com/kimjooyoon/meta-ontology-go/pull/1106),
with all six required machine checks and post-main CI passing. The
[promotion receipt](publication/native-integer-main-promotion-1106.json) records
the exact source, proof and merge bindings, the retained auxiliary dev-run
503 failure, and the remaining Actions-only promotion limitation.

## Native partial construction with varying-coordinate feedback

The [SDK 0.2.7 native pilot](docs/native-unfixed-pilot.md) opts into skipping model
predictions on typed coordinates constant in every remaining path. In 24 paired
bilingual observations, calls fall from 144 to 120 with unchanged candidate order,
body and retained 87.5% contradictory-case completeness. Three actual emitted Go
programs pass 42 independent state-oracle evaluations. QAT median CPU time and
all median RSS values rise slightly despite fewer predictions. Exact CI/proof,
source-bound raw captures, resource sidecars and compressed public HF evidence
are available; main promotion is separate. Reused views do not establish arbitrary
natural-language accuracy or causal speedup. First-shot correctness is not acceptance.

This option is now merged into native main through
[PR 1125](https://github.com/kimjooyoon/meta-ontology-go/pull/1125). A fresh
48-call main pilot preserves the same bounded partial outcomes and call reduction,
with original UNKNOWN post-push CI context retained. Its
[public HF appendix](https://huggingface.co/asketeddy/gooo-feedback-path-tiny-v1/tree/5e5718256beb43667483154cc0e4464981a66fb4/research/unfixed-native-main-pilot-20261001)
includes all raw observations and actual emitted-Go values. FP32/PTQ main medians
increase slightly despite fewer calls; all median RSS values rise. Main push and
full evidence-source CI verification remain separate records.

## Retained native construction and finite completeness

The [native retained-worker study](docs/retained-native-study.md) measures 180
constructions over six reused bilingual views, using disconnected and four frozen
own-model arms. The worker loads its optional model once, emits each finished
request immediately and keeps fresh source-bound sessions per request. The same
code tree is adopted in native main through
[PR 1127](https://github.com/kimjooyoon/meta-ontology-go/pull/1127). Feature measurements,
the six-check promotion proof, clean main build and subsequent push-CI observation
are separate in the [source receipt](publication/retained-native-source-publication-20261001.json).

[Finite completeness](docs/finite-construction-completeness.md) separates 87.5%
declared-case completion, 100% attainment of the independently enumerated four-path
maximum and the remaining 12.5% finite contradiction gap. Attempt-weighted progress
also remains visible. All arms, including disconnected, reach the same final score;
this cohort establishes no model-specific functional improvement. The
[public HF appendix](https://huggingface.co/asketeddy/gooo-feedback-path-tiny-v1/tree/4f0d2c02719b48546c29f9a72a8ef13f34c11159/research/retained-native-completeness-20261001)
preserves raw evidence and original UNKNOWN hints. First-shot correctness is not
acceptance, and repeated views are not new independent experiments.

## Candidate budgets and bilingual partial construction

The [main budget study](docs/native-budget-study.md) now executes 1,944 real
constructions with candidate budgets 1/2/4, including initial-only and feedback
arms. At budget two, existing new FP32 reaches the restricted 93.75% mixed-contract
maximum and all observed separate-input cases; disconnected reaches 58.33% and
60.58% respectively. These are reused views over twelve existing intention groups,
with sixteen execution inputs, not a new language-generalization benchmark.

Matched feedback helps one QAT Korean intention repeated across three contracts
and hurts one parent English complete-contract view. Eighty of 972 bilingual
pairs emit different bodies. Curves, failures, model calls, CPU/RSS, startup
outliers and all raw observations are retained in the
[public HF appendix](https://huggingface.co/asketeddy/gooo-feedback-path-tiny-v1/tree/fb09079ece12f9bee821730b5e0514016098fd77/research/native-budget-main-20261001).
An initial model hint plus bounded deterministic tests is useful here; feedback
and any checkpoint promotion need evidence of marginal benefit.

## Distinguishing passed tests from resolved paths

The [finite diagnosis study](docs/path-diagnosis.md) performs 144 two-candidate
SDK searches followed by 144 deterministic diagnoses, comparing 576 candidates.
Own FP32 uses 144 initial predictions; diagnosis adds zero predictions. Twelve
actual emitted-Go executions evaluate 372 function inputs against an independent
state oracle. All 24 sparse FP32 views pass their tests but remain case-ambiguous;
twenty have a supplied probe where an alternative differs. Four retain unresolved
bounded probe agreement. Witnesses contain two observed candidate outputs and no
invented expected answer. This is reused development evidence over twelve existing
intention groups, and does not itself repair a program or raise accuracy.

The additional SDK diagnosis median is about 0.089–0.093 ms, excluding model load
and plan preparation. The new
[Go SDK v0.2.8 release](https://github.com/kimjooyoon/gooo-decision-runtime/releases/tag/v0.2.8-experimental)
exposes bounded, cancellable `PreparedPlan.Diagnose`; native main adoption is
recorded below. [HF raw evidence](https://huggingface.co/asketeddy/gooo-feedback-path-tiny-v1/tree/f41547f98ffc76d015bc027826afa21d619b1904/research/path-diagnosis-sdk-20261001)
includes all captures, preregistration and actual Go values. Earlier model weights,
card and regression evidence remain available.

## Native compiler diagnostic connection

The [native integration smoke](docs/native-path-diagnosis-smoke.md) uses the
published own FP32 model directly in the Gooo compiler: eight invocations,
four initial predictions, four deterministic diagnoses and zero extra diagnostic
predictions. All four diagnosis-off/on pairs retain the selected body. Two
actually emitted Go programs were compiled/run for four function evaluations.
This reuses one synthetic intention; the English view selects the opposite
subtraction order while passing its sparse test. The negative result and
input-three witnesses are preserved.

Diagnostic stage observations range from 0.034 to 0.053 ms. Whole-child wall
observations include the retained first disconnected startup outlier of 465 ms;
these eight calls do not establish a stable speedup or host CPU utilization.
[The public HF appendix](https://huggingface.co/asketeddy/gooo-feedback-path-tiny-v1/tree/7397092ea711e5b3f8a4ddba7f304a33647d2bdb/research/native-path-diagnosis-feature-20261001)
contains raw receipts, resource sidecars, selected emitted-Go values and the
unchanged preregistration. This captures clean feature source; main adoption
and canonical CI proof are recorded separately.

[Main adoption PR 1129](https://github.com/kimjooyoon/meta-ontology-go/pull/1129)
is merged after six successful canonical checks and independently verified
promotion proof. Clean main compiler/worker builds use Go 1.27.1 and SDK 0.2.8.
[Eight repeated main smoke calls](runs/native-path-diagnosis-main-20261001/report.json)
preserve all four off/on pairs and the negative English choice. Four additional
initial predictions and zero diagnostic predictions produce four actual
selected-Go function evaluations. These repetitions are not new independent
intention groups. Main diagnostic stages range from about 0.035 to 0.075 ms.

The [HF main adoption appendix](https://huggingface.co/asketeddy/gooo-feedback-path-tiny-v1/tree/07f3e8bc8359a88f3e89d92ee38b96b346d1ab32/research/native-path-diagnosis-main-adoption-20261001)
contains metadata and summary; full raw main captures stay on GitHub. Main push
CI was a separate pending observation in the frozen adoption receipt.
The [later terminal main-push receipt](publication/native-path-diagnosis-main-push-finalization-20261001.json)
records all six canonical checks and the independently verified proof as PASS.
The scoped
automatic promotion App remains unconfigured; ordinary authenticated CLI
promotion completed without overrides or human approval requests. Existing
weights remain unchanged. The [next experiment plan](docs/path-diagnosis-followup-plan.md)
prioritizes bounded judgment, partial completion, provenance and iteration cost.

## Continued bilingual Gooo judgment

The [paired-judgment study](docs/bilingual-judgment-results.md) performs 224 actual
MPS optimizer steps in control and bilingual-consistency arms, retains all six
exports and their negative comparisons, and makes 28 actual adopted-main compiler
dogfood calls. The paired term did not improve the control. Sparse passing paths
and language agreement can both be wrong. Explicit full contracts allow bounded
two-path continuation to complete the measured finite cases; this is not universal
intent correctness. Prediction medians are about 10–12 microseconds, while decoded
ternary tensors occupy 12,896 bytes. The new models remain experimental. Raw
captures, independent emitted-Go execution and source/test/model digests are public.
The [HF bilingual judgment appendix](https://huggingface.co/asketeddy/gooo-feedback-path-tiny-v1/tree/e930f1e3f6041477db166f8cb74ea74e6ea12937/research/bilingual-gooo-judgment-20261001)
contains six experimental exports and selected evidence. Anonymous immutable
byte verification passes for all 33 appendix files and 25 selected core files.
The existing core weights and card remain unchanged.

## Separate structure and bilingual intent inputs

The [fixed-channel study](docs/split-context-judgment-results.md) publishes a
matched positioned-v1 / split-v2 comparison: 560 actual MPS optimizer steps,
six own-model exports, zero new independent intentions. Every new split variant
requires more continuation attempts in this capture. FP32 adds 103 attempts
versus 69 in v1 across 320 views. Full authored contracts still complete all
3,840 finite cases per cell with zero extra predictions in the two-path Go
evaluation. Greater language agreement alone can mean the same wrong body.

Raw captures and numerical parity are retained. One actual compiler probe of v2
is rejected by the current SDK before inference; this is compatibility evidence,
not successful native adoption. The [next controlled studies](docs/split-context-followup-plan.md)
prioritize bounded judgment, typed source context and measurable continuation
cost. Runtime and publication are in Go; training/export are offline.
The [immutable HF split-context appendix](https://huggingface.co/asketeddy/gooo-feedback-path-tiny-v1/tree/ad979c4db936cebaeb996acdd9f48b9b4ff135e3/research/split-context-gooo-judgment-20261001)
contains all six exports and selected evidence. Anonymous byte verification
passes for 30 appendix files and 25 selected core files. The
[finite completeness and attempt metrics](publication/split-context-descriptive-metrics-20261001.json)
retain their explicit case/view denominators. CI preserves its own fresh parity,
judgment and packaging reports as downloadable artifacts.

The [own-model SDK context dogfood](docs/own-model-sdk-context-results.md) uses
[SDK v0.2.9](https://github.com/kimjooyoon/gooo-decision-runtime/releases/tag/v0.2.9-experimental)
to serialize verified typed-plan facts and actual Korean/English intents, then
rank legal paths with the explicit own model. Sixteen SDK calls make twelve
predictions; full contracts complete eight bodies with three additional attempts.
Three sparse wrong-intent choices remain in the raw captures. This is a separate
Go SDK assembly stage; native main adoption and original-source context binding
remain pending. CI replays these cells with independently retained captures.
The [immutable HF SDK context appendix](https://huggingface.co/asketeddy/gooo-feedback-path-tiny-v1/tree/ddedb067b1fd366a1d3f80767e2e738ecbf8507a/research/own-model-sdk-context-20261001)
preserves 25 archived inputs/evidence records and readable methods. Anonymous
verification passes for 11 appendix files, 30 retained split-model files and
25 selected core files. All five publication-source CI jobs pass in the
[finalization receipt](publication/own-model-sdk-context-publication-finalization-20261001.json).
