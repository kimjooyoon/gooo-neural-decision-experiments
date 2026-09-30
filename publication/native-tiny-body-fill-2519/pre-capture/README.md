# Native tiny_go body-codegen smoke

This directory holds two finite, source-only `.gooo` fixtures, their closed fill plans, and a standard-library Go wrapper. It has not made any tiny_go or external-provider calls yet.

The arithmetic fixture passes the chosen expression through a local assignment, an `if/else`, and two updates. Its three choices have distinct `+`, `-`, and `*` roots. The expected values are explicitly listed in `fixtures/arithmetic.plan.json`, including signed int64 boundaries and Go int64 wraparound. The Boolean fixture makes a local update under a hole condition and compares `<`, `<=`, and `==` on negative, zero, positive, and signed-boundary values. Both plans contain only the declared finite suite; there is no held-out suite.

`smoke-runner` is built from `smoke.go` with Go 1.27, offline settings, and `-trimpath`. It accepts one fixture and one shared trained bundle per process. It launches exactly one `gooo body-codegen` child, removes `GOOO_LAYA_URL` and `GOOO_LAYA_API_KEY` from that child's environment, stores raw stdout/stderr, and independently recomputes all candidate and selected-case scores. Each cell has a fresh output directory. The wrapper performs no retries or warmups.

`cli_active_wall_ms` measures only the child command interval, including cold CLI startup and model loading. The report's `tiny_model_load_ms`, `tiny_decision_ms`, and `bodycodegen_total_ms` are retained separately. Child user/system CPU and max RSS come from `RUSAGE_CHILDREN` around the sole CLI child; RSS is bytes on Darwin. Model/binary hashing happens outside the CLI wall interval, and no generated-Go compile is mixed into it.

## Pins

- Compiler source commit: `2519f27a8d7de834687bc32318c29662e75f5bd9`
- Native CLI SHA-256: `9cf07b11464080c97fc6a9cc7c52af5fdddbf759ea6e1c563962234a4da62d13`
- Runner source SHA-256: `999d3d5dc8dcdab63d9777365c31f3e266a933c9bbc6f02c415e3573af3e27f7`
- Runner binary SHA-256: `514e8dc873cb1cd40b72d9c6c32e06a5e6b3994c903e9cb7248480adf85dc0ab`
- fp32 metadata / weights SHA-256: `c5867ee872cbf0fb8227015bd3e94540415786437704ed01b66e6bb0b421c930` / `854da0f8fc0dcf5560d1f1b8838ea30081752020dbc2005772207217c5d1d17a`
- ptq_ternary metadata / weights SHA-256: `5766a32a9d70ef16c895e7cf8765d7884d59ab6241fcfa225ac2291731a77a4e` / `3c60ab7154f02f04a94b1bc68274e05c4ae86f10185b19dbe7251c26610a5c8b`
- qat_ternary metadata / weights SHA-256: `4746fc7cbd2864e9536b43d779c7ecc39aa35f8f6f64943c73b951f9e967c8a7` / `bd4444d5233f07c18168a4dfa0053afde180ce97e390a993e4e0fbb33fb88337`

The runner verifies the expected source, CLI, model metadata, and model weights hashes before it starts the child, then checks the binary and bundle hashes again after the child exits. It records hashes and model provenance without copying the trained weights.

## Per-cell invocation

Run each command from this directory, sequentially, only after the pinned native CLI is available. Each output directory must be new.

```sh
./smoke-runner \
  --binary /tmp/gooo-native-tiny-20261001 \
  --binary-sha256 9cf07b11464080c97fc6a9cc7c52af5fdddbf759ea6e1c563962234a4da62d13 \
  --compiler-source-sha 2519f27a8d7de834687bc32318c29662e75f5bd9 \
  --runner-source /tmp/gooo-native-tiny-smoke-20261001/smoke.go \
  --runner-source-sha256 999d3d5dc8dcdab63d9777365c31f3e266a933c9bbc6f02c415e3573af3e27f7 \
  --fixture arithmetic \
  --model-variant fp32 \
  --model /Users/alice/meta-go/research/metaprogramming/gooo-neural-decision-experiments/runs/pilot-mps-20260930-v1/models/fp32/model.json \
  --model-metadata-sha256 c5867ee872cbf0fb8227015bd3e94540415786437704ed01b66e6bb0b421c930 \
  --model-weights-sha256 854da0f8fc0dcf5560d1f1b8838ea30081752020dbc2005772207217c5d1d17a \
  --output-dir results/arithmetic-fp32
```

For the other five cells, change `--fixture` to `arithmetic` or `boolean`, set the variant and matching model path/hashes above, and choose the corresponding new output directory. Preserve any captured nonzero or validation-error cell as-is; do not rerun it into the same directory.

No output from this wrapper proves general accuracy. Results cover only these two declared fixtures and their six finite cases apiece.
