---
license: mit
language:
  - en
  - ko
tags:
  - gooo
  - golang
  - metaprogramming
  - codegen
  - ternary
  - from-scratch
  - prov-o
  - experimental
---

# Gooo three-choice feedback tiny v1

**Publication phase: trained, Go kernel audited; full SDK/native study pending.**

Nine independent own small classifiers rank eight complete legal masks over
three typed Gooo body decisions. Architecture is 768/24/8 with 18,656 parameters.
This is structured semantic-context decision inference, not a token-generating
language model or general Korean/English understanding. A deterministic typed
compiler/search remains authoritative over legality, construction and tests.

No Laya, pretrained, frozen teacher or previous student weights initialized
these students. A frozen previous own FP32 model supplies actual training-failure
observations only. All three arms share one new Go-created seeded initializer.
There are 3,072 source views/1,536 bilingual groups from eight authored families.
Training uses 2,048 original views plus 7,667 unique observed failure contexts;
calibration and development each retain 512 separate original views.

The arms are uniform initial targets, passing-set initial targets and passing-set
targets with actual-failure contexts. Each source group has total weight one and
each language one half. All three FP32, PTQ and QAT variants remain, including
negative results. Offline local MPS performed 4,800 actual updates; every step
and epoch receipt is retained. Python is limited to offline optimization/export.
Go supplies full features, collection, inference, selection and runtime bridges.

## Artifacts and results

- `models/<arm>/<variant>/model.json` and `weights.bin`: all nine exports.
- `evidence/source-teacher.zip`: every original native/source export and actual
  teacher capture, including the first failed collection prefix.
- `evidence/training.zip`: all 642 prepared/trained/audit files, including the
  initializer, full feature matrix, 4,800 updates, 600 epochs and numerical parity.
- `publication-manifest.json`: every staged file's exact bytes and SHA-256.
- `training/`, `verification/`, `source/` and `provenance.jsonld`: scope and lineage.

FP32 calibration passing-set NLL is 1.822441/1.819579/1.811903 for uniform initial,
set initial and set feedback. QAT values are 1.798077/1.802863/1.798459; PTQ
values are 1.837039/1.836282/1.838565. Feedback FP32 improves this objective,
while uniform-initial QAT is slightly better than feedback QAT. PTQ variants
regress. These are calibration objectives, not development success percentages.
No default promotion or full SDK/native study result is included in this edition.

Actual Go kernel verification passed all nine models: 432 parity predictions
plus 18,018 warm/allocation probes. Maximum error was 1.081e-6 (bound 1e-5).
Warmed full-feature predictions took 16.505–16.874 microseconds with zero extra
heap allocations on this recorded host. Each ternary file is 3,854 bytes;
decoded tensors are 18,752 bytes plus 8 scale bytes and a 3,200-byte request
workspace. FP32 stored/resident tensors are 74,624 bytes. Five-trit packing is
nominally 1.6 stored matrix bits, not 1.58-bit process or inference RAM.

Training-loop intervals sum to 27.561 seconds, CPU 60.09–76.77% of one core,
sampled MPS tensors 30.63 MiB and driver 80.70 MiB. Lifetime process RSS peaked
at 1.09 GiB; it includes Torch/corpus/optimizer/export state. GPU utilization and
host CPU causal change were not measured. Kernel timing does not demonstrate
an overall codegen or FP32-to-ternary speedup.

## Go execution contract

Use `github.com/kimjooyoon/gooo-decision-runtime@v0.2.13-experimental` with Go
1.27.1. The `jointdecision` package exposes `LoadThree`, `EncodeThree`,
`ThreeWorkspace` and `ThreePrediction`. `LoadThree` accepts these model files.
Each input must contain exactly three complete canonical source-v3 semantic
contexts, each at most 512 UTF-8 bytes, in declared order. The `gooo;joint3|`
length-prefixed frame is at most 1,600 bytes. Use the compiler to export those
source-bound contexts; do not replace them with plain text or drop coordinates.

The current native compiler integration accepts the separate three-choice ABI
and can retain typed path progress. Unsupported arity, disconnected models and
full-context overflow preserve deterministic execution with zero prediction
calls. Models are immutable; workspaces and session state belong to each caller.
This edition has not yet exercised these nine students in the full planned
native study. Stock Transformers/Hub hosted inference cannot load this custom
Go weight format directly; no remote runtime service is supplied.

## Limitations and lineage

Authored wording and finite eight-mask spaces limit generalization. Complete
bounded enumeration is not arbitrary-language or natural-language completeness.
Only synthetic source/test/intent evidence and own model artifacts are included;
tokens, host paths and private repository contents are excluded from publication.
PROV-O records source, actual teacher observation, fresh initialization, matched
offline optimization and actual Go inference as separate activities.

[Source and ongoing research](https://github.com/kimjooyoon/gooo-neural-decision-experiments)
and [compiler](https://github.com/kimjooyoon/meta-ontology-go).
