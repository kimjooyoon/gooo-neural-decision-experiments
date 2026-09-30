# Body-plan interpreter memory layout

`Compile` still uses maps while validating lexical scopes. The compiled
`Program` now stores a slot for each local declaration, a slot for each
assignment, and a compact expression-to-slot table indexed by lexical scope.
That scope dimension matters: the same expression node can be shared by two
separate branches whose local declarations have the same name but occupy
different slots.

Each `Evaluate` call creates a fixed `[maxStmts]Value` frame on its own stack.
Branch-local declarations have distinct slots, while assignment to an outer
local writes its existing slot. The compiled plan, choices, and slot tables are
read-only, so concurrent calls share no mutable evaluation state. Go and Gooo
rendering and compile-time type validation keep the same semantics. The measured
success path uses no heap allocations; this does not claim zero allocations for
the whole compile, render, or scoring protocol.

On the measured 64-bit Go target, `Value` occupies 32 bytes, so the fixed
128-entry frame reserves 4,096 bytes per active evaluation. The one-byte
expression-slot table has one row per lexical scope. A plan with 128 statements
can have at most 127 `if` statements (it also needs a return), so there are at
most 255 scopes including the root and 254 branch scopes. With 128 expression
nodes, the table's maximum logical size is 32,640 bytes; the Go slice capacity
may round above its length. Slot IDs are assigned in statement-tree traversal
order, and each `if` receives its then and else scope IDs in that same order.
This trades a bounded per-program table and a fixed stack frame for repeated
runtime map allocation and name lookup. Compile-time lexical validation still
uses maps.

Before recursive type validation or rendering, compilation also applies a
bounded preflight. It estimates encoded plan size with a 64 KiB budget, then
memoizes each expression node's Go and Gooo spelling cost once and charges that
cost at every statement use. This catches shared-DAG expansion before building
large intermediate strings. Each generated Go and Gooo source projection is
limited to 128 KiB; the Gooo estimate includes its quoted body. The estimates
are intentionally conservative and may reject a plan whose final source would
fit. The preflight does not change `Evaluate` or the frozen study cohort.

## Interpreter benchmark

The benchmark compiles `arithmeticPlan()` once, then repeatedly calls
`Evaluate` with varied inputs. It exercises local reads, reassignment, a
conditional, branch returns, and integer arithmetic. Compile and source
generation are outside the measured loop. Command:

```sh
go test -run '^$' -bench '^BenchmarkEvaluateArithmeticPlan$' -benchmem -count=5 ./internal/bodyplan
```

Captured on Go 1.27.0, darwin/arm64, Apple M4. The pre-change bodyplan source
SHA-256 was
`286fc65f177bc77a02b54889ef2f4928b60590b802d003b4c1c145a0b02583ae`.
The optimized slot-backed source at benchmark capture had SHA-256
`2f1a00967786ceea9679ca3a2abf9f1bb8dd4af8c951abc1df1430149b21ea9b`.
The current source adds the compile-time size preflight after that benchmark;
the exact measured source is archived as
[`bodyplan_optimized_before_resource_guard.go.txt`](../internal/bodyplan/benchmark-evidence-20260930/bodyplan_optimized_before_resource_guard.go.txt).

| Version | Median time | Range across 5 runs | Bytes/op | Allocs/op |
| --- | ---: | ---: | ---: | ---: |
| Map-backed evaluator | 239.7 ns | 229.5–286.0 ns | 544 | 5 |
| Slot-backed evaluator | 113.9 ns | 113.4–115.0 ns | 0 | 0 |

This is an isolated interpreter microbenchmark on one plan shape. It does not
measure model inference, JSON handling, native Gooo compilation, or end-to-end
study throughput. The five raw runs, per-run values, harness digest, and source
digests are archived in
[`benchmark-runs.txt`](../internal/bodyplan/benchmark-evidence-20260930/benchmark-runs.txt)
and [`benchmark-runs.json`](../internal/bodyplan/benchmark-evidence-20260930/benchmark-runs.json).
The exact test source at benchmark time and baseline `bodyplan.go` from commit
`7d626b9ab0b503f5c3d571be09524d689271bd48` are also archived beside those files.
