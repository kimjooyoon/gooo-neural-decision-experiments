# Shared local Gooo judgment, composed into three-choice path scores

This protocol is frozen before new optimizer work. It follows the existing
three-choice study and its poor bilingual initial agreement. The earlier
two-choice paired-JS experiment did not improve development continuation, so this
experiment changes the composition structure instead of repeating that penalty.

## Hypothesis and scope

One small source/intent judge reused at each of three typed decisions may learn
more transferable local choices than a dense joint classifier. This is a
hypothesis, not an expected improvement. Gooo still owns legal alternatives,
finite expectations, source binding and deterministic continuation. Independent
local probabilities cannot represent every joint dependency; retain failures and
same-wrong agreement explicitly.

Both arms use the already frozen, independently Go-verified 10,739 source-bound
rows and the original 768-column feature matrix. Train/calibration/development
groups remain 1,024/256/256. Development examples have already been observed;
this is an architecture comparison, not new independent intentions, unseen
generalization or an untouched holdout. No input is truncated or re-encoded.

## Two fresh matched arms

1. `dense`: the existing 768 -> 24 -> 8 ReLU MLP, 18,656 optimizer parameters.
2. `shared-local`: the same 256 -> 8 -> 2 ReLU judge reused over all three
   ordered source/intent parts, with 2,072 optimizer parameters. The last local
   layer has no bias. Each joint mask score is the sum of its three bit scores.

The shared model expands exactly into the existing 768 -> 24 -> 8 ABI: three
identical diagonal input blocks, repeated hidden biases, zero off-diagonal
weights, and joint rows assembled from the shared two-row output matrix. The
joint output bias is zero. Go's model format and runtime remain unchanged.
Fewer trainable parameters do not imply reduced resident runtime memory: this
first experiment intentionally exports the complete existing dense ABI.

Both start from the original reproducible fresh Go initializer, never pretrained
or previously trained weights. Dense takes it unchanged. Shared takes the first
eight hidden rows / first 256 input columns, the first eight hidden biases, and
the first eight columns of each of the first two output rows. It repeats those
parameters as described above, without rescaling. This deterministic recipe is
part of the experiment; the arms have different topology and effective capacity.

## Fixed optimization and export

Each arm uses the same function/language-balanced actual-feedback rows, passing-
set NLL, AdamW (learning rate 0.001, weight decay 0.01), shuffle seed 20261032,
100 FP32 epochs and 100 QAT epochs. Each epoch has eight 128-group batches:
800 updates per stage, 3,200 updates total. QAT starts its arm's selected FP32
checkpoint. Selection uses calibration passing-set NLL only, with the existing
temperature grid 0.5/1/2/4 and deterministic epoch/temperature tie breaks.

QAT quantizes the **expanded** matrices using the existing all-element scale
rule, including structural zeros. This may reduce quality; it is not silently
replaced with a different scale. FP32, PTQ and QAT exports from both arms are all
retained. Five trits per byte are 1.6 stored matrix bits, not total 1.58-bit RAM.
Exact prediction/logit parity, model pins, actual updates, sampled MPS allocation,
process CPU, wall time and lifetime RSS are separate observations.

## Evaluation and publication

Go must verify the expanded shared topology, all 288 exported parity rows and
the six actual inference kernels. Evaluate all 512 development initial views
for each of six exports: full passing-set mass, initial finite case matches,
ranked continuation attempts, bilingual disagreement, same-wrong agreement and
per-family results. The comparison must keep complete finite targets and tied
passing masks; agreement alone is never treated as correctness.

Native dogfood uses all eight families, Korean/English, configuration 20 / goal
4 for all six exports: 96 actual Gooo generations. Each generated body must be
followed immediately by the source-bound native runtime observer with the
existing 24-case independent oracle envelope. No batch-wide generation barrier,
post-emission model call, source mutation or new approval dependency is added.

There is no default-model promotion in this experiment. Publish both controls,
regressions, model artifacts, source hashes and bounded evidence to public GitHub
and Hugging Face after excluding private paths and credentials. Original data,
teacher captures and previous studies remain immutable. Reuse the original
feature bank rather than duplicate it; new retained artifacts have a 64 MiB cap
inside the existing 768 MiB study cap. Failed/canceled prefixes are preserved and
not restarted to manufacture success. GPU work remains local and bounded; Python
only performs offline optimization/export, with runtime and audit in Go.
