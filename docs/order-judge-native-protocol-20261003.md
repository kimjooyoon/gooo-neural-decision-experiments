# Whole-candidate judge inside Gooo: native protocol

Follow the fixed initial model and actual search replay with direct in-compiler
invocation. This protocol is committed before native observations begin. Retain
weights SHA256 cf00ccc83d17d28ed73fcb869366151a48ffccd3aa8ca8e635aabf19810b9e78
and all earlier failures. No fitting or quantization changes are planned here.

## Requests and controls

Use all 64 requests in the existing `new-template` and `new-constants` development
groups: four arithmetic families, English/Korean, both source orders and both
desired orders. These have already been observed; this is an integration and
behavior reproduction study, not a new general-language benchmark.

For each request, run deterministic and explicit whole-candidate-model arms
at budgets 1 and 8, giving 256 generations. Enable exact descriptor reuse for
the new model. The standalone deterministic control already showed unchanged
outcomes with reuse; ordinary compiler deterministic search remains the control.

Use the original three selection inputs -2,0,3 and the original eight evaluation
inputs -3 through4. Five evaluation inputs are absent from the selection suite.
Every generation is followed immediately by a clean native build and two executions
of that generated result before moving to the next arm. This gives 512 native
runs if the cohort completes. Counts remain planned until observed.

## Actual call order

Original Gooo and recipe → typed expansion → source binding → local model load
(or retained immutable model) → whole-candidate prediction → bounded finite
candidate execution → selected Gooo/Go emission → immediate build and execution.

Do not encode a precomputed choice in a seed or replace the original source with
an externally selected body. The native compiler must load the actual model.
Expect one new-model prediction per supported model generation and zero in the
disconnected arm. Its ranking input is the complete original intent and actual
candidate operations; evaluation outputs are absent from that input.

## Measures and implementation checks

Record source/recipe/model/compiler/SDK identities, every ranking, model-call
count, selected mask, attempted bodies and equal-descriptor skips. Compare native
outputs with the independent authored arithmetic reference, including budget-1
failures. Report request-complete and finite-case-complete counts separately.
Compare all expected results with the frozen standalone cohort; preserve any
representation or semantic mismatch instead of tuning away the failure.

Record prediction-only, whole-generation and native build/run timing separately.
Resource measurements describe the measured process and its children, not global
CPU utilization. Keep all raw observations with the clock/resource scope stated.

Unit coverage includes source mismatch before model access, bounded artifact
loading, unsupported explicit options, optional-model determinism, finite partial
results, source/receipt replay, cancellation and parallel retained-model callers.
Unsupported model profiles or features require an explicit decline/error record;
no silent truncation or ignored sampling/feedback options.

The prior released models and frozen V3/V4 packages remain unchanged. Existing
observation-driven direct resolution can still skip model work when source-bound
observations uniquely determine the body. Native use of that path has its own
recorded scope; do not count it as a prediction by this new model.
