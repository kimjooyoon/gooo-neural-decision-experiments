## Fresh native structural inference (2026-10-01 KST)

The subsequent direct integration uses native compiler source
`d21ce275ec0832e38ad4962ad03f4546c1b46e9f` and public Go SDK
`v0.2.0-experimental` at `e67cb5934d892fbc81be18038be2d624f1a6b5e0`.
The SDK's CI passed before tagging. Native implementation and fixtures are
public in [PR 1112](https://github.com/kimjooyoon/meta-ontology-go/pull/1112).
This supersedes the preceding review's limitation to replaying saved selections:
native `body-codegen --path-plan [--path-model]` now loads a structural bundle
and performs one in-process prediction per decision before finite tests.

The compiler first binds the declared fallback to the authoritative Gooo body
with its existing canonical equivalence witness. It validates all individually
offered options and combined scope/types, ranks finite alternatives, retains
partial results, replaces only an in-memory computes literal, and emits marked
Go with stable semantic identity, native type checking and deterministic replay.
It calls no external provider, even if Laya environment settings are configured.
Disconnection retains deterministic finite search from the declared fallback.

The fresh probe at runner source `b4d4ebe08335ab96427e2add3b012126d4687a3f`
uses one compound body with local reference, branch assignment/layout, and
declaration order: 8 combinations, 2 scope-invalid. Two languages × four arms
(offline/FP32/PTQ/QAT) × attempt budgets 4/8 × full/partial finite contracts give
**32 native calls and 72 fresh native model predictions**. These are views of
one compound experiment, not 32 distinct synthesis ideas. No weights are updated;
the nine published bundles retain their prior bytes.

All 32 exact native emitted Go functions independently execute 10 integer inputs
absent from the search cases, including int64 overflow edges. The 32 Go tests
pass **selected arena/native semantic parity**, while the separate arithmetic
intent observations pass **226/320 cases**. All budget-8 arms pass 160/160 of
those arithmetic observations. At budget 4, offline misses all 40; the models
pass all 60 English cases but only 6/60 Korean cases. Thus a small search budget
can still miss the intended assembly, even with identical initial selected
labels, because ranking weights change subsequent exploration. No claim of
unseen natural-language accuracy or full-domain proof is made.

The deliberately wrong partial contract requests 999 for input 3. With budget 8
the correct arithmetic function still yields a recorded **2/3 = 66.67%** on
that declared suite. Native lowering remains complete. Neither the incorrect
expectation nor the partial fraction is silently repaired or presented as 100%.

| Arm | Median prediction | Median native stages | Median lifetime native RSS |
| --- | ---: | ---: | ---: |
| Offline | no prediction | 1.571 ms | 18,350,080 bytes |
| FP32 | 8.375 us | 1.681 ms | 18,554,880 bytes |
| PTQ ternary | 9.646 us | 1.623 ms | 18,415,616 bytes |
| QAT ternary | 9.833 us | 1.658 ms | 18,415,616 bytes |

Process CPU-time/wall ratios have arm medians 81.49–82.89% of one core. These
are per-child measurements, **not host CPU utilization increases**. The native
stage durations exclude process startup; per-cell process wall time is retained
separately. This single fixed-order local probe has no matched repetitions and
does not establish a speedup. Inference uses no GPU. The packed ternary weights
remain 2,759 bytes; process RSS includes Go runtime, parsing, typing, arenas,
search and decoded tensors and is not the packed-weight size.

Initial fixture preparation rejected unsupported activity-ID syntax and then
caught a manually miscomputed arithmetic expectation before this source-bound
probe. The corrected explicit contract is `5*input+2`, minus 3 for negatives,
plus 3 otherwise. Native full local tests also report failures in unrelated
macOS namespace replacement, symlink diagnostics and source-splitter assertions;
no full local-suite pass is claimed. The new focused race tests passed. Actual
Linux CI results remain independently inspectable on the public PR.

The earlier ad-hoc mixed-language FP32 check made one additional native call
and three predictions; its 2.744 ms stage observation is excluded from the
32-cell primary probe. Publication contains only synthetic fixtures, public
model metadata/weights and selected evidence. No Laya weights, environment,
credentials, private repositories or user paths are included.
