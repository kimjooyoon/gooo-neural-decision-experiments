# Gooo neural decision experiments

Public experimental Gooo-specific natural-language to typed IR decisions.
Runtime, orchestration, data generation and compiler bridges are written in Go.
Python is used only for offline PyTorch/MPS training and checkpoint export.

First pilot: eight bounded binary-operation decisions, English/Korean synthetic
instructions, template-grouped train/calibration/test splits. This is a small
classifier, not a full natural-language compiler. Laya is the research baseline;
the first tiny model is independently initialized, not copied Laya weights.

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
[v0.1.0-experimental release](https://github.com/kimjooyoon/gooo-neural-decision-experiments/releases/tag/v0.1.0-experimental)
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

Future controlled studies and the completeness dimensions are in
[next-experiments.md](docs/next-experiments.md). They are plans, not claimed runs.

## Multi-node body experiment

The [typed body-plan path](docs/multi-node-body-plans.md) now supports authored
expression arenas, typed operation holes, scoped locals, assignments, nested
branches and returns. `cmd/gooo-body-compose` fills multiple holes using explicit
model choices or deterministic fallbacks. A bounded training-only search is
optional. The [Go study runner](tools/evaluate-bodyplans/README.md) validates
native Gooo generation and independently compiles/executes emitted Go, with
explicit planned, observed and unknown counts. This extension is separate from
the immutable v1 operator-model results and executable release.
