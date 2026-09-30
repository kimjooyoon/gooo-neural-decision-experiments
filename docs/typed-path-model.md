# Gooo structural path model

This is direct root development without subagents. It extends our metaprogramming
experiment from binary-operation selection to five structural choices:

| Choice | Compiler-owned edit |
|---|---|
| Local reference | Select one of two in-scope stored variables for an expression |
| Assignment target | Select which declared variable receives a computed update |
| Operand order | Preserve or exchange two fixed subtraction operands |
| Branch layout | Preserve or exchange complete `if` branch bodies |
| Root order | Choose a declared permutation of assignment statements |

The model returns a closed structural label. Names, statement indices and body
contents are supplied by the typed plan. Type/scope checking and deterministic
Gooo/Go rendering remain compiler-owned. Every individual offered option is
checked before inference; interacting selections are type-checked together
before execution. An explicit combined reference/order test demonstrates that
individually valid choices can be invalid together and must be rejected.

## Contracts and learning

Structural models use `gooo/tiny-path-decision-model/v1`, separate from the
existing operation ABI. Both use flat 256→48→8 arrays, but their label meanings
are distinct. Loaders and operation/path bridges reject the other contract.
The training comparisons transfer the previous FP32 hidden layer or initialize
independently, each with a new output head. Old operation predictions are not
renamed into structures. A versioned positioned feature experiment retains
coarse word order; its global classification performance regressed. Both the
collision diagnosis and measured regressions are preserved.

The Go curriculum contains 6,240 views: 4,800 training, 480 calibration and 960
test. The test views represent 320 original English/Korean instructions and 160
structural program configurations. Numeric configurations and instruction
templates are disjoint across splits. Plain, actual fallback Gooo activity, and
PROV-O vocabulary views of one instruction always share a split. The Gooo view
does not contain the gold-filled body. PROV-O conditioning is not OWL reasoning.

Existing Go tests compare all 640 family/direction/configuration programs with
independent arithmetic formulas, including int64 boundary and branch-boundary
inputs. These tests do not demonstrate model accuracy or native compiler replay.
The model audit separately records raw proposal correctness, selected label
correctness, acceptance, finite case correctness and typed program outcomes.

## Runnable assembly

```text
go run ./cmd/gooo-path-compose --plan <typed-path-plan.json>
go run ./cmd/gooo-path-compose --plan <typed-path-plan.json> --model <path-model.json>
go run ./cmd/gooo-path-compose --plan <typed-path-plan.json> --model <path-model.json> --seed <explicit-seed>
go run ./cmd/gooo-path-compose --plan <typed-path-plan.json> --model <path-model.json> --tests <finite-tests.json> --max-attempts 2
```

With no model, assembly uses declared fallbacks, makes no prediction/network
calls and is deterministic. Probability sampling is explicit and binds the
plan, choice, model metadata/weights and seed. Inference uses caller-owned fixed
scratch arrays. Typed-plan validation, rendering and report serialization are
separate allocating setup work. In explicit TDD mode the model predicts once per
decision before finite evaluation, then tests guide a bounded heap of declared
paths. Confidence abstention is observed, while finite acceptance is a separate
mode. Conditional eligible logits prevent unrelated global heads from wasting
the compiler's already-known choice kind. They are ranking weights, not calibrated
completion probabilities. Context deadlines and candidate limits are required.

Completion records include tested/untested candidate counts, type rejections,
case results and partial outcomes. Integer test documents cannot silently accept
Boolean results through a zero-valued integer field. Neither gold path labels
nor holdout cases are inputs to model prediction or the search API.

The reserved probe covers 640 new bilingual instructions in 1,920 views. FP32
ranking uses 2,468 candidate evaluations versus 2,880 without a model; final
unseen-input cases are 15,360/15,360 in each arm. Search median grows from
69.584 to 76.834 µs. See the model card for all variants, resources, failed
training attempt, conservative zero-acceptance result and reused development
evaluation disclosure. No arbitrary-body or general intent completion is inferred.

This Go assembler emits Gooo for the native compiler. It is an experimental
structural stage before native codegen; the native `--tiny-model` flag still
uses the operation ABI. Native in-process structural-provider integration,
training from arbitrary PROV-O feedback traces and production online learning
are not implemented by this stage.

The subsequent source-bound native replay is recorded in
`runs/typed-path-native-replay-bound-20261001`: 20 generations through the actual
compiler and 240/240 independently executed emitted-Go cases, repeating five
unique functions across languages/arms. Native replay uses saved selections and
makes zero additional predictions. The earlier runner revision typo and all
captures are retained in the neighboring run with an explicit exclusion/correction.

## Evidence handling

Two failed fixture checks are preserved in `preexecution/`: a missing input
name and a Go projection with an unused local. Both happened before model
training or native execution. The input was corrected; the unused-local fixture's
computation, instruction and oracle were changed together. Type checking was
not weakened. Training and audit outputs are written to fresh directories and
retain their own source, dataset, model and case denominators.
