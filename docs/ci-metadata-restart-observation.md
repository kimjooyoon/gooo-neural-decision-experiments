# PR metadata restart observation

An actual publication-side bottleneck was observed during the incremental native
feature. This is separate from model inference and finite search performance.

The native workflow accepts `pull_request` `edited` events. Its concurrency group
is the PR and workflow, with `cancel-in-progress: true`. Updating an already
submitted PR body therefore starts full validation and cancels work for the same
source head. Model judgments do not cause this restart.

## Recorded runs

| Run | Source | Outcome / cause |
| --- | --- | --- |
| 36797800264 | f6323846… | Module graph fixed-point substep failed on two old SDK checksum lines; superseded after the source repair |
| 36798203105 | 4ca583b1… | Format, vet, unit and semantic checks passed; race/remaining jobs cancelled after PR body was edited |
| 36798980665 | 4ca583b1… | New authoritative run for the unchanged repaired source, started 2026-10-01T01:00:26Z |

The second body edit at 01:00:23Z added measured dogfood results. It was an agent
publication action, not a user review requirement. The resulting cancellation
must not be represented as a race-test failure or successful complete CI.

The immediate workflow is to finish the body before opening/submitting the PR,
publish additional experiment details in the research repository, and hold PR
metadata stable while consuming the exact run and proof. No required check is
removed. Guardian and required approving reviews remain absent.

Removing `edited` without further analysis would also remove the existing
automatic promotion refresh signal: the configured design refreshes an existing
main promotion PR through a body marker. Job-level skip conditions alone are
insufficient because workflow concurrency may cancel the previous run before
those conditions execute, and skipped checks may obscure the proof selector.
A future trigger redesign must distinguish irrelevant metadata from source/base
changes and promotion refresh, while preserving exact-run six-check proof
selection. This observation does not implement that redesign or claim that the
permanent Actions App configuration is available.

The model study remains fixed at 288 native calls and 1,296 fresh predictions.
CI retries and synthetic unit predictions are separate from those measurements.
