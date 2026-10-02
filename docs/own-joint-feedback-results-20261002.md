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
but runtime tensors decode to 12,496 int8 bytes plus scales. Request workspace
is 2,160 bytes. Five trits per byte is 1.6 stored bits per ternary weight,
separate from the theoretical log2(3) and total process RAM.

## Next execution evidence

The preregistration requires 240 actual adopted-main Gooo generations and
independently compiled Go executions, with 3,840 ordered invocations. SDK
evidence above does not fulfill that native execution requirement. Native
results and immutable public HF raw evidence will be recorded after collection
and independent audit. See the frozen protocol and public JSON summaries.
