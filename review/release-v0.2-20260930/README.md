# Experimental v0.2 release verifier

This read-only Go helper checks the three target archives produced by `tools/package-release` and compares two separately supplied output directories byte for byte. It accepts only these targets:

- `darwin/arm64`
- `linux/amd64`
- `linux/arm64`

Each archive must contain exactly three CLI binaries, the MIT license, the usage README, `model-contract.json`, the three fixed model metadata/weight pairs, `SOURCE-MANIFEST.json`, and inner `SHA256SUMS`. The outer build directory must contain exactly the three archives and its `SHA256SUMS` file.

The verifier checks archive and tar/gzip normalization, safe relative paths, sorted regular-file entries, fixed modes and timestamps, exact payload allowlists, normalized source manifests, outer and inner digests, target formats, model metadata/weight pins, and source/version/target bindings. It writes a JSON receipt only to a new output file; an existing output is an error.

After the release owner supplies two completed build directories and the source commit pin, run:

```sh
go run ./review/release-v0.2-20260930 \
  --first /path/to/build-one \
  --second /path/to/build-two \
  --source-sha COMMIT_SHA \
  --version v0.2.0-experimental \
  --out /path/to/new-verification.json
```

On a Darwin ARM64 host, append `--smoke-darwin` to extract the verified Darwin archive into a temporary directory and run all three packaged CLIs against the three packaged models using small public synthetic inputs. The smoke uses no network and removes its temporary directory. The verifier never executes Linux binaries; it checks their ELF architecture and archive bindings only.

The caller-provided source SHA and version are matched against the archive manifest and README. This verifier does not authenticate the Git commit, establish that the two output directories came from independent clean builds, or rebuild the source. Those provenance checks belong to the release owner and CI. It does not change Git state, publish assets, or modify either input directory.

## v0.2 verification record

The release artifacts are pinned to source revision `72813219c891285a5c6af43406cb0688d3a8b4c8` and version `v0.2.0-experimental`. The latest verifier source SHA-256 is `523a2bb8c770077a7ad9afc74094af1e8b9e0adf829a14e8b7395f7956dfbaaf`. It checks the exact confidence threshold pinned in each model metadata file: `0.125` for FP32 and QAT, and `0.5429213483146067` for PTQ.

The earlier verifier SHA-256 `e3adff47fd7c765f6c619e0eadc9c627a2373c341a5307b4717f00ccc467c2cf` expected `0.125` for every variant, so it rejected the correctly pinned PTQ metadata. That helper and the failure evidence are preserved in the attempts archive and are described by the path-neutral derivation manifest at `attempts/derivation-manifest.json`.

The corrected verifier passed all three target archives and found the two supplied build outputs byte-identical. Darwin/arm64 smoke passed all 9 combinations of three CLIs and three model variants, with no network use. Linux binaries were validated by archive and ELF metadata only; Linux execution was not performed. The independent verification receipt SHA-256 is `671deb590a389e395e84f5bea1a16649fe159ee65efeda302c3203fad596e4c9`.

The complete raw attempt files are preserved in a private external archive. The public attempt capsules replace local helper/build/output paths with relative paths and placeholders; their manifest records the raw and derived hashes and transformation reasons. The private archive path is intentionally omitted.

The final review corrected one derived byte count in the manifest; its initial
version is retained in [manifest corrections](manifest-corrections/README.md).
All raw evidence, capsule content and execution receipts remain unchanged.
