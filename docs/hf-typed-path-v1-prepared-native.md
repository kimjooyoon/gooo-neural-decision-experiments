## Prepared native Gooo paths (2026-10-01 KST)

The Go-only SDK [v0.2.1-experimental](https://github.com/kimjooyoon/gooo-decision-runtime/tree/v0.2.1-experimental)
owns one validated immutable plan snapshot and its compiled fallback. Native
[PR 1114](https://github.com/kimjooyoon/meta-ontology-go/pull/1114) uses that
snapshot for source binding and search, preserving every combined candidate's
existing scope/type checks, final native verification and zero source writes.
It introduces no global cache, retained test outcomes or additional model call.

The fixed new compound intent combines six decisions: comparison operand order,
Boolean local and assignment target, Integer assignment target, branch bodies
and Boolean-update order. English/Korean, four arms and budgets 8/64 form 16
cells, paired over five balanced-order repetitions. The clean baseline is native
main `9b8b900e32384b5397fb3d08dc545e22cc7530da`; the clean prepared candidate is
`e155148f5ce114ee4c55e720a0e90cc34cc458ce`. These are views of one idea, not
160 independent tasks. No checkpoint is selected or trained on this study.

There are **160 actual native generations, 720 local predictions, zero external
calls and zero optimizer steps**. All 80 old/new pairs retain identical exact Go
source and search semantics. The emitted Go independently executes in 160
isolated packages and matches the selected arena on all 1,440 observations.
Independent intent arithmetic passes 645/720 per version, including 360/360 at
budget 64 and 285/360 at budget 8. Fifty native calls retain a partial result.

Authored structural target-label agreement is separately 390/480 per version.
Even at full budget, arithmetic succeeds on 360/360 inputs while target-label
agreement is 220/240. Equivalent finite behavior does not establish every
requested internal variable/path choice. The target labels and nine independent
arithmetic inputs are inspected after selection, never passed into the model or
finite search API. The seven stated cases alone guide bounded search.

| Local five-repetition comparison, median | Baseline | Prepared |
| --- | ---: | ---: |
| Native processing stages, excludes startup | 3.202 ms | 1.562 ms |
| Whole native child wall time | 9.839 ms | 8.448 ms |
| Lifetime child peak RSS | 20,807,680 bytes | 19,308,544 bytes |
| Process CPU/wall, relative to one core | 89.96% | 86.04% |

The median paired stage saving is 1.590 ms. Balanced ordering limits systematic
first-run bias, but this remains one local workload on one host. RSS is an OS
child-lifetime high-water value, not packed weights or prepared-object resident
size; CPU/wall is not the host's utilization increase. New `plan_prepare_ms` is
separate, while older binding/search stages included preparation. Compare total
processing before attributing per-stage differences.

The compact comparison/cohort/independent-Go evidence accompanies this model
repository. Full 643-file raw captures, the initial missing-entity fixture
failure, preliminary smokes and the runner are in the
[public raw capture](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/853650502d93af88d4da84cacbe1ea768c3d72e8/runs/prepared-native-conditional-20261001).
Runner/workload source is frozen at research revision
`61f12318262e69520ab4af6663afb95004c494dc`.
The first authoring failure was refused before inference and corrected by
declaring the fixture's Integer entity. All nine model weight bundles stay
unchanged; the update improves compiler integration and measurement, not model
first-choice accuracy. Earlier model-card sections retain historical observations.
