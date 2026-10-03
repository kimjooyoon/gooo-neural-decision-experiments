# Instruction order through native Gooo generation

Observed 2026-10-03 on darwin/arm64 with Go 1.27.1, installed main compiler
[`729482ed`](https://github.com/kimjooyoon/meta-ontology-go/commit/729482ed6ff6c3fd2a5664db415ced48e14e122b),
SDK v0.2.18-experimental. Collector and the pre-collection definition are pinned
to [`4671023a`](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/4671023a4e9e3a5d00bbb6ee61067088446f7f16/cmd/native-order-observe).

Four authored operation pairs, two language views and two requested orders make
16 requests per arm/budget. Three selection cases and eight native cases are
defined independently as ordinary arithmetic. Five native inputs per request
are selection-disjoint. Each source contains a local variable and two assignments;
the recipe allows their order and two operand orders to vary.

The two frozen compact models are `positioned-original/fp32` and
`bag-original/fp32` from the existing explicit-arithmetic export. Their learned
weights differ as well as their feature representation. This comparison measures
the existing models in use; it does not isolate the causal effect of an encoder.

## Completed behavior and search cost

| Arm | First attempt: requests meeting all eight cases | First-attempt cases | Up to eight attempts: complete requests | Cases after bounded search | Total candidate evaluations after bounded search |
| --- | ---: | ---: | ---: | ---: | ---: |
| Deterministic | 8/16 | 66/128 | 16/16 | 128/128 | 24 |
| Positioned own model | 6/16 | 52/128 | 16/16 | 128/128 | 53 |
| Bag own model | 6/16 | 52/128 | 16/16 | 128/128 | 46 |

The full collection has **96 generations, 192 compiled executions, 64 actual
joint predictions and zero training updates**. All controls together meet
554/768 finite expectations. The bounded-search arms meet 384/384, including
240 selection-disjoint expectations. Functional failures from the first-attempt
arms are retained in their generation and native receipts.

The eight masks describe structural choices. Some commute and produce equal
functions; their count is not a count of distinct behaviors. A root-order bit
matching the instruction also does not establish correctness of the two operand
choices. Both are visible in `records.json` and the full receipts.

## What happened to order information

Across eight reversed-intention pairs (four families × two languages), using
real source-derived inputs:

- Bag features and complete prediction distributions are identical in 8/8 pairs.
- Positioned features and prediction distributions differ in 8/8 pairs.
- Both existing models still propose the same first mask across each pair.
- Each model gets the root-order bit right in 8/16 requests and completes 6/16
  functions on the first attempt. English operand reversals cause additional failures.

The earlier synthetic preflight's feature distinction survives native context
construction, but it does not improve first-choice functionality with these
weights. A useful next model question is a small, separately versioned contrast
training experiment on source-bound order choices. The current result gives no
reason to deploy a new input ABI or replace the current model automatically.

Bounded TDD completes the declared tasks now. On this small family, ordinary
enumeration also uses fewer evaluations than either learned ranking. This is a
practical language result: an imperfect first proposal can still become a working
body through a small, explicit contract and a bounded continuation.

## Process observations

Each row below contains 16 different requests, one sample per request. These are
descriptive medians, not repeated per-task latency estimates.

| Arm and attempt budget | Generation median | Peak RSS median | CPU relative to one core, median |
| --- | ---: | ---: | ---: |
| Deterministic, 1 | 8.051 ms | 17.27 MiB | 88.47% |
| Positioned, 1 | 8.763 ms | 17.94 MiB | 90.01% |
| Bag, 1 | 9.075 ms | 17.88 MiB | 89.71% |
| Deterministic, 8 | 8.129 ms | 17.34 MiB | 88.92% |
| Positioned, 8 | 9.352 ms | 18.05 MiB | 90.59% |
| Bag, 8 | 8.861 ms | 17.88 MiB | 90.28% |

The first deterministic process took 574.741 ms and is retained. Its cause was
not isolated. Arm order rotates and budget order alternates, with existing caches;
cache state and whole-host CPU change were not instrumented. Native build/run
costs are separately retained in each record. No model speedup is observed here.

## Scope and reproduction

The source and natural-language examples are authored, small integer functions.
There is no unseen-task or paraphrase-generalization claim and no new training.
The expectation count measures these finite cases, not all possible inputs.
Model confidence, source validity, search coverage and runtime success keep
their separate meanings. In particular, a generation report's `decision: PASS`
can coexist with functional `PARTIAL`; the functional counts above come from
the actual case results.

Run the pinned [collector](../../cmd/native-order-observe/README.md) from a clean
checkout using the exact compiler identity in `manifest.json` and a fresh output
directory. This directory preserves every source/recipe/case, generation/native
receipt, process observation, model/input/feature digest and first/selected mask.
`SHA256SUMS` covers all files except itself. The raw publication is about 8 MiB.
