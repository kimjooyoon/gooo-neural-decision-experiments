# Installed record field assembly

Compiler main `f6da667e11951b6939fc5e30442e01a09ca06e85`, clean tree
`fb844d23d86ddd216ef5caa9a416303d5f603805`, Go 1.27.1, SDK
v0.2.21-experimental, macOS arm64. Main [PR1226](https://github.com/kimjooyoon/meta-ontology-go/pull/1226)
passed its complete [CI run](https://github.com/kimjooyoon/meta-ontology-go/actions/runs/37210035915)
and source-bound result verification. Both local executables were built from
that clean main revision and installed at 2026-10-04 14:58:39 UTC. The public
`installation.json` retains source/build identities with local paths omitted.

The installed executable produced 24 fresh controls and six saved full-budget
replays: budgets 1/2/4/8, three paired trials per model/deterministic mode. This
is one authored three-field/two-node graph. Five selection cases and seven
runtime cases have five overlapping inputs; two runtime cases add escaping and
an unchanged-record path. All 30 controls retain the candidate's selected source
and generated Go hashes, rankings, selected masks and finite output/field counts.

| Attempt budget | Selection fields, both modes | Selection cases, both modes | Compiled record fields, both modes | Compiled named outputs, both modes |
| ---: | ---: | ---: | ---: | ---: |
| 1 | 6/15 (40%) | 2/5 | 9/21 | 6/14 |
| 2 | 9/15 (60%) | 2/5 | 13/21 | 6/14 |
| 4 | 12/15 (80%) | 2/5 | 17/21 | 6/14 |
| 8 | 15/15 (100%) | 5/5 | 21/21 | 14/14 |

Full-budget medians from three fresh trials per mode:

| Observation | Own FP32 model | Deterministic |
| --- | ---: | ---: |
| Local prediction | 31.625µs | 0 calls |
| Complete graph generation | 9.890ms | 9.337ms |
| Native execution pipeline | 302.444ms | 303.667ms |
| Whole command | 323.462ms | 324.194ms |
| Peak process RSS | 82.94MiB | 82.42MiB |
| Process user + system CPU time | 0.24s | 0.24s |

The own FP32 model is unchanged: 74,624 resident tensor bytes and the published
metadata/weight identities recorded in every model capture. Fresh model requests
make one initial prediction; all six saved compositions make zero new
predictions. Saved replay still builds the native graph and runs current cases
twice. The runtime pipeline includes tool selection, toolchain observation,
source replay, Go build and native executions.

These process resources include Go builds and children. Earlier development
has warmed the build cache, and paired mode order alternates. Small timing and
memory differences include host noise. Host-wide CPU utilization changes are
unmeasured. The model and deterministic controls have equal finite counts and
attempt counts on this shape. Field-specific training and cross-source quality
remain subsequent work; this cohort introduces no new weights or GPU training.

`metrics.json` retains all rows and process resources. Each fresh response keeps
the actual values and generated source. Full-budget saved compositions and Gooo
checkpoints are included beside their replay. Read remaining fields with:

```sh
go run ./cmd/record-completeness \
  --input publication/record-field-assembly-20261004/installed/budget-4-model-0.json
```

The summary reports 12/15 selection fields (80%), then shows the missing `state`
values and the separate compiled field/output ratios. The [earlier candidate](../candidate)
and [one-request ternary pilots](../quantized-pilot) retain their own revisions.
