# Public Go module consumer verification

This consumer fetched the public
[`gooo-decision-runtime@v0.1.0-experimental`](https://github.com/kimjooyoon/gooo-decision-runtime/releases/tag/v0.1.0-experimental)
through Go's normal module mechanism, without a local `replace`. The version
resolves to `ea52a698d8b2d7a10549ff98160f203a9d36ec31`; `go mod verify`
passed. The module checksum is recorded in `go.sum` and `receipt.json`.

The consumer independently checks the module's three production source files
and license against fixed origin digests. It loads the three existing pilot
bundles, independently hashes their weight files, builds typed `add` IR, and
runs 64 concurrent predictions per bundle across four workers. Each worker owns
its 1,248-byte workspace. All three `PredictInto` allocation checks returned
zero; response construction through `Decide` is outside that allocation claim.

This is an import/inference/concurrency verification, not an accuracy experiment,
latency benchmark, CPU/RSS measurement, native compiler integration, or Laya
call. Model weights remain in the shared pilot directory and are not copied
into the SDK or this consumer folder. The SDK's 11 tracked source/test/doc files
at the versioned commit total 46,337 bytes; Git storage and compiled binaries
are excluded from that source-size figure.

## Reproduce without replacing the frozen receipt

From this directory, set `taskModelRoot` to the existing pilot's `models`
directory (containing `fp32`, `ptq_ternary`, and `qat_ternary`), then run:

```sh
go mod download
go mod verify
go run . \
  "$taskModelRoot/fp32/model.json" \
  "$taskModelRoot/ptq_ternary/model.json" \
  "$taskModelRoot/qat_ternary/model.json" \
  "$(go list -m -f '{{.Dir}}' github.com/kimjooyoon/gooo-decision-runtime)" \
  > receipt-replay.json
```

The recorded `receipt.json` SHA-256 is
`cd262363a8c8f89ebb811158b2dae21c5bb1399d708be6220d1f592062135904`.
The receipt intentionally contains no local paths. A replay should keep its own
result rather than overwrite this recorded execution.
