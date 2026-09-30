# Gooo compiler model development loop

This iteration was performed directly without subagents. The tiny Go model is
an optional native compiler proposal provider; it was used on actual Gooo
sources and their typed body-fill plans while developing the model/tooling.

## Implemented path

1. Go constructs the public synthetic curriculum from the frozen original split.
   A plain intent, a typed Gooo declaration and a PROV-O context form three views
   of each instruction. Split grouping survives this expansion.
2. The existing offline MPS environment fine tunes FP32, then ternary-aware QAT.
   PTQ is derived from FP32. Two known failed native prompts are explicitly
   train-only repairs and receive additional optimization exposures.
3. An independent Go audit validates 6,146 total rows, 768 held-out views and
   96 sampled Go/Python feature/logit/probability comparisons.
4. Native `gooo body-codegen --tiny-model` loads the strict bundle and proposes a
   root operation inside a typed expression hole. The source, body plan,
   eligible candidates, finite case scoring and deterministic fallback remain
   compiler-owned. Prediction currently occurs after candidate evaluation.
5. The Go dogfood tool saves raw native output before validating it, executes the
   emitted Go against the same finite cases, and writes PROV-O Turtle containing
   source, plan, model, generated-code and verification digests.
6. A fixed allowlist packages only public model/evidence files. Anonymous fetch
   at one immutable HF revision verifies every published byte.

Offline training uses Python; curriculum construction, inference, typed assembly,
process supervision, independent Go replay and publication verification use Go.
No external Laya inference service was called in this iteration.

## Observed denominators

| Observation | Denominator | Result and interpretation |
|---|---:|---|
| Synthetic new-model held-out FP32 | 768 views / 256 original instructions | 768 correct |
| Synthetic new-model held-out PTQ | same | 643 correct; 565/601 accepted correct |
| Synthetic new-model held-out QAT | same | 768 correct |
| Go/Python sampled numerical parity | 96 rows | all within absolute tolerance 1e-4 |
| Native parent capture | 6 children / 6 predictions | raw operation 0/6 correct; original independent replay failed in harness |
| Parent Go-only replay, corrected | 6 programs / 36 cases | 36 pass; zero new native/model calls |
| Native improved model | 6 children / 6 predictions | raw operation 6/6; 5/6 applied; 36/36 generated-Go cases pass |
| Native disconnected baseline | 2 children / 0 predictions | 12/12 generated-Go cases pass |

Native generation made 14 child invocations once each, 12 local predictions,
zero external model calls, zero warmups and zero retries. The diagnostic Go
audit makes separate adaptive prediction loops. Go-only adjudication invokes
no native compiler or model. The six model cells use two known repair-training
intents under three variants, not six unseen tasks.

Those counts describe the local training iteration. The independent Linux CI
run [36759253245](https://github.com/kimjooyoon/gooo-neural-decision-experiments/actions/runs/36759253245)
at source `e04eeb36f9679ac80be26785aef9fc102dde58bf` separately made six native
model invocations and two disconnected invocations, passing 36/36 and 12/12
finite generated-Go cases. All three CI jobs passed. Each subsequent CI run
repeats its own cohort; it is not included in the 14 local invocation count.
The path-neutral [CI reports](../publication/compiler-prov-v2-ci-e04/) retain
their own compiler/toolchain bindings and resources.

The HF bundle is public at immutable revision
`0635717235c38a15b444f020283768c6122a6cb1`. A Go verifier sent no credentials,
checked the public repository inventory, and fetched all 23 allowlisted files
against the local SHA-256 and byte counts. See the
[anonymous verification receipt](../publication/hf-compiler-prov-v2-public-verification.json).

## Measurements

The two 60-epoch MPS optimization loops took 1.824 and 1.323 seconds. End-of-epoch
sampled allocated GPU memory peaked at 5,880,320 B; this does not include every
transient allocation. The Python process lifetime RSS peaked at 471,531,520 B.

The improved native cohort takes 6.49–11.52 ms per process, with provider decisions
of 37.6–63.4 microseconds. Native lifetime peak RSS reaches 16,924,672 B. Child
CPU totals 48.952 ms over 55.255 ms native wall, 88.6% of one core; that is not
the host utilization increase. The Go verification phase is outside this cohort.

The standalone kernel audit reports about 8.6 microseconds FP32 and 9.5
microseconds QAT with zero allocations. QAT weights are 2,759 B on disk; decoded
resident tensor arrays plus scales are 12,904 B, and each worker owns 1,248 B
scratch. Packing is 1.6 bits/matrix weight, with floating point biases/scales.
No packed 1.58-bit execution kernel or throughput speedup is demonstrated.

## Failed attempts are separate evidence

- `audit-attempt-1.json` rejects the new total row count before inference.
- `go-audit.json` retains the failed old test denominator (256 versus 768);
  `go-audit-v2.json` is the separately corrected audit.
- `native-parent/report.json` retains the trimpath Go-path harness failure.
- `parent-adjudication/report.json` retains the second harness failure caused by
  a hardcoded package name. Its original six raw verification logs include host
  temporary paths and are preserved privately. The public projection lists
  their unchanged digests and omits the pathful logs; it does not represent
  redacted text as original raw bytes.
- `parent-adjudication-v2/` contains a successful, separate Go-only replay after
  parsing the package name from the emitted source. Original capture fields are
  inherited historical measurements, not new native/model invocations.

PROV-O links represent observed entities and activities. They do not turn finite
tests into proofs of full intent, and prompt vocabulary conditioning is not an
ontology reasoner.

## Next compiler decisions to study

The current eight-output ABI is deliberately small. Richer experiments should
extend the typed choice space to operand/reference selection, branch layout,
assignment dependency order, and choosing a subplan. Each choice needs its own
independent intent oracle, type eligibility and finite case denominator.

The proposed feedback data record binds the source intent, typed plan, eligible
choices, raw model proposal, applied/abstained status, counterexamples and final
selection through PROV-O entities and activities. Collecting these records is
implemented by the dogfood traces; automatic curriculum construction from
arbitrary traces and production online retraining are future work.

Model improvement should report separate rates for raw proposal correctness,
acceptance coverage, post-compiler case completeness, unknown intent obligations,
candidate evaluations, latency and memory. Existing 100% finite test results
must not be presented as complete natural-language understanding.
