# Gooo neural decision experiments

We are developing small local models that help **Gooo assemble programs from
declared choices**. The compiler provides the plan and checks the result; the
model suggests which permitted construction to try next. Failed examples become
context for another bounded attempt. This repository holds the experiments,
training tools and original evidence behind that process.

## Try a local construction

From this repository's root, use Go1.27.1 and put Go's executable install
directory in PATH. This pins the compiler to the source used in the latest
installed study; the small model is already included here.

```sh
GOTOOLCHAIN=go1.27.1 go install github.com/kimjooyoon/meta-ontology-go/cmd/gooo@01d21e9260b2c0f1d02f8029824f3dda41631e9d
GOTOOLCHAIN=go1.27.1 go run ./examples/file-body-run examples/whole-candidate-order file-body-inputs
gooo body-path-run --source file-body-inputs/ko-source.gooo \
  --activity AssembleKorean --path-plan file-body-inputs/ko-recipe.json \
  --cases file-body-inputs/cases-128.json \
  --model publication/order-judge-initial-20261003/model.json \
  --repeat 2 --timing --out korean-body-results
gooo body-path-run --verify-timing --out korean-body-results
```

Use fresh input/output directory names. Omit `--model` for deterministic
construction. The example declares a pure `Integer -> Integer` body and 128
caller expectations. Read its generated Go and actual runtime outputs in the
result directory. `unobserved` means no usable output was measured; `0/128`
means measured outputs missed every expectation. [Full usage and next actions](https://github.com/kimjooyoon/meta-ontology-go/wiki/File-Based-Body-Run).

## Start here — 2026-10-04

- [Avoid waiting before input-file validation](publication/file-input-open-20261004):
  an actual source-path swap held the installed CLI before output creation until
  its sole FIFO writer was opened; metadata and weights reproduced the same wait.
  Current candidate `0eb69e5f` shares the nonblocking opener across these boundaries.
  Its 64 swaps have zero writer releases/timeouts; 25 regular completions keep
  earlier Go and 3,200/3,200 observations, while 39 file errors occur before construction.
  Two model FIFO requests return without a writer. Separate own-model/deterministic
  smoke keeps 1,024/1,024. Initial candidate observations remain archived separately.
  Dev #1202 is in its new-head CI; installation remains `01d21e92`.

- [Distinguish unobserved expectations from actual measured zero](publication/finite-observation-display-20261004):
  installed terminal summaries use current case observations and replay state.
  Eight ordinary constructions/four own-model judgments/16 runs keep 1,024/1,024
  and all frozen Go. Five separate CLI controls retain three original errors and
  two deliberately changed expectation sets, preserving actual 0/256 observations.
  Nine synthetic display states are counted separately from native execution.
  Dev #1200 and main #1201 passed canonical CI and independent source proofs,
  then merged. Clean main `01d21e92` is installed. Fresh model warm responses
  were 29.64/24.50ms; the original first responses are also retained. The small
  variable/condition quickstart separately keeps 6/6. Saved receipts distinguish
  UNKNOWN from actual observed zero and retain a null aggregate score.

- [Handle a stalled executable FIFO and read failed expectations](publication/nonregular-tool-20261004):
  Unix opens without waiting for a FIFO writer and validates the opened file kind.
  The candidate rejected an actual FIFO in 604.14ms total CLI time, with no
  native/model work and 128 unobserved expectations. Original interrupted blocking
  observation, failing tests and compile failure are retained. Same-window ABBA
  controls kept 12,288/12,288 expectations and all frozen Go in 96 constructions,
  48 judgments and 192 runs; warm medians 21.54/22.07ms overlap. Dev #1198 passed
  canonical checks and an independent source proof, then merged. Main #1199 also
  passed its own source-bound proof/promotion checks and merged. That study
  installed main `93fa2742`: eight successes/four predictions/16 runs kept
  1,024/1,024; the installed FIFO failed promptly in 19.665ms total CLI time.
  The direct variable/condition quickstart additionally kept 6/6 over two
  deterministic constructions/four native runs. Original failures remain public.

- [Reduce repeated hash allocations and read missing timing stages](publication/owned-hash-reader-20261004):
  a lazily owned 32 KiB buffer, released on close/cancellation, retains current
  whole-file hashes and both native runs. The warm hash benchmark allocated
  about 33.5→0.8kB/call. Fresh old/new ABBA construction made 96 generations,
  48 own-model judgments and 192 runs: 12,288/12,288 finite expectations and
  all frozen generated sources match. Warm responses were 64.13→65.88ms;
  these overlapping one-device observations establish no speed improvement.
  Missing stderr phases now read `unobserved`. Dev #1196/main #1197 passed their
  own canonical CI and independent proofs, then merged. At that stage, clean main `93c6463d`
  was installed: eight fresh successes, four predictions, 16 runs and 1,024/1,024
  finite expectations; a separate missing-tool failure retains 128 unobserved
  expectations and zero runs. Weights, training and intent scope are unchanged.

- [Read body construction, validation, native execution and saving costs](publication/body-wall-phases-20261003):
  optional compiler phase sidecars and read-only file-binding checks. Two clean
  candidate revisions ran 192 generations, 96 actual tiny-model judgments and
  384 native runs on the same two known KO/EN fixtures, keeping all 24,576 finite
  expectations and generated bytes. Final warm plain/timed medians were
  64.77/64.71ms; whole Go-file hashing took 14.40ms. These observations identify
  costs without establishing a recorder speedup. Original CI modernizer failure
  and source revisions are retained; weights and intent scope are unchanged.
  Dev #1194/main #1195 passed their own six canonical checks and independent
  exact-source proofs, then merged. At that stage, clean main `041c8bbf` was installed; eight fresh
  generations, four predictions and 16 runs kept 1,024/1,024 finite expectations.
- [Reproduce historical model results and retain the original platform differences](publication/legacy-platform-replay-20261003):
  new macOS/Linux audits each reproduce all 18,432 development rows and 24 compact
  files from their fixed platform archives. The original 101 summary differences
  and 272 changed complete rankings remain in the Linux record. Both the legacy
  replay and the separate explicit-arithmetic comparison pass; all 19 research
  CI jobs passed before PR #1 merged as `87c6f219`. Weights and expectations are
  unchanged. The reader makes zero predictions; each audit records 25,824 runtime
  predictions on known inputs. Original comparator and local failures are public.
  [Anonymous Hugging Face readback](publication/legacy-platform-hf-readback-20261003.json)
  verifies all 24 appendix files, both model cards and unchanged example weights.
- [Reuse the first native Go version check during repeated body execution](publication/toolchain-version-reuse-20261003):
  unchanged source replay, current Go-file hashes and two native runs per request.
  Paired macOS observations on the same two known EN/KO fixtures: warm response
  median 79.91→63.69ms, current-child CPU 31.21→13.37ms; 96 generations,
  48 own-model predictions, 192 native runs and 12,288/12,288 finite expectations.
  Original check/output stays in runtime v3; old v2 comparisons remain identical.
  Model weights and the number of distinct intent tasks are unchanged.
  Dev #1192/main #1193 passed canonical CI and independent immutable proofs,
  then merged. At that stage, clean main `ed2c2cac` was installed; fresh replay made eight
  generations/four predictions/16 runs with 1,024/1,024 expectations unchanged.
- [Replay saved IR baselines with Go](publication/go-baseline-replay-20261003):
  an explicit source-version repair preserves the original freeze and current
  Go1.27.1 scripts. A stdlib Go tool binds all 142 frozen inputs, then replays
  52 saved modules and 96 candidate packages, including the three expected
  compile failures. Source hashes, original failures and unknown denominators
  are retained. Local and Linux replay pass; public PR #19 is merged.
- [Fresh candidate and feedback replay in Go](publication/go-revision2-replay-20261003):
  93 original and 96 repaired candidate packages execute in temporary copies;
  all 189 fresh result files match frozen evidence. The same 32 intentions,
  96 provider plans and 32 feedback sets are checked with zero model calls.
- [Write body statements directly on multiple lines](publication/raw-body-usability-20261003):
  raw backtick literals preserve decoded text, line endings and source diagnostics.
  Local syntax/body-codegen race and whole-module vet pass. Eight generations,
  four actual own-model predictions and 16 native runs retain 1,024/1,024 finite
  expectations and all eight original generated sources. Dev #1190/main #1191
  passed canonical CI and independent proofs, then merged. Clean main `eb8477d5`
  is installed and freshly reproduces 1,024/1,024 with four predictions/16 runs.
  Separate raw clamp cases pass 8/8; malformed UTF-8 is rejected. The new warm
  responses are 77–94ms and retain the slowdown from the earlier 28–33ms sample.
- [Read owned runtime records in completeness comparisons](publication/owned-runtime-comparison-20261003):
  dev #1188/main #1189 passed their own six canonical checks and independently
  verified immutable proofs, and clean main `05746e4a` is installed. New own-model
  generation/execution passes 256/256 finite expectations; conditional/variable
  assembly passes 6/6. Eighteen earlier installed-record comparisons match the
  candidate outputs byte-for-byte. Changed resource units/denominators retain
  both observations without a numerical improvement. All initial failures remain.
- [Use source/recipe/case files directly](publication/body-path-file-cli-20261003):
  experimental `gooo body-path-run` saves generated Go and current native
  observations without constructing input JSON lines. Named EN/KO activities,
  eight generations, four actual model calls, 16 native runs and 1,024/1,024
  finite expectations on 128 integer inputs. Sparse-example failures and a
  Gooo-oracle candidate resolution are preserved separately. Dev #1186 passed
  canonical CI and independent proof and is merged. Main #1187 also passed,
  merged and is locally installed as `fc854ef4`; all eight bodies and 1,024/1,024
  finite expectations match. Conditional assembly and original unmet outcomes
  remain preserved. Warm response including execution: 29–33ms on this device.
- [Keep one verified executable while running current inputs](publication/retained-native-execution-20261003):
  512 fresh generations/predictions and 1,024 native runs retain every original
  outcome, including partial failures. Warm execution median 294→29ms, current
  child CPU 215→9.8ms; 128/128 paired calls faster on macOS. `--retain-native` in
  the Go example uses immediate stream execution; omitting the model stays
  deterministic. Full records, first failed launch and exact measurement scope
  are public. Dev #1184/main #1185 passed canonical CI and independent proofs,
  merged, and main `8c6ec01c` is installed. Installation replay: four generations,
  eight native runs and 32/32 cases; repeat responses including execution 30ms.
- [Recipe projection reuse is merged and installed](publication/recipe-projection-installed-main-20261003):
  main `3bee55da`, SDK v0.2.20. Existing English/Korean tasks, integrated stream
  and the one-command example complete 24 generations, 12 model calls,
  48 compiled executions and 192/192 finite expectations with unchanged weights.
- [Generate and execute the Korean example with one Go command](examples/whole-candidate-order):
  `go run ./cmd/order-example --model ... --out ...` reads source, recipe and
  finite cases, executes each body immediately and saves complete receipts.
  Omit the model for deterministic construction. Sequential open-input requests
  expose preparation reuse and separate response from native execution time.
  [Local and Linux receipts](publication/order-example-immediate-20261003):
  each collection completes four generations, eight native runs and 32/32 cases.
- [Reuse a checked source projection inside recipe expansion](publication/source-recipe-projection-reuse-20261003):
  512 generations and 1,024 native executions retain original outcomes. Paired
  decoder measurement: 8,192 decodes, all document/plan pairs match, median
  0.371→0.292 ms and allocated bytes 729→572 KB. Includes the prior timing
  comparison with no improvement and all 751 slower paired observations.
- [Repeated construction is merged and installed](publication/body-stream-installed-main-20261003):
  `gooo body-path-stream` accepts source recipes as JSON lines and emits each
  result when ready. Model omission is deterministic. Local dogfooding: four
  generated bodies, eight native executions, 32/32 expectations. Compiler
  [PR #1180](https://github.com/kimjooyoon/meta-ontology-go/pull/1180) and
  [main #1181](https://github.com/kimjooyoon/meta-ontology-go/pull/1181) are merged.
  Clean installed main `cb2892cb` reproduces all four generations and 32 cases.
- [Prepared candidates are merged and installed](publication/order-prepared-installed-main-20261003):
  SDK v0.2.20, main `8117fbae`, 16 generated bodies and 32 native executions,
  128/128 expectations. The [512-generation API comparison](publication/order-prepared-native-20261003)
  measures second-call generation medians of 1.017 → 0.693 ms with unchanged weights.
- [Existing retained-worker observations](publication/prepared-worker-native-20261003):
  96 generations, 48 predictions, 192 native executions and 768/768 expectations.
  Sequential repeats reuse preparation; alternating plans expose the one-plan
  cache's limits. Full sources, measurements and a two-request example are public.
- [Runnable Korean order-assembly example](examples/whole-candidate-order):
  source, short recipe, eight execution cases and two CLI commands. The model
  selects actual operations for `input * 2 + 1` inside Gooo code generation.
- [The new 16 KiB judge now runs inside Gooo codegen](publication/order-judge-native-20261003):
  256 generations, 128 direct model predictions, 512 immediately compiled runs.
  On 64 observed development requests, first-attempt completion is 32 → 52;
  budget-8 completion is 64/64 in both routes, with 96 → 76 body evaluations.
  Generation wall median rises from 8.58 to 9.26 ms. All partial outcomes and
  CPU/memory costs are published; further training updates are zero.
  [Released Go SDK and full Linux/arm64 replay](publication/order-judge-sdk-20261003).
- [Whole-candidate tiny judge: 16 KiB, 64 training requests](publication/order-judge-initial-20261003):
  first finite completion 48/96 → 82/96 on authored development contrasts.
  Real search plus exact equal-body reuse uses 110 versus 144 body evaluations;
  complete search latency remains slightly higher. Includes weights, sources,
  all failures and 480 follow-up model calls. The native integration follow-up
  above retains these initial weights and observations.
  [Hugging Face model](https://huggingface.co/asketeddy/gooo-order-judge-tiny-v1) ·
  [Linux replay and full record comparison](publication/order-judge-linux-20261003).
- [Small recipes derived from Gooo source](publication/source-recipes-20261003):
  24 generations and 48 compiled runs; 12 recipe/full-document pairs match.
  Authored JSON shrinks by 63%. Oracle arms meet 72/72 runtime expectations;
  single-example controls meet 12/72. Added source expansion has measurable cost.
  [한국어 설명](docs/source-recipes-20261003.ko.md).
- [A 128-byte clue about instruction order](publication/intent-order-preflight-20261003):
  32 controlled pairs have identical old-model predictions; directed clause
  counters distinguish them and retain a repeated-operation counterexample.
  This is an untrained feature experiment with 64 real frozen-model calls.
- [Direct projection after a unique observation](publication/path-observation-resolution-20261003):
  120 generations and 240 compiled runs; resolved arms skip model/search and meet
  144/144 finite expectations. Model-requested codegen median is 8.27 ms versus
  9.49 ms for cached observations followed by search, on one authored task.
- [Candidate-output reuse in the compiler](publication/path-observation-reuse-20261003):
  96 generations with 120 actual tiny-model predictions; probe evaluations fall
  from 33 to 14 per oracle request. Full codegen latency still needs improvement.
- [최근 연구와 다음 작은 언어 기능](docs/agile-language-research-20261003.ko.md):
  Laya, LAVOIR, OpenAI, Meta and current community discussions; a bounded Go
  implementation that proposes the next useful execution input without training.

- [Gooo Wiki, 한국어](https://github.com/kimjooyoon/meta-ontology-go/wiki):
  an introduction, language/model walkthrough, metric definitions and research connections.
- [Language and compiler](https://github.com/kimjooyoon/meta-ontology-go): source,
  semantic IR, typed body generation, native execution and completeness receipts.
- [Project direction, 한국어](https://github.com/kimjooyoon/meta-ontology-go/blob/dev/docs/language-direction.ko.md):
  the workshop metaphor, current capabilities, intended differences and next work.
- [Go inference runtime](https://github.com/kimjooyoon/gooo-decision-runtime):
  fixed tensor arrays, caller workspaces, path ranking and finite feedback search.
- [Current whole-candidate Hugging Face model](https://huggingface.co/asketeddy/gooo-order-judge-tiny-v1):
  independently initialized 4,096-parameter judge, Go inference and evidence.
- [Earlier shared judges](https://huggingface.co/asketeddy/gooo-shared-judgment-tiny-v1):
  FP32/PTQ/QAT weights and their separate contracts and measurements.
- [Current model card source](publication/order-judge-model-card-20261003.md):
  model usage, full comparison table, lineage and research acknowledgments.

The current whole-candidate judge has **4,096 trainable parameters** and 16 KiB
of FP32 weights. Earlier shared judges use 2,072 parameters across three
decisions. Gooo supplies eight complete candidate paths.
The research asks whether this division of work can reduce search cost while
preserving source meaning and making incomplete behavior easy to inspect.

### Previous shared-judge training: keep the complete instruction

The [four-arm full-input comparison](docs/full-input-judgment-initial-results-20261003.md)
completed **6,400 local MPS updates** and retained all twelve FP32/PTQ/QAT exports.
On the original 512 development inputs, first-path finite completeness was
113 for positioned-original FP32, 266 for positioned-varied, 368 for
bag-original and 320 for bag-varied. Bag-original reduced extra ranked attempts
from 1,469 to 186. Its representation also aliases a documented operation-order
counterexample. Wording augmentation and ternary conversion have mixed results;
the complete tables retain these regressions.

Go independently reconciled every update, replayed 624 export vectors and 624
compact equivalents, and made 6,144 calibration plus 18,432 development
predictions. Training took 42.18 seconds, averaging 49.83% of one CPU core with
1.81 GB process peak RSS. The [complete public evidence](publication/full-input-initial-study-20261003/README.md)
contains all prepared features, full texts, models, optimizer journals and first
Go observations. The SDK replay and native observation below have since completed.
The remaining wording/split comparisons have a
[published protocol](docs/full-input-all-forms-protocol-20261003.md); its planned
calls are separate from completed observations. The cohort has been observed before.
All 33,792 inputs are now prepared and verified, with zero model calls. The
planned 811,008-call collection has not started; useful small language features
take priority under the updated research direction above.

[Linux portability follow-up](docs/full-input-numerical-portability-followup-20261003.md):
all 18,432 first selected masks match arm64, while 272 full candidate rankings
differ under small score deltas. The exact comparison failed on partial-completion
curves. The [explicit-arithmetic follow-up](docs/full-input-separate-arithmetic-results-20261003.md)
now pairs 147,456 actual predictions with unchanged weights: all 18,432 input
pairs match exactly in intermediate values, scores and candidate order under the
new rule. First-path completion and extra attempts stay unchanged. The original
failure remains reproducible. The matching SDK is published and the native stage
below has completed generation and execution observations.

### From research code to the public Go SDK

[SDK v0.2.15-experimental](https://github.com/kimjooyoon/gooo-decision-runtime/releases/tag/v0.2.15-experimental)
adds V4 input features and explicit arithmetic. At its release source `59c8d34`,
local arm64 and Linux CI each processed 18,432 complete inputs with 36,864 actual
predictions. Both matched the frozen intermediate values, probabilities and
full candidate rankings. [Complete SDK reports](publication/full-input-sdk-replay-20261003/README.md)
retain the two observations and their source/artifact identities.

### Completed native integration

The [registered native stage](docs/full-input-sdk-native-protocol-20261003.md)
has completed **400 generations, 816 local predictions and 800 compiled runs**
with the new contracts. All 9,600 supplied expectations matched; all 192
expanded/compact pairs retained generated Go, search/feedback semantics and
ordered outputs. Each generation immediately led to its build and two runs.
An independent reader checked 1,760 progress and 432 feedback records.

On these sixteen known requests, bag-original FP32 completed its first path in
16/16 cases; its broader development result remains 368/512. Its compact median
prediction was 8.33 µs, whole codegen 10.00 ms and compiler peak RSS 17.78 MiB.
The disconnected arm completed all expectations with zero predictions and 96
extra candidates; its codegen median was 9.80 ms. Model selection quality,
search cost and process latency each retain their own measurement.
[Complete results and resource scope](docs/full-input-native-results-20261003.md)
and [all original evidence](publication/full-input-native-20261003/README.md).

The measured compiler source is `e461c1d`, using SDK v0.2.15. The integration merged
to dev as `75b2b7d` in [PR 1158](https://github.com/kimjooyoon/meta-ontology-go/pull/1158).
Main promotion completed as `fc0e99c4` in [PR 1159](https://github.com/kimjooyoon/meta-ontology-go/pull/1159).
This observation made zero training updates and
kept the existing model default. All 400 receipts retain `permission_boundary`
as their first unresolved dimension.

### Motivating diagnosis: input phrasing changes the chosen path

The [bilingual wrapper audit](docs/bilingual-wrapper-audit-results-20261003.md)
holds six published models, Gooo sources and finite targets fixed while varying
five authored input forms. It made **92,160 real Go predictions**; an independent
reader replayed every prediction with zero numerical difference locally and
recomputed all 30 condition summaries. Shared FP32 completes 113/512 original
development views and 480/512 after removing the known curriculum prefix.
The full text of both forms remains in the evidence. This is a format-sensitivity
diagnosis; the published original-input quality measurement stays 113/512.
The full-input comparison above follows this diagnosis.
[Closed diagnostic evidence bundle](publication/bilingual-wrapper-audit-20261003/README.md).

The [comparison protocol](docs/full-input-judgment-preregistration-20261003.md)
froze four freshly initialized 2,072-parameter judges: positioned or
whole-text fragment features, each trained with original or varied wording.
Go reconstructs complete inputs and rejected forms, and independently replays
their features and weights before local MPS optimization. The plan fixes 6,400
optimizer updates and retains all twelve FP32/PTQ/QAT exports. A negative control
shows how fragment counts can alias two different operation orders; it remains
part of the evidence even if average quality improves. The protocol/source
publication preceded optimization; the new observations are linked above.

[Independent Linux replay](https://github.com/kimjooyoon/gooo-neural-decision-experiments/actions/runs/37049184049)
reproduced all 92,160 selected paths and all 30 condition summaries within the
declared numerical tolerance. The same run's 13 CI jobs passed. This replay has
its own prediction count, separate from the original collection.

### What has improved, and what needs work

| Question | Latest observation | Next measurement |
| --- | --- | --- |
| Does the new representation help path selection? | Original-input FP32 completion rose 113→368 out of 512; extra ranked attempts fell 1,469→186 | Remaining forms/splits, new tasks and the retained operation-order alias |
| Do Korean/English intentions lead to valid behavior? | Bag-original FP32 chose valid paths in both languages for 180/256 pairs; 68 pairs chose the same wrong path | Both-valid agreement across wording, task families and newly authored intentions |
| Does compact storage preserve behavior? | Explicit arithmetic matches 18,432 cross-platform rankings; all 192 native expanded/compact pairs match | Broader inputs and operation-order representation |
| Can generated code run? | New V3/V4 models passed 9,600 supplied expectations across 800 compiled runs | Broader types and workflows |

Model quality, representation parity, finite execution and resource usage have
separate denominators. The current development cohort has been observed in
earlier studies. Whole-domain completeness and bilingual meaning preservation
remain open research questions. Original failures, slow samples and regressions
stay available in the dated reports below.

### How this should develop the language

Gooo's declarations are the assembly plan; a tiny judge helps choose parts within
that plan. The language work is to express more useful plans and preserve their
meaning through generation, feedback and reuse. Conditions, assignments,
references and ordered branches provide the current starting point. Larger typed
constructions, natural-language discovery of plans, and reusable learned
abstractions form the next research direction.

We track progress as declared behavior completed, extra construction attempts,
source/result traceability, unresolved obligations, latency and memory. A first
path is complete here when all 16 supplied expectations pass; partial coverage
and later completion remain separate measurements. Public evidence currently
compares Gooo variants on authored tasks. Comparisons with other language systems
need a shared task set, tool budget and definition of completion.

For actual compiler use, select the pinned V3 compact bundle described in the
[integration guide](https://github.com/kimjooyoon/meta-ontology-go/blob/main/docs/three-choice-path-model.md).
The V4 full-input models run in this repository's research runtime, SDK v0.2.15,
and the compiler integration revision linked above.

### Research connections

We appreciate [SKETCH](https://people.csail.mit.edu/asolar/papers/asplos06-final.pdf)
for specification-guided completion,
[DeepCoder](https://arxiv.org/abs/1611.01989) for learned guidance of program search,
[DreamCoder](https://arxiv.org/abs/2006.08381)
for neural search and reusable program abstractions,
[Laya](https://huggingface.co/convaiinnovations/laya) for the structured-decision
interface used in our early experiments,
[BitNet b1.58](https://arxiv.org/abs/2402.17764) for the ternary-weight research
direction, and [W3C PROV-O](https://www.w3.org/TR/prov-o/) for provenance vocabulary.
Our experiments combine these ideas with Gooo-owned declarations, finite tests,
deterministic continuation and small Go inference kernels.

## Evidence log

The following sections preserve individual experiments and release histories.
Their measurements belong to the named datasets, model files and source revisions.

### Compiler observation deltas

The [actual comparator dogfood](publication/completeness-delta-dogfood-20261003/README.md)
uses the published compact FP32 judge for one fresh generation, three predictions
and two compiled runs, matching 24 supplied expectations. The new compiler
comparison identifies two UNKNOWN-to-observed transitions, eight new axes and
the preserved permission frontier. All original receipts and three deterministic
comparison outputs are included. This integration sample has its own scope,
separate from the quality and timing studies below.

## Compact models in actual Gooo codegen

[The complete native appendix](publication/compact-shared-native-20261003/README.md)
records 96 generations, 298 actual model predictions and 192 compiled runs.
All 2,304 finite expectations passed; all 48 expanded/compact pairs retained
equal unseeded generated source, path search, feedback semantics and ordered
outputs. An independent compiler consumer verified 596 progress records and
202 feedback records. Every receipt still identifies the unresolved permission
boundary; this is neither a new holdout nor whole-language completeness.

Compact prediction medians were 23.7–30.2 µs, while fresh-process codegen medians
were 27.8–29.5 ms. QAT's complete codegen median increased slightly, and cold
cache outliers remain in the report. Process CPU medians were 83.7–84.6% of one
core. The full 684-member evidence archive occupies about 3.04 MB compressed.
CI verifies the complete inventory and independently consumes the frozen
runtime receipts against the exact compiler source.
[Hugging Face native appendix](https://huggingface.co/asketeddy/gooo-shared-judgment-tiny-v1/tree/37db3a1f8a06669081a370ee1e7b23a64cbc00fb/research/compact-native-20261003)
is public; [anonymous transport verification](publication/compact-shared-native-hf-verification-20261003.json)
rechecked all eight new/updated files and all 684 archived members.

## Shared judge in compact Go storage

[Public compact own models](https://huggingface.co/asketeddy/gooo-shared-judgment-tiny-v1/tree/985999a89caba6a31cc7147f66ba29a5ce76a1d9/research/compact-runtime-20261003)
and [Go SDK v0.2.14-experimental](https://github.com/kimjooyoon/gooo-decision-runtime/releases/tag/v0.2.14-experimental)
are available. Anonymous immutable downloads verified all ten appendix files,
the updated model card and the complete decoded comparison ledger.
[Transport evidence](publication/compact-shared-hf-verification-20261003.json)
is separate from the numerical/runtime observations below.

The [preregistered representation study](preexecution/shared-three-compact-runtime-preregistration-20261003.json)
now exports an explicit shared-weight ABI for the existing own models. No new
optimizer updates or quality selection were performed. All 10,739 frozen
initial/feedback inputs match bit-for-bit for all three variants: 64,434 actual
parity predictions, including features, hidden values, logits, probabilities and
selected mask. The complete [comparison ledger](publication/compact-shared-runtime-20261003/comparisons.jsonl.gz)
is compressed losslessly; the [manifest](publication/compact-shared-runtime-20261003/manifest.json)
pins every public file and its decoded ledger. CI reconstructs the original
source/teacher data and repeats every actual kernel comparison.

| Variant | Expanded → compact weight bytes | Expanded → compact resident tensors | Expanded → compact median kernel time |
| --- | ---: | ---: | ---: |
| FP32 | 74,624 → 8,288 | 74,624 → 8,288 | 40.833 → 24.458 µs |
| PTQ ternary | 3,854 → 446 | 18,752 → 2,096 | 40.333 → 25.125 µs |
| QAT ternary | 3,854 → 446 | 18,752 → 2,096 | 47.292 → 27.667 µs |

Ternary models additionally retain eight scale bytes. The per-caller workspace
remains 3,200 bytes; all valid warmed kernels allocate zero heap objects. Timing
uses 2,000 interleaved observations per representation/variant on local arm64,
Go 1.27.1, and includes feature projection. It excludes loading and native
codegen. The full auditor took 13.833 seconds and averaged 112.74% of one CPU
core including dataset reconstruction and hashing; its 185,712,640-byte peak
RSS describes the auditor, not the small inference model. See the
[full scoped report](publication/compact-shared-runtime-20261003/report.json).

The converter verifies every shared copy and removed positive zero, keeps the
original matrix scales, and preserves 24-term float accumulation order. Runtime
receipts identify the actual compact artifact. Seeds are still bound to artifact
hashes: repeated use of one artifact is reproducible, while a matching seed
across different representations need not select the same mask. Native compiler
adoption and paired source-to-runtime measurements are now recorded in the
compact native section above. The original expanded models and quality regressions
remain available.

## Shared local Gooo judge — fresh model comparison

[Public own models on Hugging Face](https://huggingface.co/asketeddy/gooo-shared-judgment-tiny-v1/tree/9ce1d57a0688c0e07dd0007bd7af558a5642cf4e)
include all six exports and the full evidence bundle. Anonymous retrieval
verified all 18 public files and all 1,116 archived members byte-for-byte.
[Transport verification](publication/shared-three-hf-verification-20261003.json)
is separate from the recorded model and native observations.

[Full results and limitations](docs/shared-three-judgment-results-20261003.md):
3,200 actual MPS updates, six exports, 288 Go parity checks, 3,072 development
predictions, and 96 immediate Gooo generation/runtime pairs. Shared FP32 raises
initial finite completeness from 95/512 to 113/512 and reduces static ranked
extra attempts from 1,572 to 1,469. Bilingual disagreement remains 255/256;
development confidence and some families regress. All negative variants remain.

The [public evidence](publication/shared-three-20261003/manifest.json) binds 1,116
members, including all journals and 2,304/2,304 passing native expectations.
CI verifies every archived byte and independently replays the six Go models.
This training study measured the expanded ABI. The later compact study above
records the reduced resident tensors. [Storage amendment](docs/shared-three-storage-amendment-20261003.md)
preserves the original zero-update failure and all older evidence.

## Three-choice Gooo construction — fresh students and actual Go kernels

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
The [native corpus audit](publication/own-three-choice-curriculum-audit-20261002.json)
now verifies all 3,072 actual exports, 9,216 complete individual inputs and
43,440,202 journal bytes. The first failed attempt remains retained.
The [actual teacher collection](docs/own-three-choice-teacher-results-20261002.md)
now retains 4,096 sessions, 37,394 real own-model predictions, 13,233 candidate
evaluations and 7,667 unique failure-conditioned train states. Its separate
offline audit reconstructs all outputs, seeded frontiers and complete contexts.
SDK session median/p95 are 0.229/0.577 ms, including search and evaluations.
The three-choice SDK comparison below is complete after a separate storage
continuation; the original native phase is now completed below. No default model is promoted.
Full source/teacher raw evidence is packaged with
closed SHA/CRC/privacy checks and retains the first failed native-export prefix.
The existing v2 models below retain their completed SDK/native study.

The [new matched training](docs/own-three-choice-training-results-20261002.md)
completed 4,800 local MPS updates from one fresh Go initializer and preserved all
nine FP/PTQ/QAT exports. All nine passed actual Go numerical parity and zero-heap
kernel probes. The complete 642-member compressed training evidence preserves
all features, update/epoch receipts and negative variants. Full SDK
behavior and actual native measurements are reported separately below; no
new default is promoted by the training/kernel phase.
The [public three-choice model edition](https://huggingface.co/asketeddy/gooo-three-choice-feedback-tiny-v1/tree/a7a9170370da3e183c1831ae4442e88870ea3a0d)
now retains all nine models and both complete raw-evidence bundles.
[Anonymous byte verification](publication/own-three-choice-hf-byte-verification-20261002.json)
confirms all 35 uploaded files at that immutable public revision.

The [actual SDK prefix](docs/own-three-choice-sdk-prefix-results-20261002.md)
retains 8,779 sessions, 33,388 real predictions and 510,256 independently verified
ordered values. All eleven calibration policies completed before a saved selector
chose `set-feedback/fp32`: 6.11% fewer extra candidates and 66.09% fewer predictions
than the frozen independent reference. The original 768 MiB raw cap then stopped
collection with six complete development cells and a partial seventh. All original
bytes remain retained; at that stop the full 11,264-session comparison and dependent
native execution were incomplete. Paired-language initial mask disagreement is 256/256,
so bilingual invariant judgment is not established. Prefix CPU/RSS are unavailable;
future failed attempts now retain terminal resource counters as well.
The [public stopped-prefix appendix](https://huggingface.co/asketeddy/gooo-three-choice-feedback-tiny-v1/tree/ba1f540ee5bf75fceab53d1446b9385c793a4396/research/sdk-prefix-20261002)
preserves all 25 original evidence members in a complete byte-verified ZIP.
[Anonymous appendix verification](publication/own-three-choice-sdk-prefix-hf-byte-verification-20261002.json)
checks all seven uploaded files at its immutable HF revision. The original
model tensors remain unchanged. The separate
[storage continuation plan](docs/own-three-choice-sdk-storage-continuation-preregistration-20261002.md)
fixes the exact 2,485 missing identities, preserves the original resource failure,
and requires complete combined audit before any dependent native execution.

The [actual missing-only continuation](docs/own-three-choice-sdk-storage-tail-results-20261002.md)
ran 2,485 additional development sessions with unchanged models and no original
operational reruns. The [combined independent audit](publication/own-three-choice-sdk-combined-audit-20261002.json)
now proves 11,264 unique sessions, 42,530 actual predictions and 658,992 ordered
values across all 22 complete cells. Selected FP32 uses 65.19% fewer development
model calls than the reference; first-candidate completion and median time do not
improve. PTQ/QAT add search work. The separate
[bilingual functional diagnosis](publication/own-three-choice-sdk-bilingual-functional-audit-20261002.json)
finds different initial outputs in all 256 selected-model development pairs;
mask differences and actual output differences are counted separately. Tail
collection measured 11.662 s, 137.55% of one CPU core and 297.7 MiB lifetime RSS,
including original audit and planning. Native validation was pending at the SDK boundary;
finite eight-mask completeness does not establish general-language judgment.
The [public missing-only appendix](https://huggingface.co/asketeddy/gooo-three-choice-feedback-tiny-v1/tree/0bc97aeb97c1d38b6f860f8f3ef88f05a981084c/research/sdk-storage-tail-20261002)
retains the full tail and negative bilingual findings with the existing model
weights unchanged. [Anonymous byte verification](publication/own-three-choice-sdk-storage-tail-hf-byte-verification-20261002.json)
matches all seven uploaded files at that immutable revision.
The [domain completeness receipt](publication/own-three-choice-domain-completeness-receipt-20261002.json)
records declaration, generation, reverse observation, use-case, boundary and
provenance evidence separately. That historical receipt preserves unknown
compiler-schema binding and native execution pending at that point.

The [actual native phase](docs/own-three-choice-native-execution-results-20261002.md)
now completes 640 real Gooo generations, 640 Go compile-and-run invocations and
10,240 ordered outputs with 1,760 actual model predictions. The
[zero-call independent audit](publication/own-three-choice-native-execution-audit-20261002.json)
binds every original input, source, typed path, SDK observation and generated-Go
output. Selected extra candidates fall 402→301 against disconnected enumeration
(-25.12%); four-candidate finite completion rises 56.25%→76.56%. First-candidate
completion remains weak, and selected initial outputs disagree in all 64/64
Korean/English pairs. Full-budget TDD completes every finite contract.

Selected codegen median is 12.922 ms, with internal native work at 2.322 ms,
recorded prediction mean 26.743 µs and median codegen child RSS 18.81 MiB.
Go compilation/run median is 443.224 ms. Disconnected model-load/context work
and predictions are zero. No total compiler speedup or host-utilization increase
is asserted. The [closed public raw bundle](publication/own-three-choice-native-execution-bundle-20261002.json)
preserves all 1,284 original raw files and supplementary audit/protocol/PROV-O
evidence. CI independently reconstructs all 640 pairs without new operational
calls. The [native domain-completeness delta](publication/own-three-choice-native-domain-completeness-delta-20261002.json)
records completed finite generation/reverse execution, preserved bilingual
negatives and the still-unknown compiler declaration/receipt-schema binding.
The [public native HF appendix](https://huggingface.co/asketeddy/gooo-three-choice-feedback-tiny-v1/tree/3151265f75aad40f6ce9f89a904bd34acdbade2c/research/native-execution-20261002)
preserves the full native raw archive and model-quality negatives. Its
[anonymous byte receipt](publication/own-three-choice-native-execution-hf-byte-verification-20261002.json)
matches all seven files and 7,600,065 public bytes at that immutable revision;
existing model weights remain unchanged.

### Shared Gooo receipt compatibility follow-up

The [typed-path common receipt observation](docs/typed-path-common-receipt-20261002.md)
uses the own model in 32 actual generations and compiled runs on existing frozen
Korean/English inputs. All 512 ordered outputs match, with 64 local predictions
and zero external calls. The new compiler binds path/source/test/control identity
and finite functional scores into the common Gooo-declared receipt, retaining
unresolved runtime and full-domain claims. Model codegen median is 31.040 ms with
19.500 MiB median peak RSS; weights are unchanged and this is not a new model
quality or speedup claim. Full raw evidence is public on Hugging Face.

## Own feedback Gooo model v2 — preceding two-choice experiment

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

## Native runtime and source-linked completeness

The [native runtime study](docs/native-runtime-receipt-20261002.md) records 32
actual own-model/disconnected Gooo generations and 64 compiled executions.
All 768 finite expectations match, including 256 inputs absent from the current
selection suites. The new compiler runtime producer preserves parent bytes and
connects actual execution back to original Gooo identity through its common
receipt. Codegen median is 35.396 ms with the own model and 30.541 ms disconnected;
the build/process boundary dominates native observation cost. Raw captures,
failure history and remaining unknowns are retained in the public HF appendix.

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
