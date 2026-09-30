# Go decision library

The module exposes a small Go API at
`github.com/kimjooyoon/gooo-neural-decision-experiments/decision`. It wraps the
same bounded runtime used by the command-line tools. It is a tiny operation
classifier with typed output, not a Laya client or a source-code generator.
The model was independently initialized from synthetic operation examples; it
is not a Laya fine-tune.

For compiler dependencies, use the smaller independently versioned
[`gooo-decision-runtime`](https://github.com/kimjooyoon/gooo-decision-runtime)
module. Its production source is pinned to this repository's `e18908b` revision;
its module distribution excludes research records and model bundles. The
[public version consumer receipt](../publication/go-runtime-sdk-v0.1-public-consumer/README.md)
records a real public `v0.1.0-experimental` import with no local replacement,
three existing bundles, typed output, and concurrent prediction checks.

## Load and predict

The model metadata and weights remain separate files. Pass `Load` the path to a
variant's `model.json`; it reads the adjacent weights file, checks metadata,
file sizes and the declared SHA-256, and then decodes the model once.

```go
package main

import (
	"fmt"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/decision"
)

func main() {
	model, err := decision.Load("models/fp32/model.json")
	if err != nil {
		panic(err)
	}
	var workspace decision.Workspace
	var prediction decision.Prediction
	if err := model.PredictInto("Add the quantity and fee.", &workspace, &prediction); err != nil {
		panic(err)
	}
	fmt.Println(model.PredictLabel(&prediction), prediction.Confidence, prediction.Abstained)
}
```

`PredictInto` accepts non-empty valid UTF-8 instruction text up to 512 bytes
and fills a `Prediction`. `PredictLabel` maps its top index to one of the fixed
labels `add`, `subtract`, `multiply`, `less_than`, `less_equal`, `equal`,
`and`, or `or`. `Labels` returns a copy of this list. Callers must check
`Prediction.Abstained` and apply their own allowed-choice policy; a global
label outside an application's declared candidate set must not be used.

`Decide` validates the request schema and operand identifiers/types, and only
returns typed binary IR when the predicted operation is compatible with those
operand types and the confidence threshold. `BuildTypedBinary` and
`AssembleGoExpression` use the same closed operation and identifier checks.
The API accepts no source fragments and does not execute generated code.

## Concurrency and memory

After `Load`, a model is read-only and may be shared by concurrent workers.
Each simultaneous `PredictInto` or `Decide` call needs its own `Workspace`;
sharing a workspace concurrently is unsafe. `WorkspaceBytes` reports the
fixed scratch arrays per worker (1,248 bytes on the current architecture).
`PredictionBytes` reports the complete prediction struct size for the current
architecture; its logits and probability arrays total 64 bytes.

`PackedFileBytes`, `ResidentTensorBytes`, `MatrixTensorBytes`,
`BiasTensorBytes`, and `MatrixScaleBytes` report model-file or tensor storage
for the loaded variant. These are model accounting values, not total process
RSS: they do not include Go runtime overhead, metadata, allocator overhead or
file cache. Ternary variants also retain scale values separately from their
packed file representation.

Inference is synchronous Go code and needs no Python, network service, or
background worker. `Load` performs file I/O once; callers decide how to handle
paths and lifecycle. Model files are not embedded in the package.

## Provenance

Record `Model.Variant()` and `Model.WeightsSHA256()` with a decision. Those
values identify which local bundle made the prediction. They must not be
reported as a Laya provider or a Laya model revision. This library does not
send requests to Laya and does not apply training or holdout labels at
inference time.

The zero value of `Model` is invalid; construct a model with `Load`. Nil and
zero-value models fail closed. The public API is intentionally narrow and
keeps packed tensors and model metadata internals private.
