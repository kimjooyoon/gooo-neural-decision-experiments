# Compiler and provenance model pilot

This pilot fine tunes the existing 12,728-parameter operator classifier. Go owns
dataset construction, inference, compiler execution and evaluation. The existing
offline PyTorch environment performs the bounded MPS training step.

The curriculum contains 6,146 rows: 4,610 training, 768 calibration and 768 test.
The original 2,048 instructions each have a plain intent, Gooo declaration and
PROV-O context view. Original template groups keep their original split across
all views. Thus the test set represents 256 original instructions in three views,
not 768 independent tasks. Two disclosed native compiler failures are additional
training repair cases and are reported separately from held-out accuracy.

FP32 starts from the public v1 FP32 weights. QAT starts from the new FP32 model.
The pilot uses a fixed seed, at most 120 epochs per trained variant, calibration
NLL for checkpoint selection, and calibration-only temperature/abstention fitting.
The default budget is 60 epochs per trained variant. The two repair rows receive
128 optimization exposures per epoch; this does not create new unique examples.

The output remains the eight-operation ABI. A model proposes a typed operator;
the compiler controls candidate eligibility, type checking, finite TDD scoring and
deterministic fallback. Model proposal correctness and final test correctness are
separate measurements.

[PROV-O](https://www.w3.org/TR/prov-o/) supplies the provenance vocabulary.
Compiler traces represent source/model/result entities, generation activities,
software agents, usage and derivation. Conditioning on this vocabulary is not an
OWL reasoner, a proof of arbitrary intent, or permission for a model to rewrite
source authority. Verification evidence supplies feedback labels; model choices
alone do not supply new ground truth.

The current packed ternary matrices use 1.6 physical bits per matrix weight.
Floating biases, metadata, activations and decoded resident arrays remain separate.
Existing model bundles, original failed captures and release versions are preserved.

Training, independent Go parity, native body generation and publication each have
their own receipts. A trained checkpoint is not claimed deployed until those later
stages finish. No model service, external inference endpoint or new Python runtime
is required by the deployed Go path.
