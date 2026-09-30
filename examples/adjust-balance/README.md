# A small authored body with three model decisions

Intent: start with `input + 3`; if the input is negative, decrement that value;
otherwise increment it. The plan declares the variable, assignment and branch
structure. Three typed holes select the initial operation, the condition, and
the negative-branch operation. The positive-branch operation is already fixed.

From the repository root:

```sh
go run ./cmd/gooo-body-compose < examples/adjust-balance/plan.json
go run ./cmd/gooo-body-compose --model runs/pilot-mps-20260930-v1/models/fp32/model.json --training-cases examples/adjust-balance/training.json < examples/adjust-balance/plan.json
```

The first command uses only the declared fallbacks. The second makes three tiny
model predictions and optionally searches the eight allowed operation mappings
using the four declared training cases. No Laya, network, Python, or CI is needed.
The command emits Gooo and Go source as JSON data; it does not execute that source.
The four public examples are training cases, not an independent heldout score.

With a downloaded v0.2 release, replace the commands with
`bin/gooo-body-compose`, and the model path with `models/fp32/model.json`.
These authored example files live in the repository and are not release assets.
