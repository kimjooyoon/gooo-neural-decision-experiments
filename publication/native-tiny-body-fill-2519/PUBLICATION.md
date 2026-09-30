# Frozen native integration capture

The original text sources, raw CLI results, receipts, and generated-Go replay
evidence are copied byte-for-byte here. `publication-manifest.json` binds every
copied text file. Executables and model bundles were excluded to keep the
repository small; their hashes remain in the original receipts. No weights were
copied. The original temporary evidence remains preserved locally.

The source was an experimental PR revision, not a promoted release. Its focused
tests passed, but [full semantic CI for this revision](https://github.com/kimjooyoon/meta-ontology-go/actions/runs/36743586612)
found that the repository projection's importer searched GOROOT/GOPATH for the
new public Go module. The new module required module-aware dependency resolution.
The original failure is retained separately; later code fixes do not rewrite
these source-bound observations.

This revision also omitted the raw operation label for four fallback decisions.
Their fallback scores cannot be attributed to the model. The next adapter
revision adds the raw label and an explicit pre-TDD application flag; any later
captures belong to their own source and executable hashes.

The two Go runners are standalone entry files. Build each individually, as
`go build -o smoke-runner smoke.go` and `go build -o replay-runner replay.go`;
they are not a single Go package. This folder has a separate module so the parent
experiment's CI does not rerun local resource experiments or combine their
entry points. Preserve all recorded files when reproducing in a new directory.

Both native modes score every candidate before an optional model prediction.
This capture does not show reduced candidate evaluation, faster code generation,
held-out accuracy, or a full-domain proof. The separately compiled programs
passed 36/36 instances of the same declared finite cases.
