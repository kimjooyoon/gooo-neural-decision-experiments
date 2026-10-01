# Native bilingual path diagnosis dogfood smoke

The preregistered Go runner `18ae830c18b47b377ae8fa97dcc39df75f6b593b`
used clean native feature `5ae486d05b3521ecc48b18bfa28d45cdd9041203`
with SDK v0.2.8 and the already published own FP32 model. This is one reused
subtraction intention with Korean and English instruction views. It is an
integration smoke, not a new language benchmark, training run or main deployment
measurement. Main adoption and source CI are recorded separately.

## Actual work

Eight compiler invocations form four diagnosis-off/on pairs. Four own-model
predictions precede construction; the four deterministic diagnoses add zero
predictions. The diagnoses observe eight typed candidates. All four pairs retain
identical selected choices and emitted Go. Two distinct selected emitted Go
programs were actually compiled/run on `[2, 3]`: four function evaluations,
matching an independent arithmetic oracle. Alternatives in diagnostic receipts
remain typed-arena observations, not native-executed alternatives.

All eight constructions pass the single declared case `input=2, expected=0`.
Both `input-2` and `2-input` satisfy it. The model selects `input-2` for the
Korean instruction and `2-input` for the English instruction. This English
selection disagrees with the stated instruction while achieving 100% finite
test completion. Preserve that negative result; no first-shot correctness or
language consistency gain is claimed. The diagnosis reports the ambiguity and
input three as a witness in every pair. Its two outputs have no expected-answer
authority. The unresolved reference count is one, not an equivalence proof.

## Resources and limits

The four native diagnostic stages take 0.034458, 0.035083, 0.050833 and 0.052750
milliseconds. Whole-child wall time across all eight calls ranges from 6.249291
to 465.077333 ms; median 8.124917 ms. The first disconnected invocation's
465 ms startup outlier is retained. Whole-child CPU time ranges from 5.540 to
24.728 ms; median 7.270 ms. Peak RSS ranges from 15.797 to 16.250 MiB, median
16.102 MiB. These are descriptive observations from eight calls, not stable
performance estimates, causal speedup or host CPU utilization changes. Compiler
stage timing excludes process startup; whole-child costs include it. Controller
and separately compiled emitted-Go costs are excluded from native process
sidecars. No model loading or resident tensor reduction is claimed.

The report and independent offline audit are byte-identical. Audit checks
source/model/compiler pins, exact selected function AST modulo in-range literal
wrappers, independent arithmetic output-vector digests and witnesses, actual
emitted-Go outputs, pair consistency and counts. It does not repeat predictions
or processes: ranking origin relies on the committed runner/model and source CI.
No new training, GPU work, upstream Laya calls or model promotion occurred.

## Use in development

Gooo declarations bind the available construction paths. Korean/English text
can rank these paths; finite tests guide bounded assembly. Optional diagnosis
then exposes missing information. A later authoritative test may resolve a
witness, but this experiment does not synthesize an expected answer from model
confidence or choose an alternative automatically. Disconnected construction
and diagnosis remain deterministic. Expression limitations and finite gaps
stay in public evidence instead of being hidden by a success percentage.
