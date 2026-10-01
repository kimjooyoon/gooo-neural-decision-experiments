# Own Gooo compiler-context tiny model: first bounded training result

This study trains our own 12728-parameter model from random initialization, with
no Laya or inherited pretrained weights. Runtime inference and compiler bridges
are Go. Training/export used local Apple MPS only, 560 optimizer updates total
across two matched arms (FP32 140 + QAT 140 each). Three representations per arm
were retained: FP32, PTQ ternary and QAT ternary.

## Compiler-bound data

2080 successful native `body-context` calls produced 1040 bilingual pairs,
640 program groups, five typed path families and the existing partition
800/80/160 pairs. Source/fallback authority and canonical input hashes were
independently audited. A first 2080-call collection had an 8MiB reader failure
and incorrect empty capture digest; it is retained, rejected and not trained on.
Total local export calls are 4160, of which exactly 2080 feed this study. There
are zero new independent intentions. All inputs are synthetic/public.

## Development cohort (canonical runtime input in every arm)

| Training arm | Representation | First whole finite programs /320 | Extra candidates | Whole after at most one extra |
|---|---|---:|---:|---:|
| Caller context | FP32 | 252 | 68 | 320 |
| Caller context | PTQ ternary | 176 | 144 | 320 |
| Caller context | QAT ternary | 244 | 76 | 320 |
| Compiler context | FP32 | 240 | 80 | 320 |
| Compiler context | PTQ ternary | 177 | 143 | 320 |
| Compiler context | QAT ternary | 219 | 101 | 320 |
| Disconnected deterministic | No model | 160 | 160 | 320 |

Each arm finishes 3840/3840 independently authored full finite cases after at
most one extra candidate. This is finite completeness, not a universal proof,
unseen benchmark or broad natural-language correctness. Sparse training has
76/320 ambiguous development views; alternatives consistent with the authored
finite cases were preserved in soft targets. Intention labels are counted
separately. Compiler-context FP32 initial accepted-set coverage is 261/320.
It initially completes 240/320 (75%), with 80 bilingual pair disagreements.

Compiler context is **not a quality improvement over the matched caller-context
control**: FP32 loses 12 initial complete programs, QAT loses 25; PTQ gains one.
Against disconnected fallback, compiler FP32 reduces extra candidates from
160 to 80 (total candidates 480 to 400), but the caller control is better still.
The result supports optional bounded judgment with deterministic completion;
it does not justify replacing the default model automatically.

## Runtime and local GPU observations

Go held-out audit: 1920 model predictions plus 192 numerical-parity predictions.
Maximum Go/Python absolute difference is 1.7881393432617188e-7.
Compiler-context median measured inference: FP32 9.583us, PTQ 10.792us,
QAT 10.458us (320 calls each; isolated process-local measurements).
Model workspace is 1248 bytes; resident FP32 tensors 50912 bytes; ternary tensors
12896 bytes plus matrix scales. The packed ternary payload is 2759 bytes, five
trits per byte (1.6 disk bits per weight); decoded int8 execution, not packed
arithmetic or a 1.58-bit process RAM claim.

Training loops totaled 1.3093 seconds. The first loop was 0.7213s and retained as
a startup-sensitive observation, not used as a comparative throughput estimate.
Sampled MPS allocation peaked at 2033920 bytes, driver allocation 53166080 bytes,
process lifetime RSS 473366528 bytes. These are sampled allocation and process
memory measures, not whole-host GPU utilization.

Actual native dogfood: 40 deterministic development views x four arms = 160
native generation calls, 120 local model predictions; canonical exported input
hashes and full-case native actuals match. The first view per family/arm also
compiled and ran generated Go (20 executions). This deterministic subset favors
the first cases in existing order and is not a population accuracy estimate.
Raw child wall/CPU/RSS and captures are retained. No provider network calls,
online optimizer or default model deployment was performed.

## Next research direction

Preserve compiler-owned context as the source of facts. Improve what the model
can distinguish: local variable definitions (not only names), branch body effects
(not only indices), dependent decision relations and repeated partial-test
feedback. Compare context encodings with matched capacity/optimizer/data, and
retain finite ambiguity rather than forcing a single answer. Teach Korean and
English judgments separately enough to measure disagreements. Default model
selection should follow completeness and additional candidate cost across these
comparisons, while disconnected execution continues deterministically.
