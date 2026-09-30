# Multi-node body evaluation

This Go runner compares the same frozen typed programs under deterministic
fallback, the three tiny-model variants, and an optional cached Laya reference.
Each initial choice is paired with training-only finite candidate search using
the same maximum attempt budget. Search does not call a model again and is not
a learned feedback policy. Gold operations and heldout cases never enter the
selection/search API.

The caller must pin the cohort SHA-256, source revision, native Gooo binary and
physical Go toolchain. Output must be a fresh directory. The runner saves
preexecution counts, every selected cell, exact source/CLI outputs, and the final
planned/observed/unknown denominators. It typechecks each body through native
`gooo body-codegen`, then compiles and executes each arm's generated Go against
both case splits. Per-case runtime markers must be unique and complete; their
scores must agree with the typed-plan interpreter. Incorrect behavior remains
an observed result. A compiler failure or missing runtime marker is unknown,
never counted as correct.

Child processes have deadlines, bounded output, and owned process groups on
macOS/Linux. Cancellation kills the compiler/test group and waits for its direct
child and I/O. Total declared training work is capped at five million case
evaluations. The executable hashes are rechecked after the run. Evidence records
pipeline wall time; it does not measure process CPU/RSS or individual latency
quantiles.

The optional Laya connector is Go HTTP. Its reference model remains in the
existing upstream PyTorch service; no new model cache or environment is copied.
Only literal loopback HTTP is accepted, redirects are disabled, and replies
must identify multilingual routing and exact finite operation probabilities.
The health revision is pinned before any POST. Each request has an eight-second
budget and is attempted once. Unresolved requests stop the capture and preserve
bounded raw reply bytes; the runner does not restart or retry them.

```sh
go build -trimpath -o /tmp/gooo-body-plan-evaluation ./tools/evaluate-bodyplans
/tmp/gooo-body-plan-evaluation \
  --cohort-sha256 FROZEN_COHORT_SHA \
  --source-revision CLEAN_SOURCE_COMMIT \
  --gooo-bin PINNED_GOOO --gooo-sha256 GOOO_SHA \
  --go-bin PINNED_GO --go-sha256 GO_SHA \
  --output-dir NEW_DIRECTORY
```

The corpus contains scenario variants across eight declared families. Arm cells
and repeated cases must not be counted as additional unique experiments or as
100 distinct metaprogramming techniques. Native Gooo's current pure profile
has a known integration question for a `let` initialized by a bare integer
literal: it may infer Go `int` instead of the plan's `int64`. Such a failure must
be captured and resolved in the compiler, rather than disguised in an oracle.
