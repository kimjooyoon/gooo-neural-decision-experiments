# Bounded bilingual Gooo judgment, preregistered comparison

Commit this protocol, Go pair generator and trainer before training or inference.
Use the existing 6,240-view feedback curriculum. Pair only its Gooo views by
program and adjacent English/Korean template, with identical fallback body,
finite cases, eligible labels and soft target. Reuse 640 program groups in 1,040
pairs: train 800, calibration 80, development test 160. There are zero new
independent intentions; prior development evaluations have already used these
groups. Do not call this an untouched language benchmark.

Fine tune the fixed own feedback FP32 parent in two arms with identical seed and
optimizer settings: finite soft-target NLL alone and the same NLL plus 0.25 times
bilingual Jensen-Shannon divergence. Each arm runs 8 FP32 and 8 QAT epochs in
128-pair batches, exactly 224 total optimizer steps across both arms. PTQ exports
reuse each arm FP32. Select checkpoints on calibration loss only. Temperatures
use calibration finite NLL; positive scaling does not change pair argmax.

Report pair agreement, same-wrong-intention agreement, both-correct intention,
finite best-set selection, probability divergence, finite completeness before
and after a separate explicit complete test contract, added path attempts,
model calls, latency and resident tensor bytes. An English/Korean agreement can
be wrong. Sparse finite passing behavior is not a proof of instruction meaning.
The complete contract must come from the independent authored arithmetic oracle;
model confidence and a diagnostic witness cannot supply expected values.

Go performs pairing, runtime, evaluation and codegen orchestration. Python is
offline MPS training/export only. Retain both arms and all negative results.
Publish weights as an experimental appendix, not as a silently promoted default.
Bind source, data, pair references, parent, new weights, tests and outputs by
digests. Keep PROV-O roles distinct. No new upstream Laya calls are implied.
