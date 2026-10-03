# Small source-conditioned candidate judge: fixed pilot protocol

This protocol precedes its first model fit. The existing V3/V4 model files,
computation inventories and observations remain frozen. The new experiment uses
an isolated `orderjudge` package and a separately named feature/model contract.

## Question and boundary

Can a small learned match between complete Korean/English intent and the two
operations in each permitted candidate reduce additional construction attempts?
The input observation pilot established that V3 root inputs alias across opposite
source orders. Its 48-byte descriptors retain the distinction. The new judge
scores all eight fully composed candidates, including the interacting operand
and root-order choices. It does not multiply independent local confidences.

The admitted profile is `let v=input; v=op(...); v=op(...); return v`, with one
root-order choice and two operand-order choices, exactly three binary decisions.
Integer add/subtract/multiply and local/input/literal leaves are covered.
Candidate descriptions canonicalize commutative operands and preserve subtraction
orientation. Unsupported profiles decline explicitly. The small numeric feature
projection initially accepts constants from -16 to 16; the exact descriptor
retains full int64 values, and values outside the model projection are rejected.

## Model and fixed training configuration

- 128 complete-intent byte-ngram features in four relative-position buckets.
- 32 source features, 16 for each ordered operation: presence, opcode, leaf kinds,
  signed constant magnitudes and negative/zero indicators.
- One 128×32 bilinear matrix, 4,096 trainable parameters / 16,384 FP32 bytes.
- Scores depend on candidate content. Softmax normalizes across the eight
  declared structural masks. Multiple masks can describe the same function.
- Multi-target loss uses the probability mass of every candidate satisfying the
  authored training reference on the declared finite inputs. Labels and observed
  outputs are excluded from inference features.
- Zero initialization, full-batch gradient descent, 400 epochs, learning rate 2,
  L2 coefficient 0.0001. No held-out selection of epoch or learning rate.
- Go implements training, inference, data orchestration and verification.
  This very small matrix uses local CPU training; GPU training is not claimed.
  Inference uses explicit FP32 product/add rounding and stable mask tie order.

## Authored cohort and development split

Four operation pairs: add-one/multiply-two, subtract-three/multiply-two,
negate/add-four, square/add-one. Both source orders and both desired orders occur
in English and Korean. Two authored training templates give 64 training requests.
Selection inputs are -2, 0 and 3; labels come from separately authored arithmetic.

Evaluation comprises three declared development groups, each with 32 requests:

1. A third instruction template, absent from training, on the original operations.
2. Renamed locals and commutative source rewrites on original operations, using
   the first training template. Both changes occur together in each request.
3. New constants: add-two/multiply-three, subtract-one/multiply-four,
   negate/add-two, square/add-three, using the first training template.

Families and templates are authored in advance. These are controlled development
contrasts, not an untouched general-language benchmark. Evaluation does not feed
back into fitting. Report every group and all failures. Selection-disjoint native
inputs are -3, -1, 1, 2 and 4; the combined native suite is -3 through 4.

## Reporting and follow-through

Pin compiler, source, expanded plan, feature version, weights and collector.
Validate each expanded plan against its source before feature construction.
Preserve the full instruction, all candidate descriptors, logits, probabilities,
ranking and functional outcomes. Report first-candidate complete requests,
finite cases met, attempts to complete with budgets 1 and 8, source order pairs,
inference cost, model bytes, workspace size and process costs separately.

A standalone fit and interpreter evaluation establish only those boundaries.
Native Gooo integration must call the new judge after source binding and before
finite candidate tests, replay its exact decisions, and compile/run the resulting
body. The compiler's model-free path must remain deterministic. New runtime/model
artifacts may be published as experimental while that integration is unfinished;
do not describe a precomputed seed or exported body as an in-compiler prediction.
