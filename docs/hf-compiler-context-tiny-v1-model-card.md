---
license: mit
language:
  - en
  - ko
tags:
  - gooo
  - metaprogramming
  - tiny-model
  - ternary
  - go
  - experimental
---
# Gooo Compiler Context Tiny v1

An independently initialized 12728-parameter model for bounded Korean/English
judgments over compiler-declared Gooo typed alternatives. No Laya/pretrained
weights are inherited. Go inference/runtime orchestration; offline local MPS
optimization only. This is a custom small MLP bundle, not a Transformers causal
language model or general text/code generator.

Feature ABI `split_context_intent_ngrams_v2`, 256 input features, 48 hidden units,
8 labels, maximum 512 UTF-8 bytes (no silent truncation). Semantic input context
`gooo/compiler-typed-path-context/v2` includes source-relative alternatives.
The compiler binds the original source, ranks legal paths with optional inference,
tests finite candidates and emits typed Go. Disconnection/overflow retains the
existing deterministic finite search and partial completeness receipts.

Published `models/{fp32,ptq_ternary,qat_ternary}` are the compiler-context arm.
The matched caller-context control, datasets, training sources, raw receipts,
negative first collection and observations are retained in `evidence.tar.gz`.

On the reused 320-view synthetic development cohort, compiler-context FP32
initially completes 240/320 whole finite programs; max one additional candidate
completes 320/320 and 3840/3840 finite cases. It needs 80 extra candidates vs 160
for disconnected fallback; the matched caller-context control needs only 68.
Compiler-context QAT needs 101 and PTQ 143. This is a **negative comparison with
the matched caller-context control**, not an accuracy improvement. Models remain
explicit experimental choices; no automatic default promotion is claimed.
Sparse targets retain ambiguity; no new independent intentions or untouched
benchmark are claimed. See `results.md` and machine-readable reports.

Measured compiler-context Go median inference: FP32 9.583us, PTQ 10.792us,
QAT 10.458us. These are local isolated model calls, not whole compiler latency.
Actual native dogfood retained 160 generation calls, 120 model predictions and
20 compiled-Go executions. Model workspace 1248 bytes; ternary payload 2759
bytes, decoded resident tensors 12896 bytes plus scales. Five trits/byte gives
1.6 disk bits, not packed runtime arithmetic or whole-process RAM.

## Use with Gooo

Compiler implementation and evidence:
[meta-ontology-go PR 1133](https://github.com/kimjooyoon/meta-ontology-go/pull/1133).
Pinned training/export native revision `7d8768f86158022b494603e22e2589d121dee0b1`.
SDK `github.com/kimjooyoon/gooo-decision-runtime` v0.2.9-experimental.

```sh
gooo body-context --plan plan.json --activity ChoosePath source.gooo
gooo body-codegen --json --path-plan plan.json --path-model models/fp32/model.json --activity ChoosePath source.gooo
```

Load the complete paired metadata/weights directory. The SDK verifies weight
hashes and feature ABI. The model only supplies ranked hints for declared paths;
source authority, type checks and authored finite tests remain in the compiler.
Do not reinterpret old feature-version weights with a new encoder.

Public source, protocol, reproduction and raw observations:
[gooo-neural-decision-experiments](https://github.com/kimjooyoon/gooo-neural-decision-experiments).
All uploaded training inputs are deliberately generated synthetic public Gooo.
Model limitations include one-hop context, sparse observational ambiguity,
Korean/English disagreement and context-control regressions. No universal
semantic correctness, Laya compatibility, hosted service or general natural
language-to-Gooo generation is claimed.
