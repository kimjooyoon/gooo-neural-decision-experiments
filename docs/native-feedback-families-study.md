# Five-family native continuation study

This extends actual compiler dogfood from one compound source to five existing
structural families: references, assignment targets, operand order, branch
layout and statement order. Ten parameterized fallback bodies represent twenty
intended path behaviors and forty bilingual instructions. Each has complete,
sparse or deliberately contradictory selection tests: 120 views, not 120 new
independent experiments. Reserved configurations 64 and 79 were excluded from
training, but prior development probes have used these families/templates.

The source-pinned pilot made ten native calls, zero model predictions and ten
actual Go processes with 190 function evaluations. The full fixed-order matrix
made 1,080 native calls, 1,204 model predictions (244 feedback), and 1,638 candidate
attempts. It actually executed twenty distinct emitted Go programs on 22 inputs
each: 440 function evaluations. The 9,720 policy observations on separate inputs
reuse those actual executions and are not 9,720 independent Go executions.
No training, GPU, upstream Laya or external model call occurred in this phase.

| Arm, each 120 views | Predictions, rank once / feedback | Finite cases passed / 640 | Separate input observations passed / 1,080 |
| --- | ---: | ---: | ---: |
| Parent FP32 | 120 / 177 | 600 | 1,017 |
| New feedback FP32 | 120 / 182 | 600 | 990 |
| New feedback PTQ | 120 / 183 | 600 | 999 |
| New feedback QAT | 120 / 182 | 600 | 990 |
| Deterministic offline | 0 | 600 | 990 |

Every arm selects a finite best option. The deliberately contradictory duplicate
retains its denominator. Sparse witnesses can make both options equally valid
on selected inputs while they disagree on other inputs: finite completion is
not original-intention or unseen-behavior completion. The new models do not
improve on the parent in these separate-input observations.

All 480 rank-once/feedback pairs preserve the same Go, finite passes, separate
passes and attempts. **One decision has only two declared paths.** After the
first failed candidate, one path remains; another ranking cannot change which
path is next. This is useful breadth evidence and an observed redundant-call
case, not evidence that feedback is ineffective for larger composed plans.

Complete native-process wall medians were 5.276 ms offline; 5.453/5.466 ms parent
rank-once/feedback; and 5.394–5.482 ms across new-model arms. CPU-time medians
were 4.554–4.771 ms, and median child peak RSS 16,949,248–17,268,736 bytes. These
are per-process observations in one baseline-first ordering with no repeats;
host utilization and causal speedup are not measured. The raw matrix retains
every observation, including startup outliers.

Go audit reinterprets every candidate and checks source/cohort/model/progress,
counterexample, actual-Go and paired-outcome bindings with zero fresh model/native
calls. Frozen source: `30fbf81b951e12d504aca19d06e8d1a8dc08a865`; native main:
`4dced73b26dde567cb7129f3e4ba5733850196d0`; SDK `v0.2.4-experimental`.

## Observed repairs

The next SDK change records a hashed `ranking_unnecessary` observation with zero
predictions when exactly one unattempted declared path remains. It retains the
counterexample, original model/case/plan bindings and frontier, consumes a round,
and rejects a repeated same-batch reconsideration. Missing or changed models,
invalid CI hints and cancellation still fail. This is implemented and unit-tested
in research; downstream SDK/native deployment and fresh measured comparison are
separate work, not an already measured speedup.

The old SDK receipt's fixed wording says “non-feedback-trained” even when our
new models have actually been tuned on finite feedback. The raw old wording is
preserved. Future receipts describe the original frozen model without asserting
an unobserved training history. This is a provenance-description repair.

The first cohort build referred to an SDK-only document-schema constant absent
from the research package. That compile failure executed zero native calls; the
correct native document schema was supplied and focused race/vet rerun passed.
An argument-order defect was corrected by inspection before actual matrix calls.

[Cohort](../studies/feedback-family-v1/manifest.json),
[pilot audit](../runs/feedback-family-pilot-20261001/audit.json),
[matrix and full captures](../runs/feedback-family-matrix-20261001/report.json).

## Public evidence appendix

The same frozen evidence is public on
[Hugging Face at immutable revision 9b26f9d](https://huggingface.co/asketeddy/gooo-feedback-path-tiny-v1/tree/9b26f9dbe2afba9abb74886c1abfa6cac5ac7ee8/research/native-family-20261001).
The deterministic gzip archive is 986,241 bytes and contains 1,128 allowlisted
raw JSON/JSONL files with per-entry hashes and sizes. Nine publication files
were verified by ten anonymous HTTP requests; the existing 25 core model files
were separately verified at that same revision by 26 anonymous requests. The
appendix adds evidence without replacing the frozen model card or weights.
Packaging and verification execute zero new model/native/Go calls. Local CI
reproduces the archive manifest without credentials or network publication.

The SDK `v0.2.5-experimental` source release passes Go 1.27.1 format/vet/unit/race
checks (run 36814919123). Native integration is tracked separately in
[PR 1122](https://github.com/kimjooyoon/meta-ontology-go/pull/1122). The first
focused native test edit had a missing loop brace; compilation failed before
native execution, the brace was repaired, and bilingual focused race/vet passed.
The first publication tool build had an unused import; no publication occurred
until the corrected tool verified the frozen allowlist and archive.
