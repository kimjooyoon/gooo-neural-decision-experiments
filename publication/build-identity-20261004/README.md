# Read the running compiler and the Go chosen for native execution

2026-10-04. [Compiler PR1208](https://github.com/kimjooyoon/meta-ontology-go/pull/1208)
adds an actual build view and a native-tool mismatch hint. Clean candidate
`ebd5f653eaab0036b2586a5cbdd8187bce780600` is frozen before these observations.
Its canonical CI has its own status; installed main remains `4f6c7566` / SDK
v0.2.21 until normal dev/main promotion and fresh installation complete.

```sh
gooo version --build
gooo version --build --json
```

The new `gooo/build-identity/v1` view reads embedded metadata from the running
executable: exact source-binding state, module version, build Go, SDK dependency
and any replacement. A modified, missing or malformed VCS identity remains
unbound. Six synthetic source states and dependency-replacement tests are
regressions, separate from real code construction. The new view makes zero
model calls or native executions. The existing version text/v1 JSON remains
byte-compatible with the installed main control. That old main rejects the
new build flag with exit2; its original response is retained.

The actual clean candidate reports its full source, Go1.27.1 and SDK v0.2.21.
The Go selected for later native work has a separate observation. A real
Go1.26.5 request preserves its generated program and actual version output,
then reports `--go-bin /path/to/go1.27.1/bin/go` as the next action.

| Existing authored-task lane | Constructions / actual model calls | Native runs | Current finite expectations | First / reused response |
| --- | ---: | ---: | --- | --- |
| Own order model, explicit Go1.27.1 | 2 / 2 | 4 | 16/16 | 1,073.26 / 72.60ms |
| Deterministic, explicit Go1.27.1 | 2 / 0 | 4 | 16/16 | 676.70 / 67.98ms |
| Own order model, actual default Go1.26.5 | 1 / 1 | 0 | Unobserved: 8 declared | 88.95ms, execution failed |

The successful lanes keep all 32 supplied expectations and every frozen
generated Go byte. Total construction calls are five and model predictions
three, including the failed native setup. Native executions are eight. The
responses cover decoding/construction/execution, excluding initial model setup
and saving/output. Different work scopes and first responses remain explicit;
no speed improvement is inferred. These repeat one existing authored order
task, with zero new intent tasks, training updates or weight changes.

## Continue a generated body after fixing the tool

The failed request's original generated body can also be verified and executed
directly. A separate `body-execute` request with Go1.27.1 kept its exact generation
parent, replayed the source/projection and completed two native runs with 8/8.
This adds runtime evidence with zero new model predictions. The original eight
unobserved expectations stay in the failed record; the appended execution has
its own outputs. `recovered-execution.json` retains that result.

```sh
gooo body-execute --source _order-results/source.gooo \
  --path-plan _order-results/recipe.json \
  --generation _order-results/run-1-generation.json \
  --cases _order-results/cases.json --go-bin "$gooo_go_bin" \
  > recovered-execution.json
```

Use the output directory from the original failed request and the actual
Go1.27.1 executable. This command uses the saved source and selection, with
fresh deterministic projection verification and actual execution.

## Saved evidence and read-only consumption

`_observations` holds all actual source, recipe, expectations, generated Go,
response, generation, runtime and timing files. The leading underscore makes
Go package discovery skip independent program snapshots. When using these
file commands inside a Go module, an output name such as `_order-results`
keeps repeated snapshots out of that module's package discovery too.

The reader checks exact compiler identity, legacy version bytes, per-request
model counts, all eight int64 inputs/expectations, native process results,
source/replay state, frozen Go and every timing-sidecar file binding. Wrong-tool
expectations stay null/unobserved; the actual version and next action remain.
The first failing regressions and corrected race results are included.

```sh
shasum -a 256 -c publication/build-identity-20261004/SHA256SUMS
go run ./publication/build-identity-20261004/readback \
  publication/build-identity-20261004
```

The reader makes zero predictions and zero native executions.
