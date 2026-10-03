# A small description of source-relative operation order

2026-10-03. The [native source counterexample](../root-order-source-alias-20261003)
gave opposite correct root choices the same local model input. This experiment
asks whether a small typed-source description can retain the missing distinction.

The compiler's explicit `body-context --include-plan` option exports its already
validated and source-bound typed plan. The observation code reads that plan and
records two ordered integer operations, with operand kinds and exact int64
constants. Names are omitted; addition and multiplication operands are sorted.
Each alternative occupies 48 bytes, so the two alternatives use 96 bytes. The
separate feature version is `gooo/two-update-order-facts/v1`.

## Observed scope and results

Four authored families are reused from the [native order study](../native-order-20261003):
add/multiply, subtract/multiply, negate/add and square/add. Each appears in two
source orders, with English or Korean intent and three presentations. This gives
48 compiler context exports. Intent and selection expectations stay fixed when
the source order changes. The collector validates the source digest, plan digest
and original typed alternatives before observing them.

| Paired check | Result |
| --- | ---: |
| Source reversal: ordered descriptions exchange places | 24/24 |
| Same pairs with identical existing V3 local root input | 24/24 |
| Local-variable rename leaves the description unchanged | 16/16 |
| Commutative operand rewrite leaves it unchanged | 16/16 |
| English/Korean intent change leaves source facts unchanged | 24/24 |

The last row measures source facts only. Natural-language text remains a separate
input for a future model. No instruction-understanding score is inferred here.
The source-order check has twelve family/presentation pairs, each observed in
both languages; the 24 rows are not 24 independent arithmetic families.

No model predictions, candidate tests, native executions or training updates
occur in this preflight. The frozen V3/V4 implementations and weights are unchanged.
The previous native experiment still reports 6/16 first-candidate complete
requests for each own-model arm. This new input has no trained-model accuracy yet.

## Cost and limits

On Apple M4, Go1.27.1/darwin-arm64, five encoding benchmark samples measured
28.02–29.31 ns/op, 0 B/op and 0 allocations/op. These measure one 48-byte
descriptor in an already available arena. Compiler parsing, source binding,
validation, JSON, process startup and model work are outside that interval.
The 48 full context-export subprocesses took 5.537–6.980 ms each, with a 6.119 ms
median. Cache state was not instrumented; no comparative speed gain is claimed.

The admitted body is `let v=input; v=op(...); v=op(...); return v`, with integer
add/subtract/multiply and local/input/int64 leaves. Nested expressions, extra
locals, branches and holes decline. Fixed-array bounds and failure atomicity are
covered by unit tests, along with int64 endpoints and noncommutative swaps.
The wrapper independently validates the whole typed plan. Other choices remain
at the base structure; a later model needs to represent interacting choices too.

These are structural descriptions, not complete semantic equivalence classes.
For example, adding 1 then 2 and adding 2 then 1 receive different descriptions
even though their output functions agree. Generalization beyond the authored
families and choosing the right alternative remain open measurements.

## Reproduce and provenance

- Compiler: clean `ba00850f57db382f4b6a040e7a9ed14f0f6a98b4`, SDK v0.2.18.
  This is an experiment branch at collection time, not an installed-main claim.
- Collector: clean `ff4251698e731f90ace5ee2c86077a144f264ae1`.
- [Protocol and invocation](../../cmd/source-order-preflight).
- `manifest.json` pins the executable and feature identities; `records.json`
  preserves every signature and export timing. Each raw source, recipe and
  context export is included. `summary.jq` independently recomputes paired counts.
- Initial collector `f464a7e3212375cc6783e53a4312306ad528fb36` stopped after the
  first valid compiler export because it expected a bare source digest. The
  compiler correctly uses a `sha256:` prefix for source/input digests and bare
  hex for SDK plan identity. `ff42516` repairs this reader and adds a regression
  using that exact raw export in `cmd/source-order-preflight/testdata`.

The next small experiment can train against source-relative alternatives using
this retained order information and the complete instruction. It should report
finite native completeness and additional attempts separately from first-choice
accuracy, and compare with deterministic search on the same budget.
