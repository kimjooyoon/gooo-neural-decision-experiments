# Actual own-model use with the shared Gooo receipt

Compiler [9fa2c608](https://github.com/kimjooyoon/meta-ontology-go/pull/1142)
connects typed-path selection and finite TDD to the Gooo-declared common receipt.
The [immutable public HF appendix](https://huggingface.co/asketeddy/gooo-three-choice-feedback-tiny-v1/tree/f5e0e2bd5105fc9c2f02a66f1f5a219a7135c0cc/research/common-receipt-20261002)
contains exact inputs, plans, generated Go, native observations, common receipts,
compiled outputs, the Go runner and independent reader.

## Observed scope

This is compatibility on already observed frozen inputs, configuration 20 / goal
4, eight composition families, Korean and English. The selected own FP32 model
is compared with disconnected deterministic search. There are 16 executions per
mode, in fixed order. No optimizer update, new held-out accuracy, model promotion
or total speedup is claimed. The original 640-pair study remains unchanged at its
original compiler revision.

- 32 actual Gooo code generations and independently compiled Go executions.
- 512 ordered outputs equal the original finite oracle.
- 64 actual local model predictions, zero predictions in disconnected mode and
  zero external provider requests.
- 32 common receipts consumed through the actual compiler decoder. All bind the
  full path observation, original/selected source, finite suite, search controls,
  local counts and model pins. All retain `execution_boundary` as their first
  unresolved compiler stage.

The model metadata SHA is
`e9d7f4770d4d8402c64eea116028e6d6c049b1522f50f399c05f2c3571932a3b`;
weights SHA is
`f5af1e35288cbad938d8416dd51873d83dff369a0e0c6e7e16f6e1f75b7f429b`.
Model weights are unchanged.

## Process observations

| Metric | Own model | Disconnected |
| --- | ---: | ---: |
| Codegen median | 31.040 ms | 27.417 ms |
| Codegen nearest-rank p95 | 479.179 ms | 32.693 ms |
| Median process peak RSS | 19.500 MiB | 17.484 MiB |
| Mean per-execution CPU, one core = 100% | 84.627% | 86.003% |
| CPU / wall across active codegen intervals | 47.813% | 85.936% |

Even-sized medians average the two middle observations. The first model process
is a cold outlier. CPU is process user+system time divided by wall time; whole-host
utilization was not measured. This fixed-order sample does not isolate a causal
CPU increase or throughput improvement.

## Meaning of completeness

The compiler now separates complete lowering from finite functional accuracy,
source binding, candidate observation and model accounting. Its regression
fixtures include complete, 2/3, observed 0/3 and unobserved final scores. The
original opt-in feedback field, unknown failure outcomes and zero-call context
declines retain their meaning. Different intent or search controls produce
different plan identities even when final Go is identical.

Finite compiler scores come from the bounded Go AST evaluator and typed
interpreter. The separate runner compiles and executes emitted Go, but those
external observations do not rewrite the original compiler receipt. A future
common runtime/reverse-observation producer must compose that evidence while
preserving original provenance. Unseen input behavior and full-domain semantics
remain unresolved.

The [consumer and resource record](../publication/typed-path-common-receipt-consumption-20261002.json)
contains exact measurements. The [anonymous public-byte verification](../publication/typed-path-common-receipt-hf-verification-20261002.json)
compares all eight public files to the prepared local bytes and checks every one
of the 293 ZIP members by size, SHA and CRC, without executing another model or
generated program. Raw fixture material passed a closed-file-list privacy scan.
