---
license: mit
language:
  - en
  - ko
tags:
  - gooo
  - golang
  - program-synthesis
  - candidate-ranking
  - experimental
---
# Gooo whole-candidate order judge

A small, independently initialized model for choosing which permitted **Gooo
program body to try first**. It reads a complete Korean or English instruction
and the ordered operations of each candidate. Gooo supplies a typed assembly
space; this judge ranks complete constructions and a bounded search checks
finite examples.

Think of the compiler as a workshop with eight assembled pieces on the bench.
The model suggests which piece to try; actual measurements determine whether it
fits. Equal pieces can reuse an earlier measurement.

## Artifact and integration status

- Dev #1202 passed its own CI and independently verified proof, then merged as `a11a3f08`.
  Main #1203 checks the identical tree on snapshot `22c2c914`. The change reads source inputs, structural metadata and whole-candidate
  weights through a shared nonblocking Unix opener, developed on `0eb69e5f`.
  Actual installed source/model FIFO waits and their controlled writer releases
  are retained. Current 64-swap controls have zero writer releases/timeouts;
  25 regular completions keep prior Go and 3,200/3,200 observations. Both model
  FIFO requests return a file error without a writer or output directory.
  A separate actual own-model/deterministic smoke keeps 1,024/1,024 in eight
  constructions/four judgments/16 native runs. End-to-end compiler warm responses were
  25.321709/27.571583ms; first responses of 298.670458/485.046083ms are retained.
  The first metadata error's total 528.789584ms is retained too. These are
  construction-and-execution timings; model-only prediction latency is unobserved.
  The tasks are existing authored tasks with unchanged weights/fit. Current installation is
  the proved main below. [Original wait, controls and Go readers](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/file-input-open-20261004).
  Separate snapshot-worker concurrency controls preserve four constructions/two
  judgments/eight native runs and 512/512 existing finite cases. Both processes
  reach EOF and join without a timeout; original process times are retained.

- Clearer terminal expectation states are installed on clean main `01d21e92`.
  Dev #1200 and main #1201 passed their own canonical CI and independent
  source proofs, then merged through normal expected-head merges. Fresh
  installed replay made eight ordinary constructions/four own-model judgments/16
  native runs keep 1,024/1,024 finite expectations and all earlier generated Go.
  Separate missing-tool, invalid-expectation and FIFO controls display unobserved
  expectations; two deliberately changed-expectation requests retain actual 0/256.
  Nine synthetic formatter states are recorded separately from native work.
  The original JSON/status/exit and runtime contracts are preserved. Current
  compiler source is `01d21e9260b2c0f1d02f8029824f3dda41631e9d`; installed binary
  SHA256 is `3d1e7ba724961bb7ad8e1866e6588bbe6212ceeee6bc93a4c888a62e2e1594cc`.
  Installed KO/EN model warm responses were 29.644083/24.497167ms; deterministic
  responses were 30.934083/24.965833ms. First responses 834.644792/477.573250ms
  and 324.127917/299.734458ms are retained. This display study has no matched
  speed comparison. The variable/condition quickstart separately made two
  deterministic constructions/four native runs, keeping 6/6 and both earlier
  generated sources. Read-only saved receipt checks retain UNKNOWN for no output
  and PROGRESS for actual zero, original units/profiles/decisions and explicit
  null aggregate scores. Weights and fit are unchanged.
  [Sources, original failures and scope](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/finite-observation-display-20261004).

- One 128×32 bilinear matrix: **4,096 parameters, 16,384 FP32 bytes**.
- Full UTF-8 instruction up to 512 bytes, positional byte-bigram/trigram features.
- Eight complete candidates, each described by two ordered integer operations.
- One root-order and two operand-order decisions; add/subtract/multiply,
  local/input/constant leaves, model constants -16..16.
- `model.json` and `weights.bin` load with the custom Go `orderjudge.Load` API.
- The 128-byte prediction workspace excludes feature arrays, preparation,
  artifact loading, program objects and receipts.
