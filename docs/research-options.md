# Research options for the Gooo decision model

**Reviewed:** 2026-09-30

## Progress since this design — 2026-10-03

The proposal below describes the initial operator-classification experiment.
Development has since reached source-derived features, joint path choices and
2,072-parameter shared judges. SDK v0.2.15 supports full-input V4 features and
explicit arithmetic; the compiler currently uses v0.2.14 with V3 models.
The [current overview](../README.md),
[language direction](https://github.com/kimjooyoon/meta-ontology-go/blob/dev/docs/language-direction.ko.md)
and [public model](https://huggingface.co/asketeddy/gooo-shared-judgment-tiny-v1)
connect the original rationale to current results and remaining work.

## Original recommendation

Keep the deployable model as the small, independently initialized classifier in
this repository. It chooses among eight typed binary operations; Go then builds
the closed IR node. It does not generate arbitrary Gooo source. Use Laya as an
optional comparison model, and treat 1.58-bit quantization as a measured model
variant rather than a promised speed or memory improvement.

This keeps the deployed path small and auditable: the input feature mapping and
label order are defined by `model-contract.json`, and the Go runtime validates
the model metadata and weights. The model has no subword tokenizer. Its current
256→48→8 network stores 50,912 bytes of FP32 tensor data; report that separately
from file size and process memory.

## Training and evaluation path

The synthetic corpus currently has 2,048 rows: 1,536 train, 256 calibration,
and 256 test. Rows are expanded from operation-and-language template groups.
Keep all variants from one template in the same split; splitting individual
rows would let nearly identical instructions appear in both training and test.
The present 256-row test split is a pilot check, not a precise estimate of
performance on natural user language. Add independently written template
families and evaluate them as a later frozen holdout before making broad
generalization claims.

For the small classifier, a practical low-memory first run is CPU or Apple MPS:

1. Train on the 1,536 training rows only. Start with FP32 cross-entropy and a
   fixed seed. Use small batches (for example, 32 or 64) and stop based on
   calibration loss; keep the implementation deterministic where the backend
   allows it.
2. Fit one temperature on calibration logits only. Do not select architecture,
   thresholds, prompts, or quantization settings using the test split.
3. Freeze the model and report test accuracy and macro-F1 by operation and
   language, confusion counts, and calibration metrics such as Brier score and
   expected calibration error. Treat Go-emitted IR parity as a separate check:
   a correct class score does not by itself prove that the runtime constructs
   the intended typed node.
4. Train three comparable variants from the same initialization and split:
   FP32; post-training ternary quantization (PTQ); and ternary-aware training
   (QAT) using the same quantized forward pass and a straight-through gradient.
   Measure packed bytes, decoded resident tensor bytes, peak process memory,
   and latency separately. Keep the FP32 run as the reference.

PyTorch documents MPS as a GPU backend for macOS training. Use FP32 for this
small pilot, check operator support on the actual device, and record any CPU
fallback because it changes the compute path. MPS does not support float64.
[PyTorch MPS backend](https://docs.pytorch.org/docs/main/notes/mps.html)
· [MPS fallback controls](https://docs.pytorch.org/docs/stable/mps_environment_variables.html)

## Where Laya fits

Laya is a useful decision-model baseline: it scores a finite set of options and
returns a distribution, which matches the operation-selection stage better
than asking a generative model to emit code. Its multilingual checkpoint uses
mmBERT-base and has about 322 million parameters; the model card reports a
1,024-token default context and a roughly 647 MB checkpoint. The card reports
that its base multilingual checkpoint scored 0.342 on its own typed-decisions
benchmark, while its task-fine-tuned checkpoint scored 0.766. Those numbers are
not estimates for Gooo; they show that zero-shot Laya should be measured rather
than assumed to work for this label set. [Laya model card](https://huggingface.co/convaiinnovations/laya)

The published Laya fine-tuning notebook targets a two-T4 Kaggle environment
and fine-tunes its backbone. That makes a full local fine-tune a poor first
choice under a low-RAM constraint. The practical local baseline is to pin the
multilingual checkpoint and tokenizer revision, run inference on the held-out
synthetic examples, and record its predictions and calibration separately. A
frozen-encoder classifier with a small new head is a possible later experiment,
but it is a distinct method; do not describe it as the official Laya training
recipe. [Laya fine-tuning notebook](https://github.com/NandhaKishorM/laya/blob/main/notebooks/laya_finetune_typed_decisions_2xT4_kaggle.ipynb)

Laya's model card is Apache-2.0; its mmBERT-base backbone card is MIT. If a
future artifact contains Laya-derived weights or tokenizer files, pin the
upstream revisions and hashes, carry the applicable license notices, and
describe the derivative accurately. This repository's current tiny model is
independently initialized and contains no Laya weights or tokenizer.
[mmBERT-base model card](https://huggingface.co/jhu-clsp/mmBERT-base)

## What “1.58-bit” means here

BitNet b1.58 refers to ternary weights in {-1, 0, +1}; the information-theoretic
value is log2(3), about 1.585 bits per weight. The official BitNet method also
uses 8-bit activations (W1.58A8) and was trained with the quantization scheme.
The official Transformers guide says these weights are not quantized on the
fly: quantization is part of pretraining or fine-tuning. Microsoft publishes
packed inference weights separately from BF16 master weights used for training
or fine-tuning. [BitNet b1.58 paper](https://arxiv.org/abs/2402.17764)
· [Transformers BitNet guide](https://huggingface.co/docs/transformers/quantization/bitnet)
· [Microsoft BitNet repository](https://github.com/microsoft/BitNet)

The current tiny classifier experiment is narrower than that full method. Its
ternary QAT keeps FP32 master weights and optimizer state during training; its
Go runtime currently decodes ternary matrices into int8 and applies per-matrix
scales. Activations remain FP32. Describe it as ternary-weight QAT with FP32
activations, not as a BitNet W1.58A8 model or a 1.58-bit runtime. Five ternary
weights packed in base 3 use 1.6 bits per weight before scales, biases, and
metadata; actual latency depends on the inference kernel and device. A study
has explored ternary training in MLPs, but it does not guarantee gains for this
particular corpus or implementation. [MLP and BitNet quantization study](https://arxiv.org/abs/2411.05882)

Compare PTQ and QAT empirically on the same frozen splits. PTQ asks whether an
already-trained FP32 model survives ternary conversion. QAT asks whether
training with that constrained forward pass can recover task performance. The
result should report classification, calibration, runtime memory, packed size,
and latency; a smaller packed file alone is not evidence of faster inference.

## Public artifact and data handling

The current corpus is generated from synthetic English and Korean operation
instructions. Keep the public model limited to that dataset, the explicit
training provenance summary, model metadata and weights, the Go parity receipt,
and their source/contract files. Do not include raw training logs, local paths,
machine identifiers, credentials, repository-private source, or user examples.
The publisher accepts only the separately reviewed exact-file allowlist and
runs the Go public-export verifier before either a dry run or an upload.

An allowlist and a secret scan reduce accidental exposure; they cannot prove
that every training input has a right to be published. Keep future data
synthetic or explicitly licensed/consented, review it before training, and
publish its provenance and evaluation scope. For any future Laya-based model,
include model-card limits, upstream revision and license details. For this
closed eight-operation classifier, always say that unsupported operations and
arbitrary source generation are outside scope.

## Primary references

- [Laya model card and fine-tuning notes](https://huggingface.co/convaiinnovations/laya)
- [Laya two-T4 fine-tuning notebook](https://github.com/NandhaKishorM/laya/blob/main/notebooks/laya_finetune_typed_decisions_2xT4_kaggle.ipynb)
- [mmBERT-base model card and MIT license](https://huggingface.co/jhu-clsp/mmBERT-base)
- [PyTorch MPS backend](https://docs.pytorch.org/docs/main/notes/mps.html)
- [Hugging Face Transformers BitNet guide](https://huggingface.co/docs/transformers/quantization/bitnet)
- [Microsoft BitNet inference repository](https://github.com/microsoft/BitNet)
- [BitNet b1.58 paper](https://arxiv.org/abs/2402.17764)
- [When are 1.58 bits enough? A bottom-up exploration of BitNet quantization](https://arxiv.org/abs/2411.05882)
