# Bilingual Gooo semantic feedback direction

The user's latest direction prioritizes Gooo and Korean/English judgment,
incremental compiler dogfood and continuous public evidence. A high first-choice
score is not the primary acceptance requirement. Work is performed directly by
root, without delegated agents.

## Current working loop

1. The compiler supplies a typed structural arena and closed alternatives.
2. Our small model reads only the bilingual intent/context and ranks eligible paths.
3. Authored finite tests evaluate candidates within a deadline and work budget.
4. Accepted typed paths render Gooo and pass through actual native codegen.
5. Exact emitted Go is executed against independent arithmetic cases.
6. Source/model/input hashes, choices, failures, completeness and cost remain public.

With no model, the same loop uses deterministic declared order. Confidence,
finite case success and unattempted alternatives remain separate observations.
No human reviewer/Guardian is required by this experiment. Existing compiler
machine checks and type/scope correctness still supply execution constraints.

## Next iterations

- Enrich real Gooo judgments: references, assignment targets, branch conditions,
  sequencing dependencies and typed body composition across several decisions.
- Preserve language ambiguity and failed interpretations in feedback records.
  Learn from generated public cases and explicit compiler/test receipts; do not
  treat a repeated paraphrase or context view as a new independent experiment.
- Measure completion on stated requirements: passing/total finite cases, known
  type obligations, unresolved decisions and unattempted alternatives. These
  denominators do not become a probability of satisfying all possible intents.
- Reduce typed validation, repeated arena copies and rendering setup cost. The
  measured FP32 ranking saved candidates but did not reduce end-to-end search time.
  Cache only immutable validated state; interacting choices must still be checked.
- Extend PROV-O records connecting source/model entities, compiler/test activities
  and observed failure evidence. Existing vocabulary conditioning is not an OWL
  reasoner, and a provenance relation alone cannot justify a generated behavior.
- Publish small reviewed model/evidence revisions with unchanged historical
  failures and a fixed synthetic-only allowlist. No private environment or large
  upstream model directory is exported.

Typed-pair training is a possible later comparison, not part of this publication.
Current exported models still train the eight-label global objective. The TDD
runtime conditions ranking on the compiler-known pair and retains global
confidence observations. Native in-process structural inference is now available
on compiler `main` through `--path-plan [--path-model]`; see the
[actual main promotion and smoke](../publication/native-typed-path-main-promotion-20261001.json).
The current compound probe preserves short-budget failures, partial finite
contracts and resource costs. Production online learning remains future work.
