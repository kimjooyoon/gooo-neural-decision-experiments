# Hugging Face model publisher

`publish-hf` uploads the exact file set in the separately reviewed
`gooo/public-export-allowlist/v1` JSON. The bundle includes `README.md` as the
Hugging Face displayed model card and `model-card.json` as the machine-checked
scope and parity binding. It first runs
`tools/verify-public-export`, checks the fixed eight-operation bundle roles and
paths, and re-hashes every file before networking. Files outside the allowlist
are never uploaded. The bundle and base64 NDJSON request are bounded to 3 MiB
and 4 MiB respectively.

The default is read-only preflight:

```sh
go run ./tools/publish-hf --bundle PATH_TO_VERIFIED_BUNDLE --allowlist PATH_TO_EXTERNAL_ALLOWLIST.json
```

The command validates the public-export receipt, reads a Hugging Face token,
checks `/api/whoami-v2` against the fixed `asketeddy` namespace, and checks the
target repository's main reference. An existing target must be anonymously
visible. The output contains no credential. It does not create a repository or
upload files.

Only run the explicit write mode after the Go parity and export-review gates
have passed:

```sh
go run ./tools/publish-hf --bundle PATH_TO_VERIFIED_BUNDLE --allowlist PATH_TO_EXTERNAL_ALLOWLIST.json --publish
```

The only target is the public model repo
`asketeddy/gooo-ir-operator-tiny-v1`. A new repo is created with `private:false`.
The publisher commits allowlisted files to `main` using Hugging Face's
`application/x-ndjson` commit API and includes the observed main commit as
`parentCommit` when one exists. It does not retry a failed or uncertain commit.
If the request times out, inspect the Hub before starting another commit. After
a successful commit, the command downloads each file anonymously at that
commit and compares its SHA-256 with the allowlist.

Token lookup order is `HF_TOKEN`, `HF_TOKEN_PATH`, `$HF_HOME/token`, then
`~/.cache/huggingface/token`. Tokens are read in-process, never accepted as a
command-line argument, and never included in output or error bodies. The
verifier child process does not inherit Hugging Face token variables. A Hub
HTTP 403 is reported without echoing the response body; it can indicate missing
write permission or a token policy restriction. Hugging Face recommends
scoping fine-grained tokens to the required repository when possible.

This publisher does not create the allowlist or certify that a source has legal
publication rights. Keep the bundle to generated/licensed content, filtered
training provenance, model artifacts, and required public Go parity evidence.

The API format follows the current [Hugging Face Hub OpenAPI specification](https://huggingface.co/.well-known/openapi.md):
[repository creation](https://huggingface.co/.well-known/openapi.json),
[`whoami-v2`](https://huggingface.co/.well-known/openapi.json), and
[model commit](https://huggingface.co/.well-known/openapi.json). See also the
[Hub upload guide](https://huggingface.co/docs/huggingface_hub/guides/upload)
and [token permissions guide](https://huggingface.co/docs/hub/security-tokens).
