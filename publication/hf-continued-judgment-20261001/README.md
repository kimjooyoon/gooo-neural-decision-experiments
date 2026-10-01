# Continued Gooo judgment after a representation decline

The development target is bounded Korean/English judgment over Gooo structures,
partial construction and inspectable continuation. First-shot perfection is not
the acceptance target. Model expression limits remain visible in receipts.

## Observed repair

The frozen SDK v0.2.3 probe used two long-intent variants of one existing compound
Gooo function: English 475 bytes and Korean 478 bytes. Original intentions fit
the 512-byte input limit. Adding observed-failure context exceeded it. Optional
feedback stopped both runs after eight candidates and emitted no Go, although
rank-once search could exhaust 64 candidates and retain a six-of-seven body.

SDK v0.2.4 records a hashed zero-prediction context decline, consumes one bounded
feedback round and continues the unchanged candidate ranking. Source binding,
native types, stable identity and replay checks still govern body emission.
The original intention is never truncated, and same-batch feedback retries
remain rejected. Other errors and cancellation still stop generation.

| Same fixed long-intent variants | SDK v0.2.3 feedback | SDK v0.2.4 feedback |
| --- | ---: | ---: |
| Native calls | 2 | 2 |
| Candidates per call | 8 | 64 |
| Initial local predictions per call | 6 | 6 |
| New feedback predictions per call | 0 | 0 |
| Emitted typed Go bodies | 0 | 2 |
| Final finite pass count | English 6/7, Korean 5/7 before stop | Both 6/7 |

The final expected value was deliberately changed to 999; the emitted behavior
at that input remains 24. The unmet case is retained rather than relabeled as
success. Six of seven is **85.714% finite functional completeness**, not general
natural-language accuracy or proof over every integer input.

The new comparison has four actual native calls, 24 initial local predictions,
four zero-call context declines and 256 total candidate attempts across the
baseline/feedback policy views. One deduplicated emitted-Go process executed
seven function inputs and matched native and typed evaluation. These are two
existing variants, not four independent intention families. The model weights
are unchanged; no upstream Laya inference, optimizer step or GPU training was
performed in this repair study.

## Time and memory

The repaired English/Korean feedback processes took 15.420/11.783 ms wall time,
15.419/11.956 ms user-plus-system CPU time and 21,741,568/21,430,272 bytes maximum
resident memory. The first English baseline took 469.238 ms, while the Korean
baseline took 12.942 ms. This single baseline-first pass does not establish a
speedup; the first-process delay is retained. Host CPU utilization and its
increase were not sampled. Process CPU percentage in the raw report uses one
core over wall time and can exceed 100% for a multithreaded process.

## Sources and reproduction

- Clean compiler feature: `6a3e45d7f185c74f5136ab81b560a14a1366f367`.
- Execution runner: `27704aa95c790eca5e0d24645c7abb6e55ee7ac1`.
- Go 1.27.1; SDK `v0.2.4-experimental`.
- Existing positioned-random FP32 structural model; metadata digest
  `1ea3bada068f2487418f17270db4ba6785a98c3ad60e0ad7eb92359400a01bb5`.
- New captures: `runs/feedback-context-decline-20261001`.
- Preserved failing comparison: `runs/feedback-input-bound-20261001`.

```sh
go run ./tools/native-feedback-study --mode continued-audit \
  --out runs/feedback-context-decline-20261001 \
  --audit-output /tmp/continued-judgment-audit.json
```

Audit mode reinterprets all 256 recorded candidates and verifies captured Go,
unchanged ranking, native parity and decline/progress hashes without new model,
native or Go subprocess calls. Public HF publication adds an allowlisted
synthetic appendix; prior weights, model card and frozen studies are retained.
Feature measurements are separate from merged-main deployment evidence.

## What to optimize next

Use the planner to define typed alternatives for conditions, local references,
assignments and order; use the tiny model to judge which declared paths to try.
Measure remaining unconstructed behavior, unique paths attempted, explicit
abstentions/declines, work per accepted partial body and continuation cost.
Separate intention-family holdouts from repeated policy views. Ambiguous
Korean/English expressions can have multiple acceptable paths. This approach
does not require unrestricted prose generation to complete useful Gooo bodies.
