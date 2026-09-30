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
