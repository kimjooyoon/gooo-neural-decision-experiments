# Own three-choice SDK results — verified incomplete prefix

## Actual execution and stop

Producer source is `b8c515fcd5a5e623aae5fd3b93c3b8f69f9f4608`.
The original 768 MiB whole-study raw cap stopped collection before another
session could run. The exact completed prefix remains unchanged: **8,779 actual
SDK sessions**, **33,388 actual predictions**, and **510,256 independently
verified ordered candidate values**. All eleven calibration cells are complete.
Six development cells are complete, followed by 75 actual views of the seventh.
The planned 11,264-session comparison is incomplete; 2,485 sessions remain.
No native generation, new optimizer update or default promotion followed this
cap failure. No seed, model, corpus, family or retained record was replaced.

Prior raw evidence was 341,374,424 bytes. This phase retained 460,811,265 bytes,
including the terminal failure receipt. Reserving complete capture/terminal
space prevented an unrecorded next call. The public compressed bundle preserves
every original phase byte, the unchanged protocol, the source dataset and the
independent audit. Compression is publication storage, not a claim that the
original capped experiment finished.

## Complete calibration comparison

Every row below contains 512 views with 16 contract cases per view. All eleven
policies and negative variants remain in the complete calibration receipt.

| Policy | Extra candidates | Actual predictions | Complete by candidate 4 | Complete by candidate 8 |
|---|---:|---:|---:|---:|
| Disconnected | 1,540 | 0 | 302 / 512 | 512 / 512 |
| Frozen v1 independent FP32 | 1,129 | 4,594 | 381 / 512 | 512 / 512 |
| Uniform initial FP32 | 1,293 | 1,776 | 363 / 512 | 512 / 512 |
| Passing-set initial FP32 | 1,276 | 1,760 | 366 / 512 | 512 / 512 |
| Passing-set real-failure FP32 | **1,060** | **1,558** | **413 / 512** | 512 / 512 |

The frozen calibration-only selector picked `set-feedback/fp32` before
development was observed. Relative to the frozen independent reference it used
6.11% fewer extra candidates and 66.09% fewer actual predictions. Relative to
the matched uniform FP32 student it used 18.02% fewer extra candidates and
12.27% fewer predictions. This is evidence for cheaper finite continuation;
full budget completion also occurs with disconnected enumeration and does not
establish general language accuracy.

The selected student's complete-function counts at budgets 1/2/4/6/8 are
106/250/413/479/512. Initial complete functions are 20.70%, below the frozen
independent reference's 25.98%. First-shot correctness is not the measured
benefit. No negative PTQ/QAT result is discarded.

## Recorded execution intervals and limitations

The selected calibration student has median SDK session wall **0.201584 ms**,
p95 **0.519750 ms**, and 122.490242 ms summed session wall. The frozen reference
has median 0.215666 ms, p95 0.568667 ms and 132.877808 ms summed session wall.
These are actual fixed-workload observations in the declared sequential policy
order; they are not a general speedup or a controlled host-utilization effect.
SDK session intervals include source assembly, tests, context and model calls;
independent audit, serialization and storage are outside those intervals.

Selected-student recorded prediction time totals 28.605736 ms over 1,558 calls,
about 18.36 microseconds per call including full context feature preparation.
Model load, compilation of independent native executables and GPU utilization
are separate measurements. This original failed producer wrote whole-process
CPU/RSS only on full completion, so collector CPU and process peak RSS are
**unavailable for this prefix**. Terminal resource recording is now repaired
for future collection attempts; no retrospective CPU/RSS values are invented.

Every initial paired Korean/English mask disagrees for the selected student:
256/256 pairs. The frozen reference disagrees on 236/256. Some masks can be
functionally equivalent, so mask disagreement alone is not a claim of different
outputs, but bilingual invariant judgment has not been established. The model
is not suitable as unchecked natural-language semantic authority.

## Complete observed development cells

The selected student's 512 complete development sessions used 1,145 extra
candidates and 1,638 predictions; frozen reference used 1,156 and 4,706.
Their complete-function curves are respectively
`[95,233,316,406,437,459,493,512]` and
`[116,250,328,379,413,456,486,512]`.
The other complete development cells are offline, feedback PTQ, feedback QAT
and passing-set initial FP32. The remaining students are not compared as if
their full development denominators were available.

The next work must retain this negative resource outcome, complete the same
remaining policy/view identities under an explicitly published storage plan,
and audit the entire comparison before dependent native execution. Model
selection remains the original calibration selection. No future development
result can be used to replace a seed, arm, wording or authored family.
