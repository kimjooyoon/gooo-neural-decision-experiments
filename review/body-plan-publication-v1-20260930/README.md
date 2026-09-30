# Body-plan run publication and privacy audit

Decision: **PASS** for publication hygiene of
`runs/body-plan-v1-laya-7d626b9-20260930/`.

The audit found the exact allowlisted layout: 10 arms, 128 case directories per
arm, and one compiled replay directory per arm. The payload contains 6,575
regular UTF-8 text files in 1,300 directories, totaling 36,493,567 bytes. There
are no symlinks, hidden paths, binary files, model-weight copies, environment
files, or cache directories. Model artifacts are represented by hashes in the
report.

All 128 raw exchange bundles were parsed. Their 222 raw request objects were
checked against the request field allowlist, and all 222 base64 provider replies
were decoded and inspected as JSON (95,715 decoded bytes). Across the saved JSON
and JSONL records and decoded replies, the audit found no unknown or duplicate
JSON keys. It found zero matches for credential/token patterns, local-machine
path patterns, email addresses, or host/session identifiers. The resource
monitor source accepts a PID argument for local sampling, but the published
sample records contain no PID or host identifier.

The audit scans generated `.gooo` inputs, generator outputs and receipts, Go
projection files, compiled replay sources/tests/output, the report, and the
resource-monitor source. It does not reassess compiler semantics, scores, model
quality, or whether heldout data influenced selection; those belong to the
separate experiment review. Pattern scanning also cannot prove the absence of
all conceivable personal data.

## Reproduce and bind

From the repository root, run:

```sh
go run ./review/body-plan-publication-v1-20260930/audit.go
```

The initial manifest and receipt, created before the run README's quantile note
was corrected, are preserved under [`v1-initial/`](v1-initial/). Their
reconstructed README, manifest, and receipt bytes were checked against the
captured original manifest and receipt SHA-256 values. The current read-only
pass writes a distinct [v2 receipt](receipt-v2.json) and [v2 per-file
manifest](file-manifest-v2.json), leaving both older artifacts untouched.

The v2 receipt records file and directory counts, byte total, raw exchange
counts, privacy scan results, and the hash of the v2 manifest. The manifest
lists each relative path, byte size, content class, and SHA-256. The receipt
also binds the exact auditor source SHA-256.

The current v2 bindings are:

- Auditor source: `c893c3836fbc77ed2749dd6a973cfb2234f64d966820df1f0a2dcd7ea8ed415a`
- File manifest: `aeead564c9ae304684a07dfa14314c5851911e595bc1ab8e2a8d6b0a2b871291`
- Receipt: `56b7365b15634fb4b9bb81659a6b945a23c78345848f902d2ebaeab4b30a115b`

The later four-file native Integer-local correction record received a separate
basic privacy pass at [`native-integer/receipt.json`](native-integer/receipt.json)
with its own [`file-manifest.json`](native-integer/file-manifest.json). All
four allowlisted files were UTF-8 text; there were no token/path/email/host-ID
pattern matches, unknown or duplicate JSON fields, or unlisted files.

The run's `report.json` is the terminal completed report. The retained
`preexecution.json` and `progress.json` are earlier running snapshots; their
bytes are included in the manifest without being presented as terminal status.
