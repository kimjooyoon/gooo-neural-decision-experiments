# Isolating the order experiment from frozen model computation

The initial order preflight at `6ad2438` passed its Linux feature/model replay.
The separate-arithmetic replay in the same CI run stopped with
`computational source/model/protocol differs` before its platform comparison:
[original job](https://github.com/kimjooyoon/gooo-neural-decision-experiments/actions/runs/37086762285/job/111098578233).

The fixed arithmetic contract inventories complete computation packages. Adding
`intent_order_sketch.go` and its tests to `internal/decision` changed that
inventory, even though the frozen model never called this experimental feature.
The inventory check correctly retained a source difference.

The experiment now lives in `internal/intentorder`, with no runtime dependency
on a model or semantic-source decoder. Its `Into` function accepts bounded UTF-8
intention text, uses the same clause-edge calculation and writes the same fixed
128-byte counters. The collection tool still calls the original semantic encoder
to validate and frame the fixed source fields separately. All frozen model
computation files return to their original bytes and paths. The inventory and
comparison checks are unchanged.

Original preflight records, benchmarks and the failed CI collection are retained.
The original benchmark included frame validation; the isolated raw-text kernel
has a different measurement boundary. CI replays every published counter,
reference, input digest and frozen-model prediction, excluding only measured
prediction duration. The repeated-excursion collision remains an explicit test.

This change keeps an untrained feature experiment separate from a released model
contract. Adding it to a trained model still requires a new explicit input
contract, training evidence and generated-program observations.

Local verification at clean source `684620c2527e3b783f2c918121f8a5ab2bd8990b`
replayed all 32 preflight pairs with 64 real old-model predictions. Every counter,
reference and prediction matched the published values after removing timings.
A fresh arithmetic collection then made 73,728 predictions over 18,432 inputs
under its four existing model/layout lanes. Comparison with the original arm64
archive passed, including exact computation source/model/protocol identity and
zero changed rankings or logits. This is a same-platform repair check; Linux
replay is tracked in the next CI run. No training updates were performed.
