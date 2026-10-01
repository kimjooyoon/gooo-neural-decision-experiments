# Observed partial construction across candidate budgets

The experiment's useful question is how much of the fixed contract remains
constructed after each bounded batch, and what judgment cost accompanies it.
It does not require a unique first prediction or perfect final completion.

The Go `tools/continuation-metrics` command reads the twelve frozen native views
already executed on feature and deployed main sources. It checks report and
capture hashes and makes zero new predictions, native calls or optimizer steps.
Each curve samples the best preserved result after 8, 16, …, 64 candidates.
Rejudgment-only duplicate points inspect no new candidate and are excluded from
the batch mean; their predictions remain in subsequent cumulative cost.

| Language | New model | First observed final-best batch | Mean observed batch completeness |
| --- | --- | ---: | ---: |
| English | FP32 | 16 candidates | 80.36% |
| English | PTQ | 8 candidates | 85.71% |
| English | QAT | 16 candidates | 80.36% |
| Korean | FP32 | 24 candidates | 82.14% |
| Korean | PTQ | 8 candidates | 85.71% |
| Korean | QAT | 16 candidates | 83.93% |

Feature and main editions reproduced these same curves. All end with 6/7
completeness: the seventh authored selection expectation is deliberately
inconsistent. One example, English FP32, improves from 3/7 at budget 8 with six
predictions to 6/7 at budget 16 with twelve predictions. A later feedback round
raises the total prediction count to eighteen while the best body is retained.
The raw point arrays record every measured batch and its actual cumulative calls.

These percentages refer to the finite cases in one existing compound function,
repeated across languages and model variants. The first observed best budget is
a batch boundary, not the exact first successful candidate. The mean uses eight
sampled endpoints, not continuous area or general natural-language accuracy.
No measurement establishes a causal feedback speedup against a paired rank-once
baseline for these new checkpoints. Unvisited paths might still improve a result;
the retrospective plateau is not permission to stop construction early.

[Frozen curves](../publication/feedback-continuation-curves-20261001.json)
and [native audit/source evidence](../runs/feedback-trained-native-20261001/audit.json).
The first test of the metric helper used exact float equality and failed on
ordinary rounding; the assertion was changed to a tolerance and rerun. That
authoring failure made zero model/native calls and changed no frozen observation.
