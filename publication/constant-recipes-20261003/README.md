# Constant bodies through Gooo recipes

Observed on 2026-10-03, darwin/arm64, Go 1.27.1. Compiler
[`b742ba75`](https://github.com/kimjooyoon/meta-ontology-go/commit/b742ba75b3107dfb5da1b385b7153bf47864a207)
uses released SDK v0.2.18-experimental. Compiler integration is tracked in
[PR 1170](https://github.com/kimjooyoon/meta-ontology-go/pull/1170).

The source declares an Integer input but never reads it:

```gooo
activity Compose(Integer) -> Integer computes "let first = 2 - 3; let second = first - 5; return second - 7"
```

The recipe gives each subtraction an operand-order choice. Its bilingual intent
asks for `(3 - 2)`, then `5 - first`, then `second - 7`. One selection case
requires `0 -> -3`. There are eight declared combinations. The compiler can now
retain the signature input without inventing a read or dummy arithmetic.

## Observation

| Three repeats per arm | Deterministic | Frozen own model |
| --- | ---: | ---: |
| Whole generation median | 9.209 ms | 9.731 ms |
| Process peak RSS median | 17.42 MiB | 19.31 MiB |
| Process CPU, one-core-relative median | 90.56% | 97.96% |
| Actual joint predictions | 0 | 3 |
| Evaluated candidates per request | 5 | 7 |
| Native finite expectations met | 18/18 | 18/18 |

All six generations emitted the same Go source. Twelve compiled executions
completed; the six-input suite includes both int64 endpoints, -7, -1, 0 and 17.
Five inputs per request are disjoint from the selection case. All 36 declared
expectations, including 30 selection-disjoint expectations, matched. The model's
first proposed body failed the selection case, as did the deterministic initial
body. Finite search reached the same successful body in both arms.

The model made one shared prediction per request. Isolated prediction times were
7,708, 7,875 and 7,667 ns; the median is 7.708 microseconds. Model loading, source
recipe expansion, validation and search are included in whole generation time.
The unchanged model increased candidate evaluations from five to seven in this
task. This run supports using the deterministic route for this small constant
construction; it does not show a speed improvement from model ranking.

This is one authored body and one bilingual intention, repeated three times per
arm, with alternating arm order and ordinary warm build caches. It measures a
newly supported language shape and its finite execution. Training updates are
zero. Whole-host CPU change, cold-cache latency and new-task generalization were
not measured. The process CPU values above use process CPU time divided by wall
time, relative to one core.

## Retained evidence

- `source.gooo`, `recipe.json`, `cases.json`: exact source and contracts.
- Six `*-generation.json` files: complete source-bound selection and failed attempts.
- Six `*-runtime.json` files: original-generation replay, native builds and two
  executions each. The source/recipe and original generation are re-consumed by
  `body-execute` before running.
- `processes.json`: every process timing/RSS and finite count.
- `manifest.json`: compiler/binary/collector/model digests, totals and scope.
- `SHA256SUMS`: exact bytes for this directory, excluding the checksum file itself.

The collector is
[`cmd/constant-recipe-observe`](../../cmd/constant-recipe-observe), first committed
at `b8505ae`. It uses the public compact `bag-original/fp32` model from
`publication/full-input-separate-arithmetic-20261003`. The weights are unchanged.

Before the fix, the source recipe `return 2 - 3` failed with
`plan must contain exactly one Integer input named input`. The SDK also rejected
a declared but unread input node. Research commit `5cda16c` fixes that node rule;
SDK PR3 pins the exact new extraction and preserves its predecessor manifest.
Local arm64 and Linux SDK checks each replayed 18,432 frozen input rows with
36,864 actual predictions and no differences. Those compatibility predictions
are separate from the three new predictions counted in this pilot.
