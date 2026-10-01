# Bilingual Gooo judgment with bounded continuation

This experimental appendix keeps both trained arms, including regressions.
First-shot accuracy is an observation; useful partial construction and finite
continuation are the development target. These are independently initialized own
tiny-model descendants, not fine-tuned upstream Laya weights.

## Actual work and scope

The source-bound generator pairs existing English/Korean Gooo views: 1,040 pairs
reuse 640 program groups, with 800 training, 80 calibration and 160 development
evaluation pairs. Zero new independent intentions. The entire original curriculum
and raw captures are in the
[public research repository](https://github.com/kimjooyoon/gooo-neural-decision-experiments).
Gooo structure, legal options, source groups and finite soft targets remain bound
to their original digests. Natural-language meaning and finite test agreement are
separate labels; PROV-O vocabulary in the source is not ontology reasoning.

Two arms start from the same own feedback FP32 parent and use the same seed.
Control minimizes finite soft-target NLL; paired adds 0.25 times bilingual
Jensen-Shannon divergence. Actual MPS training performed 224 optimizer steps
across FP32 and QAT in both arms. PTQ reuses each FP32. The four training loops
totaled about 1.52 seconds; the first loop includes startup and was 1.29 seconds,
while the other loops were about 0.07–0.08 seconds. Fixed arm order and a single
run do not establish comparative training throughput. Sampled MPS allocated
memory peaked at 2,033,920 bytes; driver memory at 53,166,080 bytes. Python process
lifetime peak RSS was about 504 MiB. These are not host utilization measurements.

## Judgment and finite continuation

Go measured 160 English/Korean pairs (320 views) in each cell. Pair agreement
alone can be wrong, as the disconnected control illustrates.

| Model | Initial agreement /160 | Same wrong intent /160 | Complete views before /320 | Added path attempts |
|---|---:|---:|---:|---:|
| Own parent FP32 | 91 | 4 | 243 | 77 |
| Control FP32 | 97 | 4 | 249 | 71 |
| Paired FP32 | 96 | 4 | 248 | 72 |
| Own parent PTQ | 99 | 1 | 257 | 63 |
| Control PTQ | 108 | 14 | 240 | 80 |
| Paired PTQ | 108 | 14 | 240 | 80 |
| Own parent QAT | 90 | 2 | 246 | 74 |
| Control QAT | 95 | 15 | 225 | 95 |
| Paired QAT | 95 | 15 | 225 | 95 |
| Disconnected deterministic | 160 | 80 | 160 | 160 |

The paired term did not improve this development comparison. Both new ternary
arms increase same-wrong agreement and require more attempts than their own
parents. This evidence is retained; no global default model promotion is made.

After a separate complete contract from the authored arithmetic oracle, every
cell reaches 320/320 complete views and 3,840/3,840 finite cases with at most one
additional legal path per view, zero additional model calls. This is exhaustive
two-path typed-arena evaluation, not universal intent correctness, arbitrary
code synthesis or native execution. The Go judgment capture performed 2,880
initial model calls. Separate full-curriculum audits in both new arms each made
5,856 calls including numerical parity and reused parent comparisons. Six new
exports pass Go/Python parity with maximum observed error about 1.55e-6.

Runtime prediction medians were about 10.1–10.2 microseconds for FP32 and
11.7–11.8 microseconds for ternary in this capture. FP32 tensor storage is 50,912
bytes. Ternary packed weights are 2,759 bytes on disk, decoded int8 tensors are
12,896 bytes resident, matrix scales 8 bytes and inference workspace 1,248 bytes.
Five base-three digits per byte give 1.6 disk bits per matrix weight; theoretical
log2(3) is about 1.585. Neither is total process RAM or packed compute speed.

## Actual adopted-main dogfood

The clean adopted compiler main `1e01c96c54f2f8dd43334b8f580af93ffaea24df`
uses Go 1.27.1 and SDK 0.2.8. Main push CI has six canonical successful checks and
an independently verified proof, separately preserved in this appendix.

Twenty-eight actual native calls reuse one subtraction intention in English and
Korean, six trained variants and disconnected control, sparse/full contracts.
Twenty-four actual model predictions. Six sparse calls choose the wrong intended
operand order while passing the one input where both paths return zero. All
fourteen full-contract calls select the intended body, using six extra candidate
attempts. Full contracts are separate calls with one new ranking each, not
zero-call native continuation. Actual emitted Go execution ran the two distinct
selected bodies on inputs 2 and 3: two processes and four function invocations.
An independent offline audit rechecks hashes, contracts, selected function
structure and arithmetic values without inference or subprocesses.

Across those 28 native child calls, observed wall time was 5.43–13.81 ms
(median 5.76 ms), CPU time median 4.93 ms, peak child RSS roughly 15.7–16.3 MiB.
These whole compiler processes are distinct from model-stage latency and tensor
memory. No causal speedup, host CPU utilization change, upstream Laya use or
production release-readiness is claimed.

## Use

Both arms expose the existing Go typed-path model ABI. From the research checkout,
use `--path-model runs/bilingual-judgment-mps-20261001/paired/models/fp32/model.json`
with the native compiler body-codegen command and an explicit typed path plan.
Disconnecting the model keeps the deterministic path policy. Always retain the
source/test contracts and partial denominators; repeated calls and translations
are not independent experiments. Model confidence is a ranking hint, not an
authority to redefine expected values.
