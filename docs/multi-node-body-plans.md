# Authored structure and model-filled operation holes

The new experimental path separates a preauthored logical structure from its
bounded operation decisions. The plan supplies an indexed expression arena and
statement tree: typed variables, assignments, branches, returns and operation
holes. A tiny model selects an allowed operation for each hole; deterministic
validation and emission produce the complete Gooo and Go bodies. Models cannot
add statements or return arbitrary source.

`gooo-body-compose` accepts the plan on stdin. With no `--model`, every hole uses
its declared fallback. With a model, valid high-confidence global choices are
retained only if they belong to that hole's typed candidate set; other results
record deterministic fallback. `--sample-seed` explicitly samples normalized
allowed probabilities and records the seed digest. Sampling is reproducible,
but neither the conditional distribution nor the global model confidence is a
guarantee of whole-program correctness.

```sh
go run ./cmd/gooo-body-compose < authored-plan.json
go run ./cmd/gooo-body-compose \
  --model runs/pilot-mps-20260930-v1/models/fp32/model.json \
  < authored-plan.json
```

Optional `--training-cases` performs bounded deterministic enumeration after the
initial selection. It keeps the initial choice on ties and stops on full
training success, candidate exhaustion or the attempt cap. It sees training
cases only. The receipt exposes first proposals, emitted choices, attempts and
finite scores separately. This is model-assisted finite search, not model
learning from CI context. The earlier Laya/native TDD studies cover a separate
feedback-aware selection experiment.

Arrays cap expression and statement counts at128 each, holes at16 and combined
depth at16. Model weights remain shared/read-only; expression IDs avoid a
heap object per IR edge. Compiled plans own copies of their arrays and choice
maps so later mutations and concurrent evaluation cannot alter a compiled
program. Unlike the primitive prediction kernel, full validation, go/types,
source formatting and interpreter environments allocate. Pipeline throughput
must be measured separately from the earlier zero-allocation prediction result.

Completeness remains a vector: declared structural coverage, typed generation,
first-choice operation agreement, training behavior, heldout behavior, actual
compiled execution, and observed/unknown operational outcomes. A high finite
score does not prove unrestricted natural-language or full-domain correctness.
