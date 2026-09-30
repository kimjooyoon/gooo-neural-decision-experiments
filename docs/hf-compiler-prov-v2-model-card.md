---
license: mit
language:
- en
- ko
pipeline_tag: text-classification
tags:
- gooo
- prov-o
- metaprogramming
- ternary
- custom-code
---

# Gooo compiler and PROV-O tiny decision model v2

A 12,728-parameter 256→48→8 MLP fine tuned from
[gooo-ir-operator-tiny-v1](https://huggingface.co/asketeddy/gooo-ir-operator-tiny-v1).
Go performs inference and typed IR assembly; the existing offline MPS/PyTorch
environment trains the model. This custom bundle is consumed by Gooo/Go code.
It is not a Transformers generation model or a hosted inference endpoint.

## Scope

The eight outputs are add, subtract, multiply, less-than, less-or-equal, equality,
AND and OR. The compiler owns source intent, type checking, candidate eligibility,
finite TDD and deterministic fallback. The model supplies one bounded proposal.
It does not create arbitrary bodies, prove full-domain behavior, or reason over OWL.

[PROV-O](https://www.w3.org/TR/prov-o/) is used to condition synthetic instruction
views and to record actual source/model/result entities, generation/verification
activities and the compiler software agent. The bundled Turtle traces describe
the successful native executions; vocabulary conditioning is not ontology reasoning.

## Training and evaluation

6,146 rows: 4,610 train, 768 calibration and 768 test. Each original instruction
has a plain intent, Gooo declaration and PROV-O context view; all views of one
template retain the same split. The 768 test rows represent 256 original held-out
instructions, not 768 independent tasks. Two previously failed native intents
were added to training and receive 128 optimization exposures per epoch each.
Their repair score is separate from the held-out score.

| Variant | Previous model on new test | New model | Weight file |
|---|---:|---:|---:|
| FP32 | 624/768 (81.25%) | 768/768 (100%) | 50,912 B |
| PTQ ternary | 584/768 (76.04%) | 643/768 (83.72%) | 2,759 B |
| QAT ternary | 586/768 (76.30%) | 768/768 (100%) | 2,759 B |

Independent Go/Python parity matches 96/96 sampled feature/logit/probability rows
within 1e-4. Go reproduces the listed 768-row scores. The PTQ selective result is
565/601 accepted (94.01%); its calibration target does not guarantee test precision.

On two disclosed repair-training intents under three variants, native model raw
choices improve from 0/6 to 6/6. Five of six new proposals are applied; PTQ abstains
on the arithmetic intent despite the correct top label. Both old and new final
programs pass 36/36 finite cases. A separate disconnected baseline passes 12/12
with zero model/provider calls. This is a repair regression, not unseen codegen
accuracy. Native TDD evaluates the declared candidates before prediction; no
reduction in candidate evaluations or throughput improvement is demonstrated.

## Timing and memory

The two 60-epoch MPS loops took 1.824 s and 1.323 s. End-of-epoch sampled MPS
allocation peaked at 5,880,320 B, driver memory at 53,182,464 B. Python process
lifetime RSS reached 471,531,520 B; this is training/tooling memory.

Go's short-prompt diagnostic kernel costs approximately 8.6 µs FP32 and 9.5 µs
QAT with zero bytes/allocations per prediction. Diagnostics use adaptive benchmark
loops and are separate from the native cohort. The six new native CLI processes
took 6.49–11.52 ms, including 37.6–63.4 µs provider decisions and model loading.
Maximum native child lifetime RSS was 16,924,672 B (16.14 MiB). Aggregate child CPU
was 48.952 ms over 55.255 ms native wall (88.6% of one core); this is not host CPU
utilization increase. Go replay CPU/memory is excluded from that resource cohort.
The first parent CLI observation of 472 ms is retained with no assigned cause;
the cohorts are not evidence of a wall-time speedup.

Packed matrix codes use 1.6 physical bits/weight (five ternary values per byte).
The QAT resident decoded tensors occupy 12,896 B plus 8 B of matrix scales;
per-worker scratch is 1,248 B. Biases, activations, metadata and process RSS are
separate. This is not a packed 1.58-bit inference kernel.

## Use

Download `models/qat_ternary/model.json` and sibling `weights.bin` from this repo.
With the Go 1.27.1 Gooo integration, pass the metadata to
`gooo body-codegen --fill-plan plan.json --tiny-model model.json --activity Activity source.gooo`.
Omit the model option for deterministic generation. Use the declared source/plan
types and retain the report's raw operation, applied flag, fallback and TDD score.

Source, dataset, failed attempts, repaired verification and raw native outputs:
[gooo-neural-decision-experiments](https://github.com/kimjooyoon/gooo-neural-decision-experiments).
The model ABI is compatible with `gooo-decision-runtime` v0.1.1-experimental.
Only public synthetic text and the two disclosed regression intents were used;
credentials, host paths, environments and private repositories are excluded.

One seed and a synthetic template split are reported. No independent natural
language workload, unseen operator family, formal intent-completeness guarantee,
or automatic retraining in a production compiler is claimed.
