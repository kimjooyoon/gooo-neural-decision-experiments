# Matched three-choice own model optimization

The original [preregistration](own-three-choice-completeness-preregistration-20261002.md)
is unchanged. This implementation is published before preparing features or
running new student optimization. Previous source/teacher public evidence has
passed all eleven jobs in run `36972302940` at source
`be03c6cce8b0aa4cf1e106aeef85891b54a4addf` and anonymous full-member verification.

## Source data and Go preparation

`internal/threestudent` accepts only the frozen native dataset, complete teacher
state file and independently audited collection byte hashes. Each state retains
its source identity, full ordered three-part context and complete passing-mask
target. Feedback states must be training-only and retain actual capture origins.
Go exports row-major little-endian FP32 features using the same fixed `[768]`
array projection as runtime. No input is shortened or replaced.

The preparation tool retains a closed phase inventory, initial weights, exact
source-bound row metadata, split counts and original raw evidence pins. Its
separate replay reconstructs all 8,247,552 float values, every row weight and
the initial weights without prediction, optimization or native execution.
Development rows are retained for parity and later evaluation; optimizer and
checkpoint selection arrays are constructed only from train/calibration rows.

Every function group has total weight one; each language has weight one half.
Initial-only arms use each original language view at weight 0.5. In the feedback
arm a language with valid unique failures has initial weight 0.25 and total
failure weight 0.25 divided equally among its distinct contexts. A language
without failures retains original weight 0.5. Duplicate session origins do not
raise a context's weight. These are the preregistered per-view 0.5/0.5 shares
after applying the language's 0.5 share.

## Independent common initialization and fixed MPS work

Go creates a fresh `math/rand.NewSource(20261031)` stream. Each FP32 parameter is
uniform in `[-1/sqrt(fan_in), 1/sqrt(fan_in)]`, serialized in `w1,b1,w2,b2` order.
The full 74,624-byte initializer is archived and independently reproduced before
optimization. No pretrained, Laya, frozen teacher or prior student tensor is
loaded into the fresh students. Python reads byte-identical Go features/weights
and is limited to offline optimization and tensor export.

Each of the three arms uses AdamW, learning rate 0.001, weight decay 0.01,
100 FP epochs and 100 QAT epochs, with shuffle seed 20261032, 128 source groups
per update and exactly eight updates per epoch. There are 4,800 actual updates;
PTQ adds none. QAT starts its own selected FP checkpoint. Straight-through
ternary QAT uses the existing explicit mean-absolute matrix scale and rounded,
clipped trits. Export stores five trits per byte plus FP32 biases: 3,854 bytes.
This is 1.6 stored matrix bits, separate from decoded int8/bias/scale/workspace
and process memory.

## Joint epoch and temperature selection

FP and QAT independently consider every epoch with temperatures `{0.5,1,2,4}`.
Selection minimizes `(calibration passing-set NLL, epoch, temperature)` in that
order. Epoch selection therefore considers calibrated probabilities, not just
temperature-one loss. Passing-set NLL uses stable logsumexp over every tied
passing mask. The uniform initial objective distributes its target across the
same set; neither objective overwrites a joint contract with marginal labels.
PTQ uses the selected FP checkpoint and chooses its temperature on calibration
only. All three exports from every arm remain, including any regression.

Every completed update is appended to a bounded stage journal; completed
epochs also have immutable receipts and all four calibration values.
A failure/cancellation retains its prefix and
stops dependent work. Full-study retained evidence remains capped at 768 MiB;
there is no restart or deletion to manufacture a successful run.

The next evidence boundaries remain separate: Go numerical parity and warm
allocation/cost probes, the complete 11,264-session SDK study, calibration-only
policy selection, 640 actual native generations and independent compilations,
and public GitHub/Hugging Face byte verification. Training/export alone cannot
promote a default or prove Gooo/natural-language completeness.
