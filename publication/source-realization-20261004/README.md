# Reuse a selected Gooo body in the next generation

Observed 2026-10-04, macOS arm64, Go 1.27.1, SDK v0.2.21-experimental,
clean compiler candidate `55a6b663332d295f14b6b78e27644f32c6972545`.
[Compiler PR #1217](https://github.com/kimjooyoon/meta-ontology-go/pull/1217).

## The language problem and change

Previously a selected Go projection was available, while its Gooo body existed
only inside the compiler. Copying that selected body into `computes` could break
the next assembly: a selected local became the default while its alternative
still named the same local. Recomputing coordinates also loses interacting
alternatives when declarations and reads move together.

Generation now returns `gooo_source`. The activity keeps the original `baseline`
and one ordered `picked` label per choice beside its working `computes` body.
The baseline is the coordinate map; the working body is the implementation.
The original choices, intent, expectations and budget remain reusable. A
`body-realize` command reconstructs the saved selection and its finite observations
without loading a model, then saves exact original/generation records and the
checkpoint into a fresh directory. Partial functional scores remain visible.

## Actual own-model dogfood

Two authored activities × model/deterministic × three consecutive generations
gave **12 constructions**, **48/48 selection expectations**, **108/108 independent
runtime expectations**, and **24 fresh native runs**. Every construction was
realized, then its saved Gooo became the next generation's input. Each
activity/mode retained one Gooo digest, one generated Go digest, one document
digest and, in model mode, one joint input digest across the three stages.

The six model generations made six real joint predictions. Deterministic
generations and all twelve realizations made zero predictions. Existing own
weights were reused; no new GPU training or Laya calls occurred. This adds
reuse of source intent and structure. The original [ranking comparison](../source-assembly-20261004)
still records weaker first model candidates on these tasks.

An actual generation record from compiler `0f7ec4c` was also replayed and upgraded.
[`legacy-realization.json`](legacy-realization.json) records its original body-only
source digest, the new checkpoint digest, 5/5 finite cases and zero model calls.
That observation is separate from the twelve-construction totals.

## Time and memory

Each generation, realization and execution was a separate CLI process. Times
include launch, loading and JSON output. Each execution builds a fresh native
artifact and runs it twice. These timings differ from the retained-worker response
timings in the earlier study. Three sequential observations per row, fixed order
and warmed native build caches limit comparisons.

| Activity / mode | Median generation | Median realization | Median execution |
| --- | ---: | ---: | ---: |
| Qualified / model | 15.696ms | 15.738ms | 319.398ms |
| Qualified / deterministic | 14.421ms | 14.643ms | 315.238ms |
| Clamp / model | 13.214ms | 13.984ms | 310.140ms |
| Clamp / deterministic | 11.235ms | 12.297ms | 307.433ms |

Model prediction itself took 19.75–32.88µs. Model generation process maximum RSS
was 20.0–21.02MiB; realization was 19.84–21.94MiB across both modes. Native
compilation/execution resources are recorded separately. RSS includes children;
it is not model-only tensor memory. `metrics.json` retains wall time and macOS
`time -l` user/system CPU seconds for all 36 commands. CPU seconds are rounded
to hundredths. Whole-host utilization change remains unobserved. There is no
speedup or accuracy improvement claim from these two repeated tasks.

## Reproduce

Build the compiler from the clean candidate above or a main revision containing
that tree. Select the actual Go1.27.1 executable. From this research repo root:

```sh
go run ./cmd/source-realization-observe --gooo /path/to/gooo \
  --private-out /tmp/gooo-realization-raw --out /tmp/gooo-realization-public
```

The reproducer uses the fixed public source/cases in
[`../source-assembly-20261004`](../source-assembly-20261004) and the unchanged
[`own-three-feedback-v1`](../../models/own-three-feedback-v1/set-feedback/models/fp32)
bundle. It requires macOS BSD `time -l`. Raw execution records and local paths
stay in the private output. Public generation records contain the authored
source, picked paths, complete finite failures/results and own-model input hashes.

For a single source:

```sh
gooo body-codegen --json --activity Qualified \
  publication/source-assembly-20261004/source.gooo.fixture > /tmp/generation.json
gooo body-realize --source publication/source-assembly-20261004/source.gooo.fixture \
  --generation /tmp/generation.json --out /tmp/gooo-realized
gooo body-codegen --json --activity Qualified /tmp/gooo-realized/realized.gooo
```

The checkpoints preserve finite choices. Additional body types, calls, loops,
larger Korean/English intent coverage and known-selection reuse are further work.