- A fresh compiler allocation study retains one 32 KiB hash scratch per used
  executor and releases it on close/cancellation. Warm full-file hash allocation
  was about 33.5→0.8kB/call. Old/new ABBA construction used these unchanged weights
  and deterministic controls: 96 generations, 48 actual predictions, 192 native
  runs, 12,288/12,288 finite expectations and all frozen generated Go matched.
  Warm responses were 64.13/65.88ms, with overlapping one-device ranges; this
  observation supports allocation reduction, with speed improvement unestablished.
  Whole-host utilization and model-only RAM remain unobserved. Dev #1196/main
  #1197 passed their own canonical CI and independent source proofs, then merged.
  [Original records, failures and Go collectors](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/owned-hash-reader-20261004).
- CPU fit and local Go inference. This architecture currently has FP32 weights.
- Earlier installation study: main `93fa2742e2f681c86546ee6fc459a76bae640913`,
  with direct file construction, immediate execution, owned runtime comparison
  and literal multiline bodies. Fresh installation replay made eight generations,
  four actual predictions and 16 native runs: 1,024/1,024 finite expectations and
  all eight earlier generated Go sources match. Separate raw clamp cases pass
  8/8 in the earlier raw-body installation. Runtime v3 also retains the first
  native Go version check, verifies current tool bytes/context and excludes
  historical checks from current resources. Dev #1198/main #1199 passed their
  own six canonical checks and independent immutable proofs, then merged.
  Optional phase diagnostics and file-binding readback are now installed.
  Fresh installed warm responses were 26.54/27.57ms with the model and
  25.50/25.99ms deterministically; original cold/slow observations remain public.
  A missing-tool request retains exit 1, 128 unobserved expectations, zero
  predictions/runs and the `native unobserved` display. A separate executable
  FIFO blocked before regular-file validation in the earlier compiler; its
  interrupted observation is retained. The installed Unix nonblocking-open/file-kind
  check rejects a writer-free FIFO. Its installed total CLI time was 19.665ms,
  with 5.527ms request response and no predictions/native runs; the earlier
  candidate's 604.14ms observation is retained separately because preparation and
  cache conditions differ. Dev #1198/main #1199 passed their own immutable proofs,
  main promotion authorization and live tree/sole-parent checks, then merged.
  Same-window old/new controls kept 12,288/12,288 finite expectations in 96
  generations/48 predictions/192 native runs; warm medians 21.54/22.07ms overlap.
  [FIFO originals, measurements and Go collectors](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/nonregular-tool-20261004).
  The installed direct variable/condition quickstart additionally made two
  deterministic constructions and four native runs, preserving 6/6 expectations
  and both earlier generated sources. That fixture's denominator is recorded
  separately from the Korean/English 128-example requests. Weights and training
  are unchanged.
  [Current installation and original failures](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/nonregular-tool-20261004).
