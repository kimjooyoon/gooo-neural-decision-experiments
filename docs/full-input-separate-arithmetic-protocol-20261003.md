# Explicit arithmetic for full-input Gooo judgments

Registered before collecting new frozen-model predictions. This follows the
[retained platform diagnosis](full-input-numerical-portability-followup-20261003.md).
The previous comparison and its failures remain available at their original paths.

## Contract and controlled change

`arithmetic_version: float32_separate_v1` selects three-choice kernels that
round each product to float32 before adding it, and round each accumulated sum.
Ternary scaling is rounded before adding bias. Iteration order, ReLU, temperature
and the existing float64 softmax remain the same. The
[Go specification](https://go.dev/ref/spec#Floating_point_operators) permits
fusion across statements and defines explicit conversions as rounding barriers.
This is the mechanism being investigated; the complete observed platform delta
still needs the controlled model comparison.

Omitted arithmetic metadata retains the original kernels. Expanded V3/V4 and
compact three-choice loaders accept the explicit version and reject unknown
versions. Two-choice loading retains its prior arithmetic contract. Compaction
and initial/feedback receipts preserve the declared identity. Legacy JSON fields
remain omitted, preserving the existing receipt/metadata representation.

Weights, scales, temperatures, source features, complete input texts, source
digests, finite expectations, form membership and training checkpoints are fixed.
There are zero optimizer updates. No input shortening, rank epsilon, score
rounding heuristic or weaker integer comparison is introduced.

## Collection and acceptance

- Read all 36 retained development journals: four arms, three variants, three
  forms, 512 views each; 18,432 matched rows. Validate the closed public bundle
  and bind every journal/model to its published digest.
- For each row, invoke legacy expanded, legacy compact, explicit expanded and
  explicit compact models: 73,728 actual predictions per platform. Keep all four
  hidden/logit/probability vectors, input/source hashes, full text, selected mask,
  complete ranked order, partial coverage and first complete candidate position.
- Compare expanded/compact arrays and predictions exactly within each arithmetic
  version. Compare full feature arrays across versions exactly. Preserve every
  old/new difference, including unchanged and worse finite outcomes.
- Record each new model metadata/weight digest. Metadata changes the arithmetic
  identity; matching weight files must remain byte-for-byte equal. Converted
  artifacts are separately named and never overwrite original training exports.
- Run independent arm64 and Linux collection at a clean, fixed source revision.
  Computational source pins must match; later documentation/evidence publication
  commits may differ and remain identified separately in the comparison.
  Require identical explicit-version feature/hidden/logit bit patterns and exact
  mask/order/finite outcomes. Measure probability differences as well, with a
  maximum absolute difference of 1e-5 for the unchanged softmax. Exceeding that
  bound or any exact requirement fails the comparison and requires a follow-up.
- Keep the legacy platform difference as an observed control. The original
  exact legacy comparator remains unchanged. New-contract evidence has its own
  identity and acceptance result.

The initial scope is the previously observed development cohort. The order-alias
counterexample, additional forms/splits, native compiler adoption, new tasks and
issue 1023's language/discovery/completeness work remain part of the wider goal.
Unit tests use synthetic cancellation cases, malformed metadata, valid/invalid
requests, zero-allocation workspaces and concurrent calls; their predictions are
separate from the frozen-model collection count.

## Resource bounds

Use the existing training exports and original full-input observations. The
collector keeps one 512-row journal and four small models live at a time. Write
lossless gzip journals with a 64 MiB total retained-output cap per platform and
a 4 GiB free-space preflight. Use one invocation per collection, a 15-minute
external deadline and zero automatic retries. Additional output/storage use is
reported before publication. No new training environment or model download is
required for the local collection.
