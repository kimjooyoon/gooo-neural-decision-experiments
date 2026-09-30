# Frozen v1 publication

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
