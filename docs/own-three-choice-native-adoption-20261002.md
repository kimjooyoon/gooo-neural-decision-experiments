# Three-choice compiler main adoption

Gooo [main 774eabb](https://github.com/kimjooyoon/meta-ontology-go/commit/774eabb226f88a317c523ce4efa24a08032066a9)
adopts public Go SDK v0.2.13-experimental. Feature [PR 1139](https://github.com/kimjooyoon/meta-ontology-go/pull/1139)
and exact-tree promotion [PR 1140](https://github.com/kimjooyoon/meta-ontology-go/pull/1140)
are merged through the normal expected-head squash API. The snapshot has the
live main parent and exactly the dev tree; no force/admin merge is used.

The [feature run](https://github.com/kimjooyoon/meta-ontology-go/actions/runs/36961115466),
[promotion run](https://github.com/kimjooyoon/meta-ontology-go/actions/runs/36963019453)
and [post-main run](https://github.com/kimjooyoon/meta-ontology-go/actions/runs/36964360262)
are terminal success. All six canonical checks pass in each run. Downloaded
`gooo/ci-proof/v5` bundles and provenance receipts pass the Go proof verifier;
promotion authorization is PASS. Main retains strict checks, enforced admin
rules and zero required human reviews. Guardian remains absent.

Both local compiler executables are built from the clean deployed commit with
Go 1.27.1 and released SDK v0.2.13. Their metadata and SHA256 hashes are recorded
in [the adoption receipt](../publication/own-three-choice-native-main-verification-20261002.json).
The installed compiler passes the billing example semantic check.

The optional pinned-v17 implementation-only compatibility experiment fails with
`AXIS_MISMATCH: compatibility is broader than implementation-only`. Its earlier
execution/test steps pass, but the historical certificate is not established or
reused. The verifier and workflow are unchanged. Whole local macOS tests retain
three platform-specific failures in unchanged packages; Linux whole-project
unit/race checks and local impacted-package race checks pass.

This deployment supports the new opt-in three-choice ABI using controlled test
weights. It does not promote a trained default or establish new learned quality.
The [authored preparation](own-three-choice-authored-fixtures-20261002.md) and
[frozen study](own-three-choice-completeness-preregistration-20261002.md) retain
separate corpus, teacher, optimization, native measurement and public model phases.

`tools/three-composition-curriculum` streams concrete source, path documents,
cases, native zero-prediction exports and the three complete inputs into bounded
JSONL journals. It requires a clean pinned runner and adopted clean native binary,
refuses existing output directories, caps individual lines at 1 MiB and combined
raw journals at 768 MiB, retains failed prefixes, and never loads a model.
Its audit reconstructs all fixtures/oracles and reconciles the captured input
bytes/hashes without making new native calls. Actual collection has not yet run
at this source publication boundary.

## First actual collection correction

The first actual attempt makes one successful zero-prediction native export,
then the collector rejects it before creating a dataset view. Context receipts
contain hashes/lengths while the complete export additionally contains text;
the initial collector incorrectly compares the entire two structs for equality.
The repair checks all metadata fields and validates complete text separately.
The exact captured native output now passes a permanent race-tested regression.
The original 10,055-byte failed prefix and [negative receipt](../publication/own-three-choice-first-collection-negative-20261002.json)
remain. Native source, protocol, authored families, cases and targets are unchanged.
No teacher observations or optimizer updates occur in this failed attempt.
