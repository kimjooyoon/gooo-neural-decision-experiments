# Comparing arithmetic as the runtime source changes

The bounded file-loader correction exposed a comparison failure in research
CI37166559740: `pinned file differs: model_test.go`. The historical arm64 report
records source `250c935642b59f4f3122360c90286e5557af3da0`; the comparator tried
to verify that inventory against today's working files before reading results.
The original failure and frozen publication remain retained.

The default comparator keeps its exact-source requirement. An explicit
`--source-delta` comparison checks each collection's file sizes and SHA256 values
against that collection's immutable Git revision. Missing revisions, absent
files, changed hashes and invalid source paths fail. Git history must be available.
The current collector now includes the imported `internal/modelfile` dependency
in its recorded inventory.

The v2 report retains both complete inventories, sorted added/removed/changed
file records, and `computational_sources_equal`. A changed inventory remains
false. It separately checks all original model bytes, protocols, paired text,
source/targets, feature/hidden/logit values, candidate rankings and finite
completion curves. The original probability bound and all legacy differences
remain measured. This compares arithmetic across recorded editions; each
file-loader behavior has its own native/replacement experiment.

Focused tests check historical source binding after a working-file change,
incorrect digest/extent/revision, missing/nonlocal source, and explicit inventory
deltas. The research CI requests the versioned mode and fetches the necessary
Git history. Full local/Linux replays are recorded after source freeze.
