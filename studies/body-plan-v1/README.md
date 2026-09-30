# Typed body-plan cohort v1

This directory freezes 128 bounded one-input `int64` program intents for the Gooo typed body-plan interface. The JSONL contains 96 `Int` results and 32 `Bool` results across eight families: arithmetic pipelines, threshold comparisons, Boolean composition, local reassignment, piecewise branches, nested branches, polynomial expression order, and signed-boundary arithmetic.

Each row includes a plan, a gold operation assignment, training examples, and heldout examples. Every case uses a typed expected value. Training and heldout inputs are disjoint for each plan; each plan has at least 7 training and 10 heldout inputs. Training includes signed extremes and values near the constants and branch thresholds. Heldout inputs contain five negative and five positive values, including signed-edge probes and values around solvable branch boundaries when they are not in training. The generator verifies that the gold mapping is the only allowed mapping that matches all training results. Four plans have 81 allowed operation mappings, which exceeds a 64-plan search cap; this is recorded so bounded-search behavior can be reported separately.

The eight operations are drawn from the fixed typed operation set in `model-contract.json`. Plans use only an expression hole with a finite candidate list; they do not enable unrestricted code generation. The set tests multiple expression nodes, locals, reassignment, conditional branches, nested control flow, operand order, Boolean results, and signed overflow. It does not prove correctness over all `int64` values or describe the full Gooo language.

## Model input boundary

For each hole, a model may receive its natural-language instruction and descriptions of that hole's allowed operations within the required type context. Do not include the row family, IDs, gold operation map, fallback, expected cases, or training and heldout examples in the model prompt. A bounded search arm may inspect training cases only after the model returns its initial choices. Heldout cases remain evaluation-only.

## Reference and provenance limits

`generate.go` writes each expected value from a handwritten Go formula for that scenario. It separately interprets the plan only to check that the declared gold operations produce the formula's values and that training examples distinguish the finite candidates. The formulas do not depend on the body-plan evaluator or the Gooo emitter. They are still authored from the same intended semantics as the plans, so this is a correlated reference check rather than a fully independent language implementation or proof.

There is no random seed: the 128 plans, critical training inputs, and heldout pool are produced by a deterministic index-based schedule. Plan IDs and names are stable labels; changing an ID, variable name, prompt wording, or fallback does not count as a new intent. The canonical-intent rule in `manifest.json` alpha-normalizes local and hole names and preserves typed expression topology, operand order, constants, fixed operations, control-flow structure, allowed hole choices, result type, and gold operation assignment.

Each hole has a deterministic fallback selected from its allowed operations after skipping the gold operation. This makes the fallback an unassisted baseline, but it is deliberately wrong by construction; it is not a model prediction and is never part of the model prompt.

## Reproduce the frozen files

From the repository root, pass a new empty output directory:

```sh
go run ./studies/body-plan-v1 /tmp/body-plan-v1-replay
```

The generator refuses to overwrite existing cohort files or write into a nonempty output directory. It locates and hashes its own source independently of the output path. The manifest binds the generated JSONL and source hashes and records dimensions and coverage counts. Before writing, the generator checks structure, typed oracle agreement, and disjoint inputs; it compiles all 1,188 allowed operation mappings with the native `bodyplan.Compile`, verifies native evaluation against handwritten gold formulas on every training and heldout case, and exhaustively checks that the training cases distinguish every non-gold candidate mapping.
