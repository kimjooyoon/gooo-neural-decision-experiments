# Compact Gooo judgments through actual native codegen

## What ran

The frozen 16 Gooo views (eight composition families, English and Korean,
configuration 20, goal 4) were generated with each FP32, PTQ ternary and QAT
ternary shared judge, in expanded and compact representations. Each generation
was immediately followed by a native build and two executions. Representation
order alternated after each view. The study made **298 actual model predictions**
in **96 generations**, including **202 predictions conditioned on finite test
failures**. This run did not provide a CI hint or update weights.

- Collector: `7e9be4489d15663cbac8f3e71bc02d2e4649dfe5`.
- Clean compiler: `7db19b6bc9a2909aa059f1265d39a539c3573a57`.
- SDK: `v0.2.14-experimental`; local Go `1.27.1`, darwin/arm64.
- Protocol SHA-256: `2dae15f2fdf33991124815685e239f8b77e36d24200685b5caf906236de4fb81`.
- Prior full-state audit SHA-256: `1aff17949d9b066949edc98c57e195e1a322207488e3e245579e8da15c7d3125`.

The collector bounded each child to 75 seconds, stdout to 1 MiB and stderr to
64 KiB. Every generation/execution completed. These 96 observations contain no
timeout or deadlock; they do not establish absence of deadlocks for arbitrary
programs or concurrent serving. Separate compiler race tests cover concurrent
retained requests. New raw compact-study evidence stays under a combined 32 MiB
cap, including the previous kernel audit, with a 4 GiB minimum free-space check.

## Actual finite behavior and completeness

All **2,304/2,304 finite expectations** passed in **192 compiled runs**, yielding
4,608 ordered outputs. Of the expectations, 768 used inputs disjoint from the
current selection suite. They are not an independent training holdout.

All **48 expanded/compact pairs** had equal generated Go, complete unseeded path
search, semantic feedback observations and ordered native cases. Only verified
artifact identities, artifact-bound receipt-chain hashes and measured timing
were normalized for that comparison. Every original capture remains available.
Seeded behavior was not equated across different artifact hashes.

An independent consumer using the exact compiler source decoded all 96 runtime
receipts, checked original source/generated-code/parent receipt bindings and
unchanged unrelated completeness dimensions, and verified **596 progress-chain
records and 202 feedback-chain records**, including actual model identities and
cumulative call counts. Native execution made zero model or external calls.

The first unresolved completeness dimension remains **permission_boundary** in
all 96 receipts. No aggregate completeness percentage is invented. Finite
functional success does not claim permission, network or whole-language
completeness. Previously reported bilingual judgment regressions are unchanged.

## Timing and memory

Each cell below is an observed median. There are 16 generation samples per
representation; prediction samples are 47/45/57 per representation for FP32/PTQ/QAT.

| Variant | Expanded prediction, µs | Compact prediction, µs | Expanded codegen, ms | Compact codegen, ms |
| --- | ---: | ---: | ---: | ---: |
| FP32 | 48.958 | 23.709 | 30.350 | 28.927 |
| PTQ ternary | 40.625 | 25.000 | 28.776 | 27.765 |
| QAT ternary | 35.291 | 30.208 | 29.237 | 29.511 |

The compact QAT end-to-end median was slightly higher in this run. Kernel
improvement is not a guaranteed end-to-end speedup. Fresh subprocesses load the
model for every generation; prediction timers exclude loading, startup,
generation and native builds. Retained-generator measurements are a subsequent
experiment, not a result of this one.

Compact codegen CPU time divided by wall time was **83.7–84.6% of one core** at
the median. This is process CPU consumption, not a measured increase in whole
machine utilization. Actual generated-program peak RSS medians were
**4.20–4.23 MiB**; these child programs contain no inference model. Inference
process peak RSS was not independently measured in this study. Model storage is
separately 8,288 bytes for FP32 or 446 bytes for ternary weights; ternary resident
tensors occupy 2,096 bytes plus eight scale bytes and the caller workspace is
3,200 bytes. Packed trits are decoded for this Go runtime.

The Go build cache was cleared immediately before the study. Its first expanded
FP32 generation took 626.446 ms and the corresponding native build 6,453.338 ms;
both are retained, including in nearest-rank p95 (16 samples means p95 is the
maximum). The enclosing invocation took 95.47 seconds and included collector
compilation. Native timing had no concurrent source builds. These local samples
do not establish a causal speedup or a cross-platform performance guarantee.

## Reproduction and public evidence

The compact models and preceding 10,739-state kernel comparison are published at
[the immutable model revision](https://huggingface.co/asketeddy/gooo-shared-judgment-tiny-v1/tree/985999a89caba6a31cc7147f66ba29a5ce76a1d9/research/compact-runtime-20261003).
This native appendix adds every generated source, plan, case suite, generation,
runtime receipt and process observation, with a closed manifest and compressed
archive. It includes the frozen collector and independent consumer source. No
training dataset or weight duplication is needed.

From the research repository, inspect the bounded package with:

```sh
go run ./tools/package-shared-three --mode verify-bundle \
  --output publication/compact-shared-native-20261003
```

For independent compiler receipt consumption, extract the verified archive,
check out compiler source `7db19b6bc9a2909aa059f1265d39a539c3573a57`, copy
`native/independent-consumer.go` to a temporary ordinary `.go` filename in that
compiler root, and run it with the extracted `native` directory as its sole
argument. It performs no new predictions or generated-program executions.

This experiment supports using the compact own model for bounded Gooo decision
paths, with finite tests and deterministic continuation carrying incomplete
judgments forward. The next measurements should isolate retained model loading
and serving cost, while language development still needs broader source/intent
discovery and explicit before/after completeness deltas.
