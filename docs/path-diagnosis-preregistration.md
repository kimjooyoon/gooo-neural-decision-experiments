# Finite path ambiguity and distinguishing inputs

Frozen before the new diagnosis study. The existing 72-view compound cohort is
reused: twelve intentions, three templates, two languages and three contracts.
This is development evidence, not a new independent task or language holdout.

Two arms use disconnected ordering and existing new FP32 initial-only ranking,
each with two search candidates. The FP32 metadata is pinned to
`47bd3ed2037c8ba0af31ec4cad3a47fe1182af171845406aba01c5107f4b24b7`.
No tuning, feedback model calls, GPU or upstream Laya calls are planned.
Search receives declared cases only. Diagnose receives its selected choices,
those same cases, a candidate budget of four, and the following 31 generic probes:

`MinInt64, MinInt64+1, -107, -33, -15, -14, -13, -9, -4, -3, -2, -1, 0,
1, 2, 3, 4, 5, 6, 7, 8, 9, 13, 14, 15, 17, 31, 33, 107, MaxInt64-1, MaxInt64`.

Diagnose is a separate deterministic optional operation. It compares finite
output vectors, not just pass counts, against the chosen body. It observes the
reference first and then ascending remaining masks. It reports all unobserved
space, type rejections, case-indistinguishable candidates, bounded probe
agreement and the first input distinguishing each case-indistinguishable
alternative. Witness outputs are candidate observations, never invented expected
answers or authorization to change intent. Probe agreement remains unresolved.

Planned denominators are 144 SDK searches, 144 diagnoses and 576 candidate
observations, with 144 initial model predictions. Each emitted distinct Go
program is compiled/executed on all 31 probes; actual counts are measured.
An independent integer-state oracle checks selected/alternative values, witnesses,
case vectors, rejections and finite counts. The original intended mask and
evaluator-only cases are excluded from both search and diagnosis inputs.
Source/model/cohort/preregistration hashes and all observations are retained.
Report diagnostics latency separately from search. This SDK study performs no
native invocation and does not establish deployed compiler integration, host CPU
utilization, an end-to-end speedup or all-input correctness.
