# Fresh installed graph reuse controls

Compiler main **`7acccf564b3def31ddd368050c2288f05bd7076f`** was normally merged
through [PR1228](https://github.com/kimjooyoon/meta-ontology-go/pull/1228) after all
12 jobs in [CI37214664915](https://github.com/kimjooyoon/meta-ontology-go/actions/runs/37214664915)
passed. The downloaded source-bound result was independently checked. Both
executables were built from the clean main tree with Go1.27.1 and SDK0.2.21,
checked on existing scalar/record flows and the new series, then installed.

This is a fresh capture using the installed CLI, separate from candidate data.
All 20 workloads match the candidate's generated-Go/driver hashes, current
finite output/field counts and build/reuse pattern. Original inputs, actual
records and deliberate wrong labels remain in each response. The same Go
observer captures 50 commands, 120 frames and 240 native executions, with five
fresh own-model judgments in total. Each workload keeps14/16 named expectations
and24/24 record fields; the two missing labels are deliberate expectations.

## Paired saved workload medians, three trials per origin

| Metric | FP32-origin stateless | FP32-origin retained | Deterministic stateless | Deterministic retained |
| --- | ---: | ---: | ---: | ---: |
| Six-suite workload wall | 2,327.40ms | 497.14ms | 2,334.28ms | 500.82ms |
| CPU user+system | 1.45s | 0.36s | 1.46s | 0.36s |
| Peak process RSS | 82.88MiB | 82.14MiB | 83.05MiB | 82.59MiB |

Each pair uses identical saved construction and current inputs, with zero new
predictions. Stateless means six sequential commands/builds; retained means one
command/build and five reuses. Startup count differs. The wall reduction is
about79% for this one six-suite workload; process RSS is similar. CPU/RSS include
native builds and children. Whole-host utilization change remains unmeasured.

Fresh FP32 prediction median is18.792µs, whole-command median493.708ms and the
15 subsequent reused-frame median19.767ms. The first disconnected fresh command
took750.03ms; its two later commands took502.06/505.12ms. Original observations
remain. One PTQ pilot predicted in18.875µs and one QAT pilot in19.333µs, with the
same current results and18,752 decoded resident tensor bytes each. Their stored
weight files remain3,854 bytes, with five trits per byte in the matrices and
FP32 biases. Weights and training are unchanged.

Installed byte identities:

- CLI: `b9054bdf0534969b57f5f09cc0b44fc92a24e5dabaeaf4fab5180172b7429c1e`.
- Worker: `22acc6fe6807455211ce62cbf70dc95ec90bafc37d98d1245e3a16342cb8c2ce`.

`--text` on the Go observer shows per-suite percentages, actual mismatches,
current build and both native-run times. The original cold candidate/native
startup observations are retained separately in the parent study.
