# Own Gooo passing-set and actual-feedback model v2

Nine fresh own models use the same 512/24/4 architecture, 12,412 parameters,
initial state, and bilingual source-bound corpus. These are Gooo structural
judgment models, not general Korean/English text generators. No pretrained or
Laya weights initialize them. The v1 own teacher contributes observed failures
only. All source collection, inference, evaluation and audits use Go; Python
is confined to offline MPS optimization and export.

## Observed SDK assembly results

The frozen experiment retains 6,144 actual teacher sessions, 12,399 teacher
predictions, 214,672 ordered evaluator invocations, and 4,159 unique continuation
states. Together with 2,304 original bilingual views this yields 6,463 student
rows. Only training configurations contribute continuation inputs. The three
arms share 3,600 actual FP32/QAT updates in total. Optimization loops took
7.829 seconds, excluding preparation and export. Learning process peak RSS
reached approximately 813 MiB; this is separate from inference memory.

All 12 policies were then measured on calibration and development:
9,216 real SDK sessions and 18,510 actual model predictions. The independent
auditor reconstructs source, arithmetic, every retained path, progress, actual
failure text, and the calibration-only selector without operational inference.

Development contains 384 function views per policy, 192 bilingual pairs,
and 16 ordered cases per view. These are already authored six-family finite
contracts; parameter, language, seed and replay counts are not new intentions.

| Policy | Complete at first candidate | Complete within 3 candidates | Extra candidates | Actual predictions |
| --- | ---: | ---: | ---: | ---: |
| Disconnected | 112/384 | 304/384 | 520 | 0 |
| Frozen v1 independent FP32 | 124/384 | 333/384 | 435 | 1,442 |
| Frozen v1 joint FP32 | 128/384 | 322/384 | 458 | 780 |
| Uniform initial FP32 | 104/384 | 326/384 | 488 | 814 |
| Passing-set initial FP32 | 104/384 | 327/384 | 488 | 815 |
| Passing-set actual-feedback FP32 | 116/384 | 343/384 | 432 | 775 |
| Passing-set actual-feedback PTQ | 124/384 | 292/384 | 502 | 794 |
| Passing-set actual-feedback QAT | 128/384 | 329/384 | 442 | 771 |

Against the matched uniform-initial FP32 control, the feedback arm reduces extra
candidate attempts by 11.48% and predictions by 4.79%; completeness within three
candidates rises from 84.90% to 89.32%. Changing the loss alone does not improve
extra attempts in this cohort. The experiment changes both the loss and observed
feedback curriculum in its final arm; it does not isolate feedback formatting
or each training example's causal effect.

Calibration selects the frozen v1 independent model: 397 extra candidates
versus 401 for the feedback FP32 student. No new model becomes an implicit
compiler default. The nine retained variants include regressions; PTQ worsens
the feedback student's extra attempts by 16.20%, and QAT by 2.31% relative to
its FP32 result. Reduced packed storage does not guarantee preserved behavior.

All policies eventually satisfy this finite contract within four candidates,
which includes deterministic enumeration. This is not a universal correctness
claim. Shared authored wording across splits and the already observed corpus
limit generalization claims. No CI hint was supplied; historical studies with
a supplied CI hint are not matched comparisons for prediction counts.

## Numerical and memory measurements

Go verified all 432 exported FP32/PTQ/QAT parity rows. The declared maximum
absolute error tolerance was 1e-5; observed maximum ternary error was about
1.49e-6, and all selected masks agreed. Each valid warmed inference allocated
zero heap objects. The feedback FP32 warm probe took approximately 10.8 us.
This kernel timing does not establish compiler wall-time improvement.

FP32 weight storage is 49,648 bytes. Ternary packed weights occupy 2,590 bytes
but runtime tensor storage is 12,496 bytes: 12,384 int8 matrix bytes and 112
FP32 bias bytes, plus 8 bytes for matrix scales. Request workspace
is 2,160 bytes. Five trits per byte is 1.6 stored bits per ternary weight,
separate from the theoretical log2(3) and total process RAM.

## Actual adopted-main execution

Adopted compiler main `363a3d8aa365c35dd634c241248b444de0050973` with SDK
v0.2.12 and Go 1.27.1 performed 240 actual Gooo generations. Every emitted Go
program was independently compiled and executed: 240 executions and 3,840
ordered invocations all matched the ordinary int64 oracle. These generations
made 473 actual model predictions. A separate zero-inference audit reconstructed
all source/context/model bindings, candidate paths, feedback, execution values
and process metrics. Codegen and Go-run process measurements are recorded
separately; Go-run includes compilation and execution.

The fixed configuration-40 subset has 48 function views per policy:

