# Small model for source-owned record fields

Protocol and trainer were published before optimization at research source
`4441c221dfa1b13230d9469d56974b5d7742b174`. The executed 480-update pilot and
its Go/native evaluation are retained in
[the result report](../publication/record-field-learning-20261005).

The next language step replaces the integer ordinal proxy with actual field
alternatives and Korean/English intent. Gooo still declares the enclosing
condition, variables, field choices, attempt budget and finite examples. The
model orders the permitted pieces; execution measures what they assemble.

## Fixed pilot

The Go curriculum generates eight source/expression families with six choice
permutations, eight first/second orientations and paired Korean/English views:
768 source files. Families 0–3 provide 384 training views, 4–5 provide 192
calibration views, and 6–7 provide 192 test views. Families vary early returns,
nested conditions, Boolean locals, field aliases, same/other-field copying,
neutral concatenation and prefix/suffix order. These variations exercise three
roles: preserve title, set state to `ready`, append `:accepted` to the reason.
They do not count as 768 independently invented semantic algorithms or satisfy
the separate 100-distinct-experiment objective.

For every view, a clean pinned compiler exports the full source context without
prediction/outcomes and performs disconnected finite construction. Its observed
complete mask must match the authored role oracle: five complete selection
cases and 15 matching fields. Go emits the exact 768-element FP32 features and
a fresh 18,656-parameter initializer seeded with 20261051. Whole-family splits
are declared before fitting; exact context and projected-feature overlaps are
reported explicitly. Body-only variation can share a model representation.

Feature contract `triple_record_field_context_v1_joint_v1` uses complete ordered
expressions, 32 structural slots per alternative and 192 intent fragment slots
per choice. Its field map excludes test inputs/outputs, enclosing control flow
and local binding resolution. Literal fragments are case-folded 2/3-byte hashes;
different literals can share a feature representation, including single-byte
constants. Finite evaluation remains necessary for these representation limits.

## Small GPU work

Offline optimization runs on local MPS. The Go feature bank is read directly;
Python performs only PyTorch optimization/export. Go owns source generation,
bridges, inference and compiled execution. No pretrained, Laya or previous own
model tensors initialize this pilot.

Use 40 FP32 epochs, then 40 straight-through ternary QAT epochs starting from
the selected FP32 checkpoint. AdamW uses learning rate 0.003, weight decay 0.01,
batch size 64 and shuffle seed 20261052. There are six updates per epoch,
240 per stage and 480 total. Calibration alone selects `(NLL, epoch,
temperature)` over `{0.5,1,2,4}`. Export FP32, PTQ and QAT regardless of quality;
PTQ adds zero updates. Store actual update/epoch journals, wall and process CPU
time, sampled MPS tensor/driver peaks, weights, full source inputs and parity
records. No host GPU utilization claim is inferred from allocation.

The preparation is about 2.25 MiB of features plus source/context observations;
the pilot output budget is 64 MiB, including preparation and optimizer output.
Failed prefixes are retained. No development-label tuning or automatic default
promotion follows this pilot.

## Evaluation after export

Go independently regenerates row features, checks export logits and observes
all 192 held-out views with each model. Report complete-mask ordering and rank
separately from finite selection. Then use the freshly trained model in actual
Gooo graph construction on new Korean/English runtime inputs. Compare model-free
ordering, the frozen ordinal model and all three record exports at the same
1/2/4/8 candidate budgets. Keep every partial result and actual mismatch.

Record prediction time, case/field completion, candidate attempts, tensor bytes,
build/replay timing and source/model identities. Public source and model artifacts
should make the executed scope understandable without assuming general code
generation accuracy.
