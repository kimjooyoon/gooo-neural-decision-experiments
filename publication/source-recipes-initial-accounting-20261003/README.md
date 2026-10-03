# Initial collector accounting failure, retained

Collector commit `92c34aa` incorrectly required three model predictions per
request. The pinned model uses one joint prediction for all three decisions.
Its native receipt records `local_model_predictions: 1` and an actual valid
`three_choice_prediction`.

Collection stopped at `recipe-modeltrue-resolvefalse-r0` after five completed
generations and ten compiled runs. Four rows had been appended to
`processes.json`; the fifth generation and runtime files exist, but its row and
final manifest were never written because the collector assertion failed.

Commit `1486ef8` fixes that count. The successful fresh collection is in
[`source-recipes-20261003`](../source-recipes-20261003). These initial captures are
retained separately and excluded from its 24-generation aggregate. Model weights
and compiler behavior did not change between these captures.
