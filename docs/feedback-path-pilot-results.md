# Frozen-model feedback judgment pilot

This appendix reuses the existing three tiny structural models. No new weights,
optimizer steps, GPU calls or upstream Laya calls occurred. The Go research
session explicitly asks the original frozen model after a partial batch using
observed finite failures and actual native-compiler CI context.

The native compiler's post-main CI is source `307159f041644a3aa56dfd325c345f5325aec902`,
run `36801794319`, with six passed checks and Go-verified proof/receipt. This
context is a judgment hint; it is not authority to change source or proof that
the newer research runner itself passed CI.

## Measured result

| Observation | Result |
| --- | --- |
| Policies / paired comparisons | 32 / 16 |
| Different compound intentions | 1 |
| Same emitted Go / same finite results | 16 / 16 pairs |
| Actual local predictions | 246, including 102 feedback predictions |
| Feedback rounds | 17 |
| Candidate attempts | 1,268 |
| Finite selection cases | 208/224 across repeated policy views |
| Separate input evaluation | 288/288 observations of 9 existing distinct inputs |
| Native codegen calls | 32 |
| Distinct generated-Go/input-vector executions | 2, reused by byte/input hash |
| Pairs with fewer / more / equal candidates | 1 / 2 / 13 |
| Model-policy median search wall time | 1.888ms baseline; 2.222ms feedback |

Complete contracts satisfy 7/7 stated cases. Deliberately inconsistent contracts
retain 6/7 and exhaust all 64 paths; reconsideration cannot manufacture a valid
answer for the conflicting expected value. Korean QAT completion used 19→17
candidates; Korean FP32 used 10→11 and PTQ used 17→18. No aggregate functional
improvement occurred. The original model was not trained to interpret this
failure summary. Extra calls and worse observations remain visible.

All policies used the same compound intention, two language views, four model
arms including disconnected controls and two contracts. Baseline runs came
first, there was one ordering and no repetitions. Search timings exclude model
load, native generation, Go compilation and process startup. CPU/RSS and host
utilization changes were not measured in this pilot. These data do not establish
general natural-language accuracy, a causal speedup, 32 new experiment ideas or
288 independent language tasks. No-model controls reproduce their progress and
source exactly despite the enabled control flag; they make zero model calls.

## Reproduction and preserved failures

The public GitHub repository is
[gooo-neural-decision-experiments](https://github.com/kimjooyoon/gooo-neural-decision-experiments).
Runner source is `63c9967505355ccc5c56780f901f4ce583008f17`;
full captures are in `runs/feedback-path-pilot-fixed-20261001`.
The audit re-interprets all 1,268 candidate bodies and checks progress, feedback,
native and executed-Go captures, with zero new model or native calls.

The earlier `runs/feedback-path-pilot-20261001` attempt is preserved. It made
zero model predictions and one rejected native call because the runner nested
a complete Gooo module inside another body. The compiler remained unchanged;
the repaired runner passes the complete module directly. A separate wrong
revision preflight was rejected before any model or native call.

This directory is a separately allowlisted **evidence appendix**. The previous
v5 core manifest, model card and nine weight files remain unchanged. Raw captures
and implementation live on GitHub. The new appendix includes only synthetic
intentions and observations; credentials, private repositories and host paths
are excluded. First-shot accuracy and perfect expression are not prerequisites
for publishing an observed partial result.
