# Gooo condition chains through typed recipes

Observed 2026-10-03, darwin/arm64, Go 1.27.1. Compiler source
[`bf51d65a`](https://github.com/kimjooyoon/meta-ontology-go/commit/bf51d65ab5e21f9ef662fcc5a7f64a87d9bea9c3)
uses SDK v0.2.18-experimental. The development branch is
[`agent/source-recipe-condition-chains-20261003`](https://github.com/kimjooyoon/meta-ontology-go/tree/agent/source-recipe-condition-chains-20261003).

An ordinary Gooo body already accepts this range-limiting function:

```gooo
activity Clamp(Integer) -> Integer computes "if input < 0 { return 0 } else if input < 10 { return input } else { return 10 }"
```

At compiler `b742ba75`, ordinary generation passes but source-recipe decoding
fails with `recipe else requires an explicit block`. The new lowering treats the
next condition as the single statement in the preceding else branch. Original
source positions, condition order, branch-local names and the existing nesting
and arena limits remain in use. The emitted Go has explicit nested blocks.
Typed-tree binding verifies the source/presentation relationship.

## Native observation

The recipe exposes the outer branch, the inner comparison's operand order and
the inner branch: three binary choices, eight declared combinations. Three
selection cases are `-1 -> 0`, `5 -> 5`, `11 -> 10`. Seven native cases include
those inputs, both int64 endpoints, 0 and 10. Expectations use ordinary clamp
arithmetic: `max(0, min(input, 10))`.

| Three repeats per arm | Deterministic | Frozen own model |
| --- | ---: | ---: |
| Whole generation median | 7.958 ms | 9.227 ms |
| Process peak RSS median | 17.45 MiB | 17.91 MiB |
| Process CPU relative to one core, median | 90.64% | 91.58% |
| Actual joint predictions | 0 | 3 |
| Evaluated candidates per request | 1 | 1 |
| Native finite expectations met | 21/21 | 21/21 |

All six emissions are equal. Twelve compiled runs completed. All **42/42**
declared native expectations matched, including 24 selection-disjoint
expectations. Both arms chose mask 0 on their first attempt. The authored fallback
already satisfies the cases; this observation checks the newly accepted source
shape and model integration. Seven combinations per request remain unattempted.

Actual isolated prediction times were 7,666, 7,917 and 8,209 ns, median 7.917
microseconds. Whole generation includes loading, recipe expansion, validation,
selection and emission. Model ranking adds work in this already complete body.
The result supports a deterministic route for this particular construction.

One authored clamp and one bilingual intention are repeated three times per
arm, with alternating arm order and existing build caches. Generation times
range from 7.347 to 367.183 ms without the model and 8.116 to 9.767 ms with it.
The first deterministic process took 367.183 ms; the raw observation is retained,
and its cause was not isolated. Cache state was not instrumented. Whole-host CPU
change, controlled cold-cache cost, broader task accuracy and cross-platform performance
remain unmeasured. The model weights and all training settings are unchanged;
optimizer updates are zero.

## Reproduction and retained scope

The existing collector adds a `condition-chain` profile:

```sh
go run ./cmd/constant-recipe-observe --profile condition-chain \
  --compiler /path/to/clean/gooo \
  --compiler-sha bf51d65ab5e21f9ef662fcc5a7f64a87d9bea9c3 \
  --compiler-source /path/to/meta-ontology-go \
  --go-bin /path/to/go1.27.1 \
  --out /path/to/fresh/evidence
```

It retrieves both compiler fixtures from the exact commit with `git show` and
checks the compiler's build revision, clean state and SDK version. Collector
source first appears at research `964112f`; both Go source digests are retained
in `manifest.json`. The default constant profile and its older captures remain
available. This new profile does not change those old data files.

This directory contains the source, recipe, exact runtime cases, six complete
generation receipts, six native replay/build/execution receipts, process costs
and digests. `SHA256SUMS` covers every file except itself. Model metadata and
weights are pinned to the unchanged public compact `bag-original/fp32` export.

Compiler regression checks cover explicit-block equivalence, source-order
selection of the inner condition, shared-local assignment, scope rejection,
depth limits and changes to condition order. Four affected packages passed race
checks; whole-repository vet, billing and modernization checks passed. Full
serial local tests retain macOS namespace/symlink/source-splitter failures, with
no runtime bounded-child failures in this run. Linux PR CI and main deployment
for the condition-chain change follow its publication.
