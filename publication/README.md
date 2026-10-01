# Public models and executable releases

## Body composer release v0.2

The public [v0.2.0-experimental release](https://github.com/kimjooyoon/gooo-neural-decision-experiments/releases/tag/v0.2.0-experimental)
is bound to source `72813219c891285a5c6af43406cb0688d3a8b4c8`. Each of its three
target archives contains `gooo-decision`, `gooo-decision-stream`,
`gooo-body-compose`, the three unchanged model bundles and source manifests.

- [Independent build verification](release-independent-verification-v0.2.json):
  two supplied builds were byte-identical, with exact archive/source/model
  hashes and 9/9 actual
  Darwin CLI/model smoke executions. Linux executables were inspected only.
- [Anonymous public asset verification](../review/release-v0.2-publication-20260930/verification-receipt.json):
  public repository/tag binding and four fresh downloaded assets matching their
  expected SHA-256, size and API digest.
- [Verifier revision and attempts](../review/release-v0.2-20260930/README.md):
  the original verifier incorrectly assumed the same confidence threshold for
  PTQ and FP32. Its failure, original helper and corrected verifier are retained.
  Attempts that originally contained local paths are published as path-neutral
  derived capsules with raw and derived hashes; complete originals remain in
  a private external archive.
- [Native Integer promotion](native-integer-main-promotion-1106.json): the study
  found a compiler inference mismatch, repaired and merged to Gooo main with
  six required checks, a verified proof artifact and successful post-main CI.

The corrected release verifier is a later audit-tool revision. It does not
change the frozen release binaries, tag, model weights or source manifests.
The small [body example](../examples/adjust-balance/README.md) demonstrates
deterministic fallback and three tiny-model decisions with training cases.
These examples are repository files, not additional release assets or heldout
research results. The optional upstream Laya comparison still uses its existing
PyTorch service; the tiny-model executables need no Python runtime.

## Frozen v1 publication

The external Go audit, 17-file public-export verifier, read-only Hub preflight,
and actual Go publisher completed successfully. The Hub model is public at
[asketeddy/gooo-ir-operator-tiny-v1](https://huggingface.co/asketeddy/gooo-ir-operator-tiny-v1).
All 17 files were fetched without authentication at commit
`1d1741fe6b88d5121e7fdba45fd90cefdd9a6c91` and matched the allowlist hashes.

Records in this directory are separate from the immutable training/audit run:

- `public-export-allowlist-v1.json`: exact paths, roles and SHA-256.
- `public-export-verification-v1.json`: Go bundle gate PASS.
- `hf-preflight-v1.json`: read-only identity/inventory check before creation.
- `hf-publish-v1.json`: actual commit and anonymous file-verification receipt.
- `preflight-corrections-v1.json`: two local gate integration failures fixed
  before any network mutation; no model or dataset bytes were changed.
- `release-independent-verification-v1.json`: two byte-identical builds,
  archive contents/checksums, model/source binding and native CLI smoke checks.
- `github-release-v0.1.0-experimental.json`: public experimental release/tag
  binding and anonymous SHA-256 verification of all four uploaded assets.

The Go executable release is public at
[v0.1.0-experimental](https://github.com/kimjooyoon/gooo-neural-decision-experiments/releases/tag/v0.1.0-experimental).
It targets the frozen code commit `6d306d3aae547cb6f5bffa503303171f4145f4c0`;
later documentation and publication receipts do not change that release.

Earlier independent review records say the full bundle was not assembled at
their review point. They remain unchanged. The publication verifier receipt
is the later evidence of the complete bundle check.

Reassemble the exact bundle into a new directory with Go:

```sh
go run ./tools/assemble-public-export --output /tmp/gooo-public-export-replay --allowlist /tmp/gooo-public-export-replay-policy.json
go run ./tools/verify-public-export /tmp/gooo-public-export-replay /tmp/gooo-public-export-replay-policy.json
```

Normal CI reproduces synthetic data, tests the runtime and publication failure
cases, checks the exact public bundle, and independently evaluates saved
parity/held-out data. It neither trains nor uploads to Hugging Face.

## Prepared bilingual native paths, 2026-10-01

The Go-only [SDK v0.2.1-experimental](https://github.com/kimjooyoon/gooo-decision-runtime/tree/v0.2.1-experimental)
prepares one immutable validated plan snapshot per generation. The
[160-call paired study](../runs/prepared-native-conditional-20261001/) records
720 fresh local predictions, 80 identical baseline/prepared code and search
pairs, 1,440 independent emitted-Go observations and 50 retained partial calls.
Functional completeness and authored structural-label agreement are separate.
No model is trained or selected on this compound fixture.

The [fixed 222-file bundle](hf-typed-path-v1-prepared-native/publication-manifest.json)
is public on [Hugging Face at immutable revision d137202](https://huggingface.co/asketeddy/gooo-typed-path-tiny-v1/tree/d137202a5cd79ec6aa616ec877b0a09dce8b4459).
All 222 files were fetched without authentication and matched their SHA-256 and
byte counts; the [Go verification receipt](typed-path-prepared-public-verification.json)
records 223 read-only requests and zero additional model calls. The prior
211-file publication and all nine weights remain unchanged in their frozen
folders and Hub history. The new compact edition carries the full comparison,
preexecution bindings, eight cohort files and independent Go events; GitHub
retains all 643 raw capture files.

CI reproduces the exact allowlist and independently recomputes captured search,
finite outcomes, paired equivalence, arithmetic, structure agreement and costs.
It does not re-run inference for this historical observation. Failed publication
comparison during a concurrent document edit and the clean repeated validation
are [retained in the validation notes](prepared-publication-validation-notes-20261001.json).
