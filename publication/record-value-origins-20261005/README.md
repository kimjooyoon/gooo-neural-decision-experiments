# Record value origins: representation and preparation cost

2026-10-05. This is a source-representation observation following the
[preregistered plan](../../docs/local-value-origin-plan.ko.md). It prepares the
next small-model study. Four authored source forms and eight exports belong to
one approach; they are not eight distinct approaches.

## What changed

Gooo can save a record or scalar before later field writes. The new source
graph keeps definitions, copies, writes, reads, previous choices, branch joins
and early-return continuation guards. A saved record resembles a photograph
beside the working document: later edits change the document's current fields.
The source determines which one an expression reads.

Compiler source `c330ee6d5d86def8a2955c2c17a534e9fc3f4426` adds this graph and a
new model-input contract. [Compiler PR1235](https://github.com/kimjooyoon/meta-ontology-go/pull/1235)
is the development integration; its merge/install status is tracked separately.
The candidate binary receipt in this folder records a clean Go1.27.1 build,
immutable SDKv0.2.24 and no replacement. SDK
[PR9](https://github.com/kimjooyoon/gooo-decision-runtime/pull/9) merged after
its unit/race/replay CI passed. These sources implement the new inference path
with synthetic ABI test weights. The published v1 model weights are unchanged.

## Independent executable probe

`cmd/record-origin-probe` invokes the candidate compiler, reads complete exported
contexts, and projects them through the public Go SDK. `probe/` retains all four
Gooo sources, both complete exports per source, all float arrays, hashes and
command wall/CPU observations. Its six checks pass:

| Check | Observation |
| --- | --- |
| Old receiver collision | Current and saved receiver forms have identical v1 arrays |
| New receiver distinction | Their v2 arrays differ |
| Local renaming | The v2 array is unchanged |
| Finite input replacement | The v2 array is unchanged |
| Source and text identity | Different receiver forms retain different source/context hashes |
| Execution separation | All eight exports record zero predictions and candidate executions |

The source profiles deliberately use saved record and scalar values outside the
choice expression too, so both alternatives type-check without unused locals.
They retain the public fixture's typed finite cases for preflight. No cases are
executed in this probe, so it does not measure candidate functionality.

The old inputs occupy459–465 bytes and new inputs708–714 bytes. These eight
compiled CLI invocations take6.40–11.23ms wall time and5.66–10.13ms child CPU.
CPU/wall is87.5–90.2% of one core during the command. Calls are ordered and
startup/cache conditions differ; this series does not establish a speed ranking
between feature versions. Whole-host CPU increase, peak RAM and GPU utilization
are unmeasured here.

## Source preparation benchmark

The separate original sequential fixture has42 graph nodes, a6,727-byte graph
and a717-byte v2 model input. Each benchmark starts from an already prepared
assembly plan, rebuilds the typed source graph, derives unique ancestor counts,
and encodes/validates the complete context. It excludes source-contract preflight,
model prediction, finite candidate search, build and execution.

Five repeats on Apple M4/macOS arm64, Go1.27.1:

| Source | Median ns/op | Median allocated B/op | Typical allocations/op |
| --- | ---: | ---: | ---: |
| Initial `8f578402` |65,691 |152,588 |654 |
| `c330ee6d`, type information held separately |59,919 |86,892 |654 |

The adjustment separates type-checker-owned information from fixed graph
scratch. Temporary allocated bytes fall43.1% in this example; time falls8.8%.
The complete CLI export is byte-identical before/after the adjustment.
`benchmark-initial.txt` and `benchmark-type-info.txt` preserve every repeat.
Allocated bytes per operation describe temporary heap work, separate from live
RAM, peak RSS, model tensor storage and the complete application footprint.

## Reproduce

Build the compiler at the pinned source with Go1.27.1. From this research checkout:

```sh
go run ./cmd/record-origin-probe --compiler /path/to/gooo \
  --source /path/to/meta-ontology-go/examples/body-codegen/record-field-updates.gooo.fixture \
  --out /tmp/gooo-origin-probe
```

The output directory must be new. Failed exports remain in the observation
report and make the probe fail. From the compiler checkout:

```sh
go test ./internal/bodycodegen -run '^$' \
  -bench '^BenchmarkRecordOriginContext$' -benchmem -count=5
```

## Next measured step

The input uses64 expression,32 origin and160 intent slots per field. Origin
counts summarize each ordered alternative with16 slots; the shared256/8/2
judge still scores fields independently. Counts can collide for different
graphs. The regression pair is one controlled distinction.

Next, mixed goals across all eight masks, unchanged/negative requirements and
held-out Korean/English wording will compare equally sized old/new judges under
the same small training budget. FP32 and ternary QAT require separate quality
observations, followed by actual generated-program execution with budgets1/2/8.
No origin-model fitting or new functional-accuracy claim is made by this folder.
The original plan placed branch joins after the first local-copy step. They are
included in this implementation because the current source choices live inside
guards; their regression evidence is source-only and does not expand this probe
into a broad conditional-program evaluation.
