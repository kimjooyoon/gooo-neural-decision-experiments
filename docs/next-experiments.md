# Next experiments and selection criteria

The published v1 is a bounded operator classifier. It demonstrates a Go model
bridge and small ternary storage, not a general natural-language compiler.
Keep the current dataset/test hashes and original results fixed; use new run
and dataset versions for further tuning.

## Current choice

The choice below records the original operator pilot. Compiler-path experiments
now prioritize bounded intent/structure judgment and iterative construction
completeness; first-shot accuracy is recorded rather than used as acceptance.
Current path results do not automatically promote a new checkpoint over earlier
broader regressions. The [native budget study](native-budget-study.md) separates
finite completion, evaluator-only inputs, bilingual agreement and iteration cost.

Use FP32 as the default for this pilot: 50,912 tensor bytes are already small,
and it was both more accurate and faster than ternary on this M4 run. PTQ is
an optional storage comparison. Do not promote QAT as an improvement: its
English `and`/`or` errors are worse, and its calibrated threshold accepted all
test rows despite those errors. Laya remains a richer encoder/decision baseline
for context-heavy operation choices; the v1 weights have no Laya lineage.

## Prioritized controlled studies

| Study | Change | Fixed comparison and decision measure |
| --- | --- | --- |
| Independent language holdout | New human-independent synthetic paraphrase families, ambiguous and unsupported requests | Freeze before training; group by intent/template; accuracy, macro-F1, abstention and unsupported false acceptance |
| Boolean repair | New training-only conjunction/disjunction templates, contrastive negation, operand-order examples | Keep v1 holdout untouched; report on a new final holdout rather than selecting by v1 test |
| Laya frozen encoder + Gooo head | Train only a small eight-operation head over pinned multilingual encoder features | Compare with tiny FP32 and zero-shot Laya on the same frozen inputs; encoder load/RSS and total latency included |
| Laya limited layer tuning | Unfreeze only final encoder layers after a head-only reference | Incremental MPS memory/time and held-out improvement; preserve upstream license/revision |
| Width and feature budget | 128/256/512 hash features and 16/32/48 hidden units | Fixed seed groups and template split; report Pareto frontier of bytes, latency and quality |
| Ternary kernel | Compare decoded int8, packed decoding during multiply and fixed-table zero skipping | End-to-end CPU latency and real resident bytes; include index/LUT memory and loader cost |
| Activation quantization | Quantize feature/hidden activation in a separate calibrated/QAT arm | Numerical parity, error by operation and measured ARM64 benefit; never label existing FP32 activation runtime W1.58A8 |
| Multi-node IR | Operation, condition, assignment, return and variable-binding choices under a declared grammar | Closed typed nodes, legal variable scope, compile/execute rate and held-out intention behavior |
| Incremental assembly | Root plan supplies a typed hole; worker selects one finite operation immediately | Per-hole ready-to-result latency, peak in-flight tasks, cancellation and completion-order behavior |
| TDD refinement | Feed only training-test failures back to the next bounded choice | Match attempt budgets against deterministic search; hidden tests remain unavailable to selection |
| CI context | Pin exact source/run and pass compact machine failure facts | Match zero/stale/current-context arms; measure added value independently of extra attempts |
| Load/backpressure | 1/2/4/8 workers and bounded queue capacities | Throughput, p50/p95 latency, RSS, CPU-seconds, no stalled completion or unbounded goroutines |

The multi-node IR study has now executed 128 frozen scenarios with the three
existing tiny models and actual Laya, with native Gooo and independent Go
execution. Its search sees training cases only and makes no feedback model
calls. Feedback-aware typed-path TDD and compact CI context have since been
executed in the [five-family study](native-feedback-families-study.md),
[interacting-path study](compound-path-study.md) and
[main candidate-budget comparison](native-budget-study.md). See the
[body capture](../runs/body-plan-v1-laya-7d626b9-20260930/README.md) for scores and
complete planned/observed denominators. The earlier persistent primitive stream
also has measured 1/2/4-worker batch evidence. The
[retained native worker](retained-native-study.md) now measures per-request whole
body latency, sequential/parallel scheduling and whole-child costs. Streaming an
individual unfinished body's holes is a separate unmeasured capability.

The table includes both executed bounded prototypes and future changes. The
linked records define each executed scope; encoder/head tuning, a packed compute
kernel and new language-generalization holdouts still require separate studies.
GPU work should begin with one small head/model arm at a time so shared 16 GiB
memory is not multiplied across worker processes. Parallel Go compilation/audit
work can proceed while that bounded GPU job runs. Runtime workers share one
read-only model and own their workspace.

## Completeness is a vector

Report planned and observed denominators for each dimension:

- Input/domain coverage: supported intent families, languages, templates and
  unsupported/ambiguous treatments.
- Structural coverage: well-formed typed IR nodes, identifier/scope correctness,
  successful deterministic assembly and compilation.
- Behavioral coverage: training and hidden examples actually executed; correct,
  incorrect, abstained, failed and unknown counts.
- Selection quality: first-choice accuracy, final bounded-search accuracy and
  calibration; match budgets before attributing improvement to model/context.
- Operational coverage: success/cancellation/backpressure cases, allocation,
  resident RAM and timing with clear measurement boundaries.
- Evidence coverage: data/source/model/receipt hashes and reproducible independent
  checks; distinguish synthetic, mock, replay and actual model calls.

A weighted scalar may be useful for ranking candidates only if all denominators
and unknown dimensions remain visible. Neither a high confidence score nor
256/256 finite tests proves correctness on all programs. Type checking and
deterministic compilation remain independent of probabilistic selection.
