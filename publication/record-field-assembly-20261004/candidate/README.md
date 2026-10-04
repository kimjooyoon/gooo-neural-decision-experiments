# Candidate field-value assembly controls

Clean compiler source `48f7027584281ef87e43220f680093bc693f4488`, Go 1.27.1,
SDK v0.2.21-experimental, macOS arm64. These observations were captured before
main promotion. A subsequent compiler commit uses the Go 1.27 struct-field
iterator required by the existing modernizer check. Installed observations have
their own source revision and resource measurements.

The graph contains `Select(Candidate, Boolean) -> Candidate` and a connected
`Label(Candidate) -> Text`. Three constructor-field choices permit keeping the
title, setting the state, and appending a marker to the original reason. Five
selection cases score 15 expected fields. Seven runtime cases score 14 named
outputs and 21 record-output fields. Five runtime inputs overlap the selection
suite; two add escaping and an unchanged-record path.

Four budgets, two modes and three paired fresh trials produce 24 observations.
Six saved full-budget controls make zero new predictions. These repeat one
authored source shape. Model trials use the existing own FP32 model unchanged:
74,624 resident tensor bytes, metadata SHA `e9d7f4770d4d8402c64eea116028e6d6c049b1522f50f399c05f2c3571932a3b`,
weights SHA `f5af1e35288cbad938d8416dd51873d83dff369a0e0c6e7e16f6e1f75b7f429b`.
The named field-ordinal context is a transfer from the frozen integer model.
Case inputs and expected outputs are excluded from the prediction context.

| Attempts allowed | Selection fields, both modes | Selection cases, both modes | Compiled record fields, both modes | Compiled named outputs, both modes |
| ---: | ---: | ---: | ---: | ---: |
| 1 | 6/15 (40%) | 2/5 | 9/21 | 6/14 |
| 2 | 9/15 (60%) | 2/5 | 13/21 | 6/14 |
| 4 | 12/15 (80%) | 2/5 | 17/21 | 6/14 |
| 8 | 15/15 (100%) | 5/5 | 21/21 | 14/14 |

All three fresh trials per cell retained those counts. With four attempts, the
model selected mask 5 while deterministic search selected mask 3, completing
different fields with equal counts. Both ranked the complete mask last and
checked eight candidates at full budget. The field metric shows gradual progress
while the whole-case metric stays unchanged until the final field is filled.

Full-budget medians, three fresh trials per mode:

| Observation | Own model | Deterministic |
| --- | ---: | ---: |
| Local prediction | 29.0µs | 0 calls |
| Complete graph generation | 9.572ms | 9.596ms |
| Whole command | 349.36ms | 342.38ms |
| Peak process RSS | 82.00MiB | 82.33MiB |
| Process user + system CPU time | 0.26s | 0.26s |

Process resources include native Go builds and children. Standard-library/build
cache state reflects earlier development; mode order alternates after the first
pair. Wall time and small differences include host noise. These process values
leave host CPU utilization changes unmeasured. No attempt reduction or model
quality improvement was observed on this source shape.

`metrics.json` retains all 30 rows. Each fresh JSON keeps the complete source,
generated code, selected fields and actual runtime values. Each full-budget
composition has a saved replay and reusable Gooo checkpoint. The Go observer
recomputes counters from actual expected values and checks agreement between the
five overlapping interpreted and compiled cases.
