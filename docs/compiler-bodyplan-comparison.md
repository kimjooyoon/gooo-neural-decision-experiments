# Same Gooo bodies, parent versus compiler/PROV model

The two previously published repair intents were too narrow to select a model
for broader Gooo development. This follow-up uses the existing frozen 128
programs and eight families with the same Go 1.27.1 executable and compiler
revision `68361e64def5457f0d0e6de972570a9885cceb96` for both bundles.

## Direct model selection

| Model | Parent cases / 1,280 | New cases / 1,280 | Change | New complete finite programs / 128 |
|---|---:|---:|---:|---:|
| FP32 | 950 (74.22%) | 1,086 (84.84%) | +10.625 percentage points | 103 |
| PTQ | 679 (53.05%) | 638 (49.84%) | −3.203 percentage points | 52 |
| QAT | 916 (71.56%) | 768 (60.00%) | −11.563 percentage points | 64 |

FP32 is the current recommendation for this measured workflow. The tiny QAT
artifact is useful as a quantization experiment, but its perfect synthetic
instruction-view test score does not transfer to these body decisions.

The raw model operation is correct in 165→189 of 222 holes for FP32, 133→124 for
PTQ, and 160→139 for QAT. PTQ fallback rises from 56 to 94 holes and lowers its
selected correct operations to 106. QAT's raw proposal itself regresses, so
changing only an abstention threshold would not address every wrong proposal.
These observations identify what changed, not a causal attribution to one
training hyperparameter.

## Search and completeness

Training-only finite search, capped at 64 attempts, reaches 1,280/1,280 held-out
function cases and all 128 finite programs for every model and the deterministic
baseline. It does not make more model calls. Direct deterministic fallback
scores 384/1,280 with zero correct declared oracle operations; its intentionally
different operations can still pass some cases through branches or cancellation.

Each bundle makes 666 local predictions, emits 1,024 native Gooo cells under
eight arms, and independently executes 36,160 generated-Go cases. Across both
captures, 72,320/72,320 planned executions are observed, with zero unknown.
This is 128 distinct intents repeated under arms, not 2,048 distinct experiments.
Held-out refers to disjoint function inputs; the synthetic authored intent corpus
does not establish independent real-world natural-language generalization.

Selection happens in the Go typed body assembler before native compilation.
The native compiler emits the already assembled Gooo; this study does not invoke
native `--tiny-model`. The preceding six-cell study separately exercised that
native optional provider. This distinction matters when counting predictions or
describing integration depth.

Two bounded runners execute concurrently without agents. They each finish in
about 30 seconds with stable binary hashes and no recorded timeout. Concurrent
pipeline wall observations do not establish a throughput speedup or absence of
all possible deadlocks. Per-child CPU/RSS is unmeasured in this broader study.

## Evidence

- [Preexecution declaration](../runs/compiler-prov-v3-bodyplan-20261001/preexecution.json)
- [Matched comparison and family metrics](../runs/compiler-prov-v3-bodyplan-20261001/comparison.json)
- [Parent capture](../runs/compiler-prov-v3-bodyplan-20261001/parent/report.json)
- [New model capture](../runs/compiler-prov-v3-bodyplan-20261001/improved/report.json)
- [10,315-file digest manifest](../runs/compiler-prov-v3-bodyplan-20261001/artifact-manifest.json)

The comparison's `PASS` means matched, complete captures and validated
denominators; it does not mean all model outputs were correct. Every incorrect
case remains in the raw output. The original narrow model bundle and earlier
failed harness attempts remain unchanged.

## Next training objective

Gooo development needs data connecting intent to typed operands, assignment
dependencies, branch conditions and subplans. PROV-O should bind each observed
proposal and counterexample to the exact source, plan, model and verification
activity. Merely adding ontology vocabulary to instruction text is insufficient.

A follow-up training set should group duplicate intent templates across all
programs before splitting, distinguish observed repair examples from fresh
evaluation programs, and select checkpoints with a held-out compiler-path
calibration set. Report raw operation correctness, acceptance, post-TDD finite
coverage and unknown obligations separately. Quantized checkpoints must meet
the same body-level criteria as FP32 before becoming the recommended model.
This follow-up objective is not presented as completed training in this record.