| Policy | Initially complete | Extra candidates | Actual predictions |
| --- | ---: | ---: | ---: |
| Disconnected | 12/48 | 70 | 0 |
| Calibration-selected v1 independent | 16/48 | 54 | 181 |
| Uniform initial FP32 | 16/48 | 57 | 99 |
| Passing-set initial FP32 | 16/48 | 57 | 98 |
| Passing-set actual-feedback FP32 | 16/48 | 53 | 95 |

This subset supports an additional-candidate improvement, with equal first
candidate completeness. It is not a new blind corpus or proof of a general
compiler wall-time improvement. The selected old model is retained according
to the preregistered calibration rule; no development-based default promotion
was performed.

The first native attempt retained five generation captures and four actual Go
execution records before an auditor rejected the disconnected plan hash.
The disconnected compiler correctly retains original source intents, whereas
model-connected input is encoded. The auditor now verifies each against its
actual source plan; an added regression test rejects swapping those plans.
The completed 240-generation run is a separate preserved attempt. No generated
function failure was hidden by discarding the first attempt.

## Observed complete-process cost

The matched feedback FP32 policy's 48 codegen children have a median wall time
of 10.792 ms, nearest-rank p95 of 12.644 ms, median measured child peak RSS of
17.55 MiB and summed CPU time / summed wall time of 83.41% of one core.
The uniform-initial control's median codegen time is 10.744 ms. The feedback
policy's compiled-Go execution median is 441.409 ms, including compilation.
The 10.8 us warmed model kernel and lower candidate count therefore do not
establish a whole-compiler speedup. Fixed policy order, caching and an already
observed corpus further limit interpretation. Child RSS is not simultaneous
process-tree RAM; this study did not measure a causal change in host CPU load.

The public process JSON can be reconstructed with the Go study's
`--mode native-metrics`, using the same `--dataset`, `--models`, `--sdk-study`
and retained native directory in `--output`, with a fresh summary path in
`--audit-report`. This first performs the zero-inference native audit, then
summarizes 240 generation and 240 compilation/execution child records.

## Public evidence and continuous execution

The [public own-model repository](https://huggingface.co/asketeddy/gooo-joint-feedback-tiny-v2)
contains nine models, PROV-O lineage and the complete fixed synthetic archive.
The clarified immutable revision is `b4e9a3e50b50e893abc52a36f49eacf032aded50`;
manifest SHA-256 is `e6eb1babcd94ccd1a75fea5d76069a5dc0bbb368f57318cc4a2608019b4a16a5`.
Anonymous retrieval verified all 48 public payloads and 614 raw archive members
(472,824,755 expanded bytes) against the local publication. This verifies
published bytes; it performs no additional inference or optimization.

The `own-joint-feedback-v2-native` CI job independently pins both public Hub
revision and manifest, streams the bounded archive into an ephemeral evidence
directory, audits original teacher/SDK/native records, executes fresh Go parity
probes and 9,216 SDK sessions, then builds exact adopted compiler main
`363a3d8aa365c35dd634c241248b444de0050973` for 240 new code generations and 3,840
actual emitted-Go invocations. Current CI outcomes are saved separately from
the original study and do not increase authored-intention counts. Artifact
retention is seven days; public frozen evidence remains reproducible.

[Source CI run 36952987501](https://github.com/kimjooyoon/gooo-neural-decision-experiments/actions/runs/36952987501)
passed all ten jobs at `c0a179dde93122faf449f6e42dbfdabce92cafb0`. Its new
v2 execution made 18,450 kernel probe predictions, 18,510 predictions in
9,216 SDK sessions, and 473 predictions in 240 native generations. All 240
new emitted Go programs compiled and all 3,840 ordered invocations passed.
These are repeated observations of the frozen cohort, not new authored
intentions or optimization updates. The 15,146,280-byte raw Actions archive has
519 fixed members and SHA-256
`0c5366fe90751d758e5608073ebbdcc4b61dc26eb2575213b6b83215e237e6ff`.
The Go CI publisher scans all raw text and preserves this archive for public
publication separately from the original model edition.

Linux CI records feedback codegen median 9.585 ms and measured child maximum
RSS median 112.36 MiB. These are not a controlled comparison with macOS.
Linux resource accounting survives `exec`, so pre-exec inherited memory is
one possible contributor to lifetime peak RSS; this study did not measure
substage RSS and does not establish that explanation for each sample.
Peak RSS cannot be substituted for steady-state model tensor memory.
See the [Linux getrusage documentation](https://man7.org/linux/man-pages/man2/getrusage.2.html)
and [fork documentation](https://man7.org/linux/man-pages/man2/fork.2.html).
