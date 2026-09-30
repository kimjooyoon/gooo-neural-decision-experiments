# End-to-end Go stream measurement

Six sequential child processes completed all 24,576 planned records. Each
condition repeats the same frozen 256 test rows 16 times (4,096 records).
These are throughput repetitions, not 24,576 independent intentions.
No provider, Python or GPU was involved in these Go inference runs.

| Model | Workers | Records / batch wall | Throughput | Child CPU average, one core = 100% | Child peak RSS |
| --- | ---: | ---: | ---: | ---: | ---: |
| FP32 | 1 | 4,096 / 77.71 ms | 52,706/s | 98.6% | 10.31 MiB |
| FP32 | 4 | 4,096 / 54.21 ms | 75,559/s | 176.7% | 11.61 MiB |
| PTQ ternary | 1 | 4,096 / 72.24 ms | 56,698/s | 98.7% | 10.05 MiB |
| PTQ ternary | 4 | 4,096 / 55.23 ms | 74,159/s | 184.3% | 12.11 MiB |
| QAT ternary | 1 | 4,096 / 74.15 ms | 55,239/s | 98.7% | 10.28 MiB |
| QAT ternary | 4 | 4,096 / 55.03 ms | 74,426/s | 186.6% | 11.61 MiB |

Wall time includes child process startup, model loading, JSON processing,
queueing and stream I/O. Parent harness input/output work affects wall time but
is excluded from child CPU/RSS counters. Batch wall divided by record count is
an amortized throughput cost; it is not an individual request latency percentile.
CPU percentage is child CPU seconds / wall seconds, on a one-core basis; it
is not host utilization or a measured increase from host idle. Peak RSS uses
Darwin child `getrusage` byte units.

Every sequence and correlation ID was observed exactly once. FP32 matched all
4,096 labels and emitted correct IR for every record. PTQ matched 3,920 labels,
emitted 3,872 correct / 144 incorrect IR nodes, and abstained on 80 records.
QAT matched/emitted 3,872 correct and emitted 224 incorrect nodes. Counts are
the 256-row pilot outcomes multiplied by 16, unchanged across worker counts.

In this snapshot, four FP32 workers reduced batch wall by about 30% while
using more CPU time and about 1.3 MiB more peak RSS. There is one process per
condition; no repeated-condition confidence interval or per-request p50/p95
was measured. Do not use the small cross-model wall differences to rank
variants. Use FP32 for accuracy; explicitly choose `--workers 1` for a smaller
active footprint or `--workers 4` for a measured concurrency starting point.

The original first invocation failed after reaching the execution loop because
its report parent directory was absent. Its known failure is retained in
`../stream-end-to-end-v1-20260930/failed-attempt.json`; unavailable metrics are
not reconstructed. The driver now rejects a missing output parent before any
child can start. This successful second attempt uses a separate output path.

The machine report binds executable SHA-256
`0453957c83f95bd2b24c36a9db1a46e8a83df8bb35ce28d7ec3a451ea957c236`,
stream source closure, driver source, model bundles and frozen dataset. See
`report.json` and the independent review for exact counts and resource fields.