- The released Go SDK v0.2.20 adds preparation reuse to the existing direct
  in-compiler generation route.
  The [native collection](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/order-judge-native-20261003)
  uses compiler commit `9d570ab1` in [PR #1176](https://github.com/kimjooyoon/meta-ontology-go/pull/1176).
- [Main PR #1179](https://github.com/kimjooyoon/meta-ontology-go/pull/1179) is merged
  as `8117fbaefac490a28e78c956bf1d40aed1372608`, with SDK v0.2.20 preparation reuse.
  The clean installed compiler
  reproduced eight existing Korean/English requests in 16 generations, 8 model
  calls and 32 compiled executions: 128/128 finite expectations passed.
  [Installation and CI evidence](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/order-prepared-installed-main-20261003).

The runtime is in the
[Go SDK v0.2.20](https://github.com/kimjooyoon/gooo-decision-runtime/tree/v0.2.20-experimental/orderjudge),
extracted from the separately pinned
[research implementation](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/8a41bb0f825dfd3f950101b33491581492c740f6/internal/orderjudge).
The compiler can export its source-bound expanded plan with `body-context --include-plan`.
The separately published [shared judge](https://huggingface.co/asketeddy/gooo-shared-judgment-tiny-v1)
has earlier native integration and different feature/weight contracts.

## Training and lineage

The matrix starts at zero and is trained in Go on **64 authored Gooo requests**,
with 400 full-batch updates, learning rate 2 and L2 0.0001. Training labels are
sets of candidates meeting separately authored arithmetic references on three
selection inputs. Held development requests do not enter the fit. The first fit
and all regressions are retained. CPU fit time was **89.61 ms** on an Apple M4.
No GPU updates were used for this matrix.

There are no Laya checkpoint weights, teacher outputs, private user messages or
repository secrets in this model. Authored English/Korean arithmetic templates,
typed Gooo plans, constants and finite reference outputs form the dataset.
The complete 160-request dataset, original sources, features, 400 loss records,
all predictions and manifests are in the
[public evidence bundle](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/8a41bb0f825dfd3f950101b33491581492c740f6/publication/order-judge-initial-20261003).

## Results and costs

Each development request has eight finite evaluation cases, including five inputs
not used for candidate selection. Families are represented in training; these
are small controlled development contrasts.

| Development contrast | Requests | First complete, deterministic / learned | Body evaluations with budget 8, deterministic / learned / learned + equal-body reuse |
| --- | ---: | ---: | ---: |
| New instruction template | 32 | 16 / 26 | 48 / 48 / 38 |
| Renamed local + commuted operands | 32 | 16 / 30 | 48 / 36 / 34 |
| New constants | 32 | 16 / 26 | 48 / 52 / 38 |
| Total | 96 | **48 / 82** | **144 / 136 / 110** |

All budget-8 arms complete 96/96 requests and 768/768 finite evaluation cases.
Deterministic equal-body reuse also uses 144 evaluations. The reuse follow-up
was designed after viewing the original observations and uses unchanged weights.
It is explicitly a development experiment on an already observed cohort.

The initial collector made 320 post-fit predictions. A separate replay made
**480 actual predictions and 960 actual bounded searches**, plus 320 baseline
SDK searches. It reconstructed the first observations exactly and recorded every
candidate evaluation. A later SDK replay reproduces the complete records on
Linux amd64 and macOS arm64, with only the two timing fields excluded.

On M4/macOS arm64, Go1.27.1:

- Prediction alone: median **1.875 µs** on the 160 initial timed inputs.
- Budget-8 complete search wrapper: median **322.04 µs** deterministic,
  **340.83 µs** learned, **339.25 µs** learned with reuse.
- Whole original source-collection/fit/evaluation process: 1.94 s wall time,
  1.60 s CPU time (about 82.5% of one core on average), 43.39 MiB maximum RSS.
- Whole follow-up replay: 0.91 s wall, 0.59 s CPU, 47.66 MiB maximum RSS.

Prediction timing excludes feature extraction and model loading. Process memory
includes the collector and associated work. These measurements show fewer body
evaluations, while preparation and bookkeeping still make learned search slower
in this small cohort. No peak system CPU utilization claim is made.

### Direct compiler follow-up

With the same weights and zero further training, the compiler completed **256
generations, 128 actual predictions and 512 immediately compiled executions** on
the 64 existing new-template/new-constants requests. Budget-1 complete requests
are 32/64 deterministic and 52/64 learned (28/32 English, 24/32 Korean). At budget
8 both routes complete 64/64 requests and 512/512 listed expectations. The model
with exact descriptor reuse evaluates 76 bodies versus the ordinary deterministic
route's 96. All outcomes and full predictions reproduce the frozen study.

Median budget-8 generation wall time is 8.579 ms deterministic versus 9.256 ms
with the model; maximum RSS medians are 17.56 versus 18.60 MiB. OS-reported
per-command CPU usage corresponds to 88.14% versus 92.50% of one core. Model
prediction alone takes 2.75 µs. Each generation starts a fresh compiler process;
native compilation and two runs take another roughly 310 ms. These are process
measurements with a fixed observed development cohort. The
[raw evidence and cost breakdown](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/order-judge-native-20261003)
retain all partial outcomes: 1,706/2,048 finite cases across both budgets/routes.

Within the supported profile, pass `--path-model /path/to/model.json` to
`gooo body-codegen --json --activity Compose --path-plan recipe.json source.gooo`.
Place this model's `weights.bin` beside the metadata. Omit `--path-model` for
deterministic generation. Explicit seeds, model-level batching and feedback are unsupported
by this first model route and produce recorded errors before prediction.

The [runnable Korean example](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/examples/whole-candidate-order)
includes the Gooo source, recipe, finite cases and commands for generation and
compiled execution. It asks for multiplication before addition while the original
body uses the opposite order.

## Run the evidence replay

Use Go1.27.1 and a fresh pair of temporary output directories:

```sh
git clone https://github.com/kimjooyoon/gooo-neural-decision-experiments.git
cd gooo-neural-decision-experiments
git checkout 8a41bb0f825dfd3f950101b33491581492c740f6
unzip -q publication/order-judge-initial-20261003/initial.zip -d /tmp/gooo-order-initial
go run ./cmd/order-judge-replay \
  --input /tmp/gooo-order-initial --output /tmp/gooo-order-replay
```

The archive holds exactly the model published here. The replay loads its weights,
reconstructs features and source candidates, checks every frozen prediction,
executes bounded searches and writes all attempts. `orderjudge.Search` accepts a
nil model for deterministic operation. Its caller supplies a deadline, a typed
plan, finite cases and an attempt budget. Compiler source binding remains the
compiler caller's responsibility.

### Repeated native toolchain checks

With these unchanged 16,384-byte weights, a later compiler experiment kept the
first actual Go version check in an owned executor. Each request still hashed
the full current Go file, checked embedded release/platform, replayed source and
ran the compiled body twice. Shell wrappers received fresh version processes.

Four EN/KO model/deterministic conditions used baseline→candidate→candidate→baseline
windows of six requests. The two existing multiplication/addition ordering fixtures produced 96
generations, 48 actual predictions and 192 native runs, with 12,288/12,288 finite
expectations and every generated source unchanged. Warm response median over
40 requests per version was 79.91→63.69ms; current-child CPU was 31.21→13.37ms.
Initial response medians were 659.25→630.17ms over eight requests per version.
Host scheduling/cache variation remains; no new intent task or training update
is counted. CPU/RSS describe current children, excluding parent/model-only/host
utilization. Maximum single-child RSS changes do not measure total request RAM.

Runtime v3 preserves the first process/output while excluding it from current
costs. Eight old v2 comparison pairs stay byte-identical; cross-profile changes
have no numeric completeness delta. The
[full original observations and replay method](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/toolchain-version-reuse-20261003)
include the final source's eight-generation/1,024-expectation confirmation.
Main #1193 passed its own canonical checks and independently verified proof,
then merged. Clean main `ed2c2cac` is installed and freshly passes 1,024/1,024
in eight generations/four predictions/16 runs. All generated sources match;
first version checks are preserved and repeated requests execute two current
native children. Installed warm model responses are 76.82ms Korean and 74.07ms
English; these later observations retain host/cache variation separately from
the paired timing study.

The research workflow at source `2b30367b` retains a FAIL in the earlier shared
model's numerical replay: 101 summary leaves differ from its frozen reference.
The current whole-candidate judge and separate-arithmetic replay jobs pass; 18
of 19 jobs pass. Original failures and exact differences are linked in the same
publication. The whole research workflow records FAIL.

The subsequent [legacy replay repair](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/legacy-platform-replay-20261003)
merged as research main `87c6f219`, after all 19 jobs in
[workflow 37126899513](https://github.com/kimjooyoon/gooo-neural-decision-experiments/actions/runs/37126899513)
passed. It requires every historical row and compact artifact to reproduce the
frozen result for the reader's actual platform. The earlier shared model's
101 differing summary path/value pairs and 272 changed full rankings remain
recorded as the original `MISMATCH`. The explicit-arithmetic cross-platform
check passes separately. This repair leaves this order judge's 16,384-byte
weights and its previously reported native behavior unchanged.

## Reading the limits

The feature projection is narrow and lossy. Long instructions, branching,
nested operations, multiple locals, other output types and broader domains need
separate scoped experiments. Finite completeness is tied to the listed inputs;
ranking probabilities are uncalibrated. There is no fresh general-language
benchmark or measured 1.58-bit version for this architecture.

The Go SDK now provides `orderprepared.NewRuntime` and `Runtime.Prepare` as two
explicit preparation steps, followed by repeated `Prepared.Search` calls. The
runtime captures a private immutable model; each prepared plan holds eight checked
programs and their fixed features. Each search predicts and checks its current
finite cases. The SDK has no global cache. Compiler integration retains at most
one complete plan per generator and binds source again before reuse.

On 64 already observed requests, 3,840 SDK searches and 1,920 model predictions
matched the original complete selection records, with zero further training.
Budget-8 model search medians were 356.98 µs for the original wrapper, 244.04 µs
for capture + preparation + search per call, and 4.83 µs for retained preparation.
Three observed live-heap deltas for 64 prepared plans and one model were about
1.7 MB. The [full SDK observations](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/order-prepared-sdk-20261003)
include the initial memory-measurement correction and Linux replay. These numbers
measure SDK intervals; compiler generation and native execution have separate costs.
Existing V3/V4 models and their historical numerical comparisons remain preserved.

The [compiler API follow-up](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/order-prepared-native-20261003)
completed 512 generations, 512 fresh predictions and 1,024 immediate native runs.
At budget 8, second-call Generate medians are 1.017 ms for a fresh owner and
0.693 ms for a retained owner; including recipe expansion and constructor gives
2.225 versus 1.627 ms. Both retain the original 64/64 complete requests. Budget-1
outcomes remain 52/64, with every original failure preserved. This compares API
ownership patterns within one compiler revision, with the original weights.
The compiler change is in [PR #1178](https://github.com/kimjooyoon/meta-ontology-go/pull/1178),
merged to dev as `9fc2634720f0a43ca7d72bac46aacba5c77822ce` after canonical CI.
[Main promotion #1179](https://github.com/kimjooyoon/meta-ontology-go/pull/1179)
also passed its checks and source-bound proof, and is merged and installed.
The installed revision and independent execution observations are linked above.

The [retained worker follow-up](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/prepared-worker-native-20261003)
ran 96 generations, 48 model predictions and 192 native executions, matching
768/768 declared expectations. Responses arrived before input EOF. All 16
sequential second model calls reused preparation; alternating two plans in
four-request batches produced no reuse in that collection. Only the latest plan
is retained. These are known examples, with one or four worker slots; batch
intervals include native execution of preceding responses and do not measure
parallel speedup.

For easier integration, [PR #1180](https://github.com/kimjooyoon/meta-ontology-go/pull/1180)
adds `gooo body-path-stream` using the same runner as `gooo-body-worker` and is
merged to dev as `e9d1fd0f87fc1aa95756fd55293421051f6f3650` after canonical CI.
An explicit `--model` retains this model in one process; omission uses
deterministic construction. Local dogfooding passed four generations, eight
compiled runs and 32/32 finite expectations. [Main promotion #1181](https://github.com/kimjooyoon/meta-ontology-go/pull/1181)
passed CI and independent proof verification, and is merged as `cb2892cb583e693190bd68f75eb4df223819f053`.
The clean installed main reproduced four generations, two model predictions,
eight compiled runs and 32/32 expectations, including second-request preparation
reuse. [Installation evidence and runnable instructions](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/body-stream-installed-main-20261003).

The compiler now reuses an original checked projection within recipe
expansion, retaining current-source and fallback checks. With these same weights,
512 additional generations and 1,024 native executions preserved the original
selection, ranking and finite outcomes, including partial results. A historical
timing comparison showed no wall improvement. Four paired process trials then
measured 8,192 fresh decodes: median decoder interval 0.3714→0.2921 ms and
allocated bytes per decode 728,580→571,800. All 4,096 document/plan pairs matched;
751 timing pairs were slower. These are compiler preparation costs; weights and
training are unchanged. [Every observation and scope](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/source-recipe-projection-reuse-20261003).

[Main #1183](https://github.com/kimjooyoon/meta-ontology-go/pull/1183) passed canonical
CI and independent proof verification and is merged and installed as
`3bee55daf8063b0daee3d543e57a528975bc63ee`. Existing English/Korean tasks, the
integrated stream and the Go example completed 24 generations, 12 model calls,
48 compiled executions and 192/192 finite expectations. All source/selection
comparisons matched. [Installation evidence](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/recipe-projection-installed-main-20261003).

### Run generation and execution together

The [Go example](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/examples/whole-candidate-order)
reads a Gooo source, short recipe and finite execution cases, sends one request
to the integrated stream, executes its response immediately, then sends the next
request with input still open. `go run ./cmd/order-example --model ... --out ...`
uses this model; omitting `--model` uses deterministic construction. It saves
every response, generation and runtime report, including finite failures.

Local and Linux collections each completed four generations, two actual model
calls and eight compiled executions with 32/32 expectations. Local model
response times were 51.39 ms initially and 2.06 ms on the repeat; native command
times were 655.81/309.06 ms. The first response can include worker setup. Linux
native times were 5,351.99/182.90 ms; that long first interval is retained.
These are two known sequential requests per route, with uncontrolled cache and
platform differences. [All observations, CPU/RSS and stage costs](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/order-example-immediate-20261003).
The new example's race, vet and actual Linux execution checks passed. No model
weights or training changed.

### Retain one executable and execute current inputs

The compiler [dev #1184](https://github.com/kimjooyoon/meta-ontology-go/pull/1184)
adds `body-path-stream --execute` and per-request `execution_cases`. Each newly
constructed body immediately enters native observation in the same process.
One owned executable is retained; current source/plan/parent replay, executable
hash checks and two current-input executions are performed on every request.
`--retain-native` selects this route in the Go example. Model omission remains
deterministic. Native construction can use bounded parallel workers; one
cancellable gate owns the native workspace and serializes execution.

On the original 64 known EN/KO development sources at budgets 1/8, 512 additional
generations/predictions and 1,024 compiled executions matched every original
ranking, source and finite result: 3,720/4,096 expectations, including partial
failures. There were 384 actual builds and 128 artifact reuses, with zero training
updates. On macOS arm64, warm execution observation median was 293.60→28.89 ms;
all 128 paired calls were faster. Current-child CPU median was 214.98→9.76 ms;
maximum single-current-child RSS median was 86,171,648→14,483,456 bytes. Prior
build history is excluded from reuse costs. These are current child observations,
separate from model tensor storage and parent/whole-host memory or utilization.
The first owned call still builds; its execution median was 299.50 ms.

The immediate Go example completed four generations, two model calls, eight
native runs and 32/32 expectations across model/deterministic routes. Repeat
responses including execution were 24.41/24.90 ms; first responses were
967.38/433.50 ms. Startup/cache/platform differences remain present. The first
collector failure exposed typed-live-scope versus decoded JSON ordering in parent
comparison; both sides now pass the same exact-number decoder before comparison.
The failure, raw observations and fixed protocol are
[public](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/retained-native-execution-20261003).
An exact-source Linux follow-up also passed: four matching generated bodies,
two predictions, eight native runs and 32/32 expectations. Repeat responses
including execution were 36.10/38.31 ms for model/deterministic routes. The
dedicated job and immutable artifact are linked in the report; this does not
change historical full-workflow arithmetic failures.

Dev #1184 and [main #1185](https://github.com/kimjooyoon/meta-ontology-go/pull/1185)
passed six canonical checks and independently verified immutable proofs, and are
merged. At that stage, clean main `8c6ec01c4931186460244f3a2975013edda62325` was locally installed.
Four generations, two predictions and eight compiled runs matched 32/32 finite
expectations and preceding generated sources. Both repeat calls reused their
executable; responses including execution were 30.72/29.52 ms. The first model
response of 1,267.64 ms is retained. Changed expectations produced a new 1/2
result and an omitted expectation was rejected before generation. Installation,
exact proof archives and all observations are linked in the report. Weights
remain unchanged.

### Direct file construction and current installation

Dev #1186 and [main #1187](https://github.com/kimjooyoon/meta-ontology-go/pull/1187)
passed their own six canonical checks and independently verified immutable proofs.
Clean main `fc854ef4` was installed with Go1.27.1. Source files, a named activity,
full typed plans or short recipes and finite expectations can now be passed
directly to one compiler command. It saves generated Go, original inputs, each
generation/runtime record and an incremental summary before the next request.

With that compiler in PATH and Go1.27.1 available, from the research repository:

```sh
gooo body-path-run --source examples/whole-candidate-order/source.gooo \
  --activity Compose --path-plan examples/whole-candidate-order/recipe.json \
  --cases examples/whole-candidate-order/cases.json \
  --model publication/order-judge-initial-20261003/model.json \
  --repeat 2 --out order-file-results
```

Use a fresh output path. `--go-bin` can select the local Go1.27.1 binary. Omit
`--model` for deterministic construction. The installed compiler ran this exact
source/recipe/case example: two generations, two actual predictions, four native
runs and 16/16 supplied expectations. The larger named EN/KO 128-input collection
separately completed eight generations/four predictions/16 runs and 1,024/1,024
expectations, with four executable reuses and unchanged generated sources.
Model-on warm response including construction/execution was 33.092ms Korean and
32.415ms English; first responses were 496.269/474.290ms. Current-child CPU across
all four warm arms was 8.829..10.957ms; maximum single-current-child RSS was
14,237,696..14,417,920 bytes. These costs exclude the earlier retained build and
do not measure model-only memory or whole-host CPU utilization. Startup/cache
conditions remain uncontrolled.

The initial sparse-example fixture retained two valid selection candidates and
missed all supplied separate runtime expectations. A source Gooo oracle added a
discriminating observation and resolved one candidate; selection/probe-disjoint
runtime inputs then passed 4/4 across two requests. Actual predictions in the
resolved oracle arm were zero, including when this model was loaded. Original
failures, unchanged weights and exact denominators remain in the
[public records](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/body-path-file-cli-20261003).
CI feedback requires a supporting model profile and explicit step/round budgets;
this whole-candidate model currently uses ordinary bounded search and rejects
feedback/batching options.

The v2 reader correction [dev #1188](https://github.com/kimjooyoon/meta-ontology-go/pull/1188)
and [main #1189](https://github.com/kimjooyoon/meta-ontology-go/pull/1189) passed their
own six canonical checks and independently verified immutable proofs, then merged.
Clean main `05746e4a` is now installed. It read 18 earlier installed-record pairs
with byte-for-byte output matches, then used this unchanged model for two new
generations/four native runs and 256/256 finite expectations. The separate variable/
conditional assembly example passed 6/6 in two generations/four runs. Warm
responses including execution were 31.297/31.455ms. Changed resource units and
4→3 denominators retain both observations without a numerical improvement.
[Reader correction sources, failures and original records](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/owned-runtime-comparison-20261003).

### Writing bodies on multiple lines

[Dev PR #1190](https://github.com/kimjooyoon/meta-ontology-go/pull/1190)
adds backtick strings for Gooo `computes` bodies. Canonical formatting preserves
the decoded content and stable IDs; literal whitespace can affect source digests
and generated layout. Local syntax/body-codegen race and whole-module vet pass.
The clean experimental compiler used this unchanged model on raw Korean/English
bodies: eight generations/four actual predictions/16 native runs, 1,024/1,024
finite expectations and all eight original generated Go sources matched.
Model-on warm responses were 32.719/31.766ms, first responses 487.173/477.383ms.
Dev and [main #1191](https://github.com/kimjooyoon/meta-ontology-go/pull/1191)
passed their own six canonical checks and independently verified immutable proofs,
then merged. Clean main `eb8477d5` was installed at that step. Its fresh eight generations,
four actual predictions and 16 native runs pass 1,024/1,024; all generated sources
match. Installed model warm responses were 82.580/81.439ms, first responses
957.629/826.189ms. Deterministic warm responses were 93.576/76.820ms. Current-child
CPU sums were 27.163–34.796ms and maximum single-current-child RSS
14,172,160–14,385,152 bytes. These exclude parent compiler/model-only/whole-host
resources and retain uncontrolled cache/scheduling effects. Weights were unchanged.
Original TDD, fixture and collector failures remain in the
[raw body records](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/raw-body-usability-20261003).

## Optional current-request wall phases

The [compiler diagnostics experiment](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/body-wall-phases-20261003)
adds `body-path-run --timing` and `--verify-timing --out` from source b6f577c3.
The source-fixed candidate and its earlier d360355b observation together made
192 generations, 96 actual predictions and 384 native runs on the same two
known KO/EN ordering fixtures. All 24,576 finite expectations and generated Go
bytes matched the earlier results. No weights, optimizer updates or intent tasks
were added. The original Go1.27.1 modernizer failure is retained.

The final candidate's warm plain/timed response medians were 64.77/64.71ms,
current-child CPU sums 13.56/13.74ms and maximum single-current-child RSS medians
4.42/4.42MB. Timed phases identify Go-file hashing (14.40ms), source replay
(3.16ms), the two native invocations (9.61/9.36ms) and original-file save/binding
(2.68/0.73ms). Saving is outside response time. Initial input/model loading,
timing-sidecar/summary writes, stdout and cleanup are outside phase capture.
These noisy local observations identify costs; they do not establish a recorder
speedup, inference-only latency, model RAM or whole-host CPU utilization.
Sixteen separate read-only checks passed with zero additional model/native work.

Dev #1194 and main #1195 merged after their own six canonical checks and
independent exact-source proof verification; main promotion also bound the live
dev tree and main parent. Clean main 041c8bbf is installed with Go1.27.1.
Fresh installation smoke made eight generations, four predictions and 16 runs,
keeping 1,024/1,024 finite expectations and original generated bytes. Four saved
timing checks passed separately. Warm installed model responses were 66.81ms KO
and 64.48ms EN, deterministic 67.18/63.34ms. First responses 1,240.08/783.29ms
and 623.77/625.37ms are retained with the originals. These installation samples
are separate from the earlier paired experiment. Installation was checked on
2026-10-04 KST; weights and training scope are unchanged.

## Reproduce body candidates and feedback in Go

The companion language curriculum now has a Go1.27.1 replay path, merged in
[public PR #19](https://github.com/kimjooyoon/gooo-metaprogramming-experiments/pull/19).
It executes 93 original and 96 repaired candidate packages for the same 32
intentions. All 189 result files are removed from temporary copies before
execution and then compared to the frozen bytes. The tool also binds 96 provider
plans and 32 feedback sets to the fresh training observations. This replay makes
zero model calls and preserves each candidate's finite behavior, compiler
failures and source versions. The weights published here retain their existing
training scope.

Run `go run . --revision2 --root ../.. --output /tmp/gooo-candidates-new` from the
corpus's `tools/baseline-replay` directory. Use a fresh output folder.
[Korean execution and result-reading guide](https://github.com/kimjooyoon/meta-ontology-go/wiki/Replaying-Body-Experiments)
and [source-bound local/Linux records](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/go-revision2-replay-20261003)
show how to read compilation, training and evaluation counts separately.

## Research acknowledgments

[ConvAI Innovations' Laya](https://huggingface.co/convaiinnovations/laya) motivated
the structured-choice interface. [Saman Sarker Joy's decomposition study](https://arxiv.org/abs/2609.33971)
raised useful questions about composing local judgments. [DeepCoder](https://arxiv.org/abs/1611.01989)
and [DreamCoder](https://arxiv.org/abs/2006.08381) connect learned guidance with
program search and reusable structure. Gooo studies a small source-bound assembly
space with explicit finite observations and cheap local inference.

### 한국어 요약

Gooo가 만든 여덟 가지 코드 후보에서 무엇을 먼저 시도할지 고르는 16 KiB 모델입니다.
64개 한영 요청으로 짧게 학습했고, 개발 집합 96건의 첫 선택 완성은 48건에서 82건으로
늘었습니다. 같은 기능의 중복 후보를 다시 실행하지 않으면 실제 후보 평가가 144회에서
110회로 줄었습니다. 전체 탐색 시간은 조금 늘어서 준비·기록 비용을 줄이는 작업을
이어갑니다. 컴파일러 내부 연결도 완료해 추가 학습 없이 256회 생성과 512회 컴파일된
실행을 관측했습니다. 기존 개발 과제 64건에서 첫 시도 완성은 32→52건, 최대 여덟 번
시도하면 양쪽 모두 64건을 완성했습니다. 실제 생성 중앙값은 8.58→9.26ms였고,
원본 과제·가중치·실패·비용을 함께 공개했습니다.
