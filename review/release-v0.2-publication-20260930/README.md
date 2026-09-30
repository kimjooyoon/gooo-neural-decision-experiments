# Frozen public v0.2 download verification

`verification-receipt.json` records a completed anonymous verification of the
public repository, v0.2 tag, release metadata and all four fresh asset downloads.
The source SHA-256 of the executed `verify_release.go` is
`3ce558c5b1748a3004fc3a218cd8b4286c66bf5a2e56db8e50c4f550ad5d42df`.

The helper checks exact asset names, sizes, expected SHA-256, API digests and
the outer checksum manifest. It writes only canonical public GitHub URLs into
the receipt. It does not execute downloaded binaries, upload files or mutate
the remote repository. The separate [build verification receipt](../../publication/release-independent-verification-v0.2.json)
records the actual nine Darwin CLI/model smoke executions.

This is a frozen one-time audit helper with a fixed relative output path.
It writes `review/release-v0.2-publication-20260930/verification-receipt.json`,
replacing that local file if present. Replay only in a fresh directory with a
copy of the exact helper at its original relative path. Keep this evidence
checkout and its saved receipt unchanged. A failed replay retains its raw
download capture privately; a successful replay removes its temporary capture.

The recorded receipt SHA-256 is
`2dd918f56ad5c6f2ac4c0eb69ca7e051b53489182589461b3bd75514858b9eef`.
Linux assets were integrity-checked but not executed on this macOS host.
