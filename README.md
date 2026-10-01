# Gooo neural decision experiments

## Continued partial construction

Korean/English Gooo intentions can now rank bounded structural paths once and
continue finite construction in batches. First-shot accuracy is not the goal:
partial results, failed cases, type rejections, unattempted paths and cost remain
inspectable. Models are optional; the disconnected order is deterministic.

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
