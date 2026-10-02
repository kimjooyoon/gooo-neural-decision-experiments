# Actual three-choice own-teacher observations

Source/collector revision: `65313ae54b50aeb6203aa8c91cb48fc6cdfd68ac`.
The preregistration and native corpus freeze remain unchanged. Collection and
the separate zero-inference audit were complete at this collection phase; no
new three-choice student had then been trained, tested on development or promoted.
The later [matched training and Go kernels](own-three-choice-training-results-20261002.md)
now have separate evidence, while full SDK/native behavior remains pending.

## Actual measured work

| Quantity | Observed |
|---|---:|
| Train function views | 2,048 |
| Fixed seeded SDK sessions | 4,096 |
| Initial independent predictions | 12,288 |
| Actual feedback predictions | 25,106 |
| Total actual teacher predictions | 37,394 |
| Distinct attempted candidates across sessions | 13,233 |
| Ordered interpreter case invocations | 211,728 |
| Fixed-coordinate observations skipped | 1,630 |
| Sole-remaining feedback observations with zero predictions | 225 |
| Teacher / derived student representation declines | 0 / 0 |
| Initial states across all splits | 3,072 |
| Unique actual-failure train continuation states | 7,667 |
| Total student rows | 10,739 |

These use the frozen v1 **independent own FP32** teacher. All predictions are
Go runtime calls; there are no Laya/provider calls, CI hints, optimizer updates,
student initialization or new native codegen calls in this phase.

| Candidate budget | Completed sessions / 4,096 | Percent |
|---|---:|---:|
| 1 | 987 | 24.10% |
| 2 | 1,921 | 46.90% |
| 4 | 3,104 | 75.78% |
| 6 | 3,657 | 89.28% |
| 8 | 4,096 | 100.00% |

This is a **training-cohort teacher curve**, not new-student performance,
holdout generalization, natural-language correctness or arbitrary-program
completeness. The eighth-candidate result includes bounded enumeration of the
entire authored legal space. Tied passing masks are retained. Mean attempts
are 3.231 per session, with 9.129 actual predictions per session.

## Cost and memory scope

- SDK session median: 228,541 ns (**0.229 ms**).
- SDK session p95: 577,041 ns (**0.577 ms**).
- Collection, state serialization and immediate audit: 10.137 s wall,
  11.058 s CPU, **109.08% of one core** on this run.
- Process lifetime peak RSS: 182,550,528 bytes (**174.09 MiB**).
- Go heap allocated after collection: 126,086,184 bytes (**120.25 MiB**).

Session time includes the original teacher's SDK initialization/ranking,
interpreter evaluations, search and internal receipt hashing. Derived student
contexts and the independent audit are outside that per-session interval.
Whole-phase CPU includes those additional tasks and concurrent Go runtime work;
it is neither a controlled host-utilization delta nor a neural-kernel benchmark.
Lifetime RSS includes full curriculum reconstruction and retained state maps.
The teacher FP32 tensor file is only 50,912 bytes. Dataset/process RAM is reported
separately from model storage and the fixed request workspace.

## Retained evidence and public replay

Teacher captures occupy 232,385,208 bytes; exact deduplicated student rows occupy
19,345,322 bytes. Both native source phases, including the first failed prefix,
and the teacher phase retain 295,192,703 bytes before public supplementary copies,
within the original 768 MiB study cap.

The [collection receipt](../publication/own-three-choice-teacher-collection-20261002.json),
[preexecution](../publication/own-three-choice-teacher-preexecution-20261002.json)
and [independent audit](../publication/own-three-choice-teacher-audit-20261002.json)
record exact hashes, calls, raw case values, failure text and source/model pins.
The public bundle includes every original phase file, frozen teacher bytes,
protocol and both full audits. Its Go packager checks every decoded member's
SHA-256/ZIP CRC, closed inventory, bounded size and privacy, including escaped
JSON strings and base64 native outputs. This package is source/teacher evidence,
not a release of trained three-choice model weights.

## Next matched experiment

Keep the original three arms: uniform initial, passing-set initial and
passing-set actual feedback. Use the shared fresh random initialization and
fixed MPS budget, all nine FP/PTQ/QAT exports, calibration-only selection and
the full SDK/native study. Group/language/state weights follow the immutable
protocol. No failed variant or inconvenient fixture will be replaced, and the
teacher is an observation source rather than student initialization.
