# Whole-candidate search: frozen-weight follow-up

The initial 64-request fit and 96-request development observations are already
closed. Its collector is `5f0caf53f18747d44f79734a2768c4dbe238a947`; weights SHA256 is
`cf00ccc83d17d28ed73fcb869366151a48ffccd3aa8ca8e635aabf19810b9e78`.
This follow-up changes search, with zero fitting updates. The already observed
cohort remains a development cohort.

## Motivation and comparison

The initial scorer improved first-choice finite completion but ranked duplicate
commutative constructions together. On new constants, full-mask attempts were
52 versus 48 for the deterministic SDK. Preserve that result.

For all 160 original requests, independently reconstruct features and every
candidate's interpreter outputs, load the exact original artifact, and perform:

- Actual deterministic and model-ranked search, budgets 1 and 8.
- The same two ranking policies with equal-descriptor skipping, budget 8.

Each model arm makes one actual prediction before reading finite outcomes. The
search evaluates programs on three selection inputs; the selected body is then
evaluated on all eight original inputs, five of which are selection-disjoint.
Compare unfiltered search outcomes and predictions with the initial frozen
observations. Compare deterministic attempts with the existing SDK search.

## Exact reuse boundary

The admitted plan describes `let v=input; two integer updates; return v`.
Its 48-byte descriptor retains each operation, leaf kind, exact constant and
operation order. Addition/multiplication operand order is canonicalized under
Go int64 semantics; subtraction orientation is preserved. Equal descriptors
therefore describe equal pure integer computations in this restricted profile.
Distinct descriptors can still describe equal functions; no completeness of
this equality detector is claimed.

Only previously evaluated equal descriptors are skipped. Budget counts actually
evaluated bodies. The search record keeps all eight declared structural masks,
actual attempts, and each skipped mask with its evaluated representative. A
skipped mask remains unattempted in the ordinary search receipt. Source/feature
preparation still checks every composed candidate. This optimization saves finite
evaluation work; it does not eliminate preparation or all allocations.

## Reporting

Report full-mask and deduplicated results separately, including both deterministic
and learned controls. Record actual model-call count, body/case count and elapsed
search time. Replay timing covers preparation, artifact hashing, ranking and
interpreter search; tiny `Predict` timing alone excludes those costs. No native
compiler integration or training improvement follows from this replay alone.

Local unit checks cover deterministic fallback masks and budgets, changing test
expectations without changing predictions, cancellation, rejected constants,
concurrent workspaces and equivalence-only skipping. The replay is suitable for
Linux CI to check the fixed FP32 arithmetic and full candidate ranking.
