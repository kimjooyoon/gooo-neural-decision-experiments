# Reproducible experimental release packager

`package-release` builds standalone archives for these three targets:

- `darwin/arm64`
- `linux/amd64`
- `linux/arm64`

Each archive contains `gooo-decision`, `gooo-decision-stream` and
`gooo-body-compose` binaries,
the MIT license, a short usage README, `model-contract.json`, and the validated
`fp32`, `ptq_ternary`, and `qat_ternary` model bundles. It also contains
`SHA256SUMS` and `SOURCE-MANIFEST.json`. The manifest binds the source commit,
Go version, target, build flags, and every payload path, size, and SHA-256. The
checksum file covers the payload and source manifest; it does not checksum
itself. A second `SHA256SUMS` in the output directory covers the three archives.

Build from a clean committed revision and choose a new output directory outside
the repository:

```sh
go run ./tools/package-release \
  --source-sha COMMIT_SHA \
  --output /tmp/gooo-ir-release-v0.2.0-experimental
```

The packager requires `--source-sha` to match `HEAD` and rejects staged,
modified, or untracked repository files. It also requires the output directory
not to exist, and it never replaces existing files. It uses fixed command,
documentation, contract, and model paths; it does not accept an arbitrary file
list. The model bundles are loaded by the repository's strict model validator
and each metadata file must bind the expected variant and sibling weights.

Before validating models or compiling commands, the packager materializes the
requested commit into a fresh temporary directory from `git archive`. It
compares every archived path and blob hash with the commit tree, requires only
regular files and directories, and enforces entry and byte limits. This makes
ignored Go files in the checkout ineligible as build inputs. A native model
validator and all three target command sets are built from that temporary
source tree. The temporary source and validator directories are removed when
packaging finishes.

Builds use `CGO_ENABLED=0`, `-trimpath`, and `-buildvcs=false`, with module
workspace and network module resolution disabled. CPU feature baselines are
pinned to `GOAMD64=v1` and `GOARM64=v8.0`, and Go's persistent environment file
and experiment overrides are disabled. The manifest records the command
packages, build flags, environment controls, and Go version. The three targets
are built sequentially. Archives use sorted regular-file entries, fixed file modes and
timestamps, and gzip with maximum compression. Rebuilding from the same commit
and Go toolchain should produce identical archive hashes.

This command only writes local files. It does not upload, tag, sign, notarize,
or create a GitHub release. These experimental binaries are unsigned.
