# First finite-set feedback tuning result

The first offline GPU run completed with 760 optimizer steps: 380 FP32 steps and
380 QAT steps, 20 epochs per trainable variant. PTQ reuses the trained FP32 state.
The parent is the existing positioned-random structural model. This is fine
tuning our own small model, not upstream Laya weights or an optimizer in Gooo.

The two sampled MPS training loops took 0.970133 and 0.406094 seconds. Sampled
allocated MPS peak was 5,810,944 bytes; sampled driver allocation was 53,166,080
bytes. Python process lifetime maximum RSS reached 523,583,488 bytes. These
are different measurements; neither is the deployed model's RAM or host GPU
utilization. Feature preparation/export time is excluded from training-loop time.

Go numerical parity passed all 96 probes with maximum logit/probability error
0.00000166893, below the 0.0001 tolerance. The Go evaluation made 5,760 actual
model calls across six arms plus 96 parity calls; offline control made zero.
All seven arms reuse 960 development views of 320 existing instruction groups.
There are 76 finite ties in these views. They are not 960 independent tasks.

| Arm | Best finite set selected / 960 | Passed finite cases / 5,120 | Paired soft-target NLL |
| --- | ---: | ---: | ---: |
| Parent FP32 | 814 | 4,072 | 0.424359 |
| Feedback FP32 | 754 | 3,748 | 0.436297 |
| Parent PTQ | 808 | 4,038 | 0.475826 |
| Feedback PTQ | 786 | 3,895 | 0.542495 |
| Parent QAT | 817 | 4,076 | 0.449959 |
| Feedback QAT | 756 | 3,742 | 0.471347 |
| Deterministic fallback | 518 | 2,514 | 0.693147 |

This run did not improve the parent on these finite-set selections or aggregate
case outcomes. That negative observation remains part of the publication.
Original intention-label agreement is recorded separately in the raw audit;
finite ties cannot establish a unique natural-language meaning. All models
retain global abstention at the development threshold. Compiler-owned pair
ranking can still explore paths and retain partial bodies under bounded TDD.

FP32 stores 50,912 tensor bytes. Ternary variants store a 2,759-byte weight file
and occupy 12,896 resident tensor bytes, plus eight matrix-scale bytes and a
1,248-byte per-worker workspace. These are model arrays, not whole-process RSS.
Five base-three weights per byte give 1.6 disk bits per matrix weight. Go expands
them to int8; no 1.58-bit total RAM or packed-compute acceleration is claimed.

Source-pinned execution conditions and complete histories are in
`runs/feedback-path-soft-target-mps-20261001/{preexecution,report,go-parity,go-audit}.json`.
The source/trainer commit is `439da7bce8e801da94720696fbf16165a3bd099e`.
This is an experimental checkpoint. Actual native dogfood and public HF weight
verification remain separate required observations before public weight release.
