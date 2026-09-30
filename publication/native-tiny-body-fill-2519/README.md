# Native tiny_go body-codegen smoke

The six planned CLI cells completed in sequence: two fixed `.gooo` fixtures (`arithmetic`, `boolean`) against the three shared model variants (`fp32`, `ptq_ternary`, `qat_ternary`). There were no warmups or retries. All six processes exited successfully, were bound to the pinned compiler and model bytes, and made one local model prediction apiece. The Laya endpoint and key were absent from each child environment; external provider calls: 0.

The arithmetic fixture passes a selected expression through a local assignment, an `if/else`, and two updates. Its choices have distinct `+`, `-`, and `*` roots. The Boolean fixture makes a local update under a hole condition and compares `<`, `<=`, and `==`. Each plan declares six explicit finite input/expected pairs, including signed int64 boundaries. The same declared cases were used for candidate scoring and the later generated-Go replay; there is no separate held-out suite and no full-domain claim.

`results/capture-summary.json` records raw-cell hashes, candidate scores, source/model pins, and per-process observations. `results/report.md` explains the outcomes and timing scope. `independent-replay/independent-replay.json` records a separate Go 1.27 compile and execution of all six generated programs against the same 36 finite cases. The independent harness changed the package clause and appended a small case runner; the generated activity function body was preserved.

## Results at a glance

- Native capture: 6/6 cells, source and model hashes verified before and after each child; 6 local model predictions, 0 external calls.
- Generated Go: 6/6 programs compiled; 36/36 declared finite cases passed.
- Two arithmetic ternary variants applied `difference` (0/6), then local finite validation selected `sum` (6/6).
- Four receipts used deterministic fallback. Their receipt candidate scores are recorded separately; the raw model operation label is not exposed, so those scores are not model-choice accuracy.
- The first CLI wall observation was 501.5 ms; the remaining five were 9.7–10.9 ms. No warmup was run, and no cause is assigned to that difference.

## Measurement scope

`cli_active_wall_ms` covers one child process from launch through exit, including cold startup and model loading. `tiny_model_load_ms`, `tiny_decision_ms`, and `bodycodegen_total_ms` are intervals reported by the compiler. Child CPU and max RSS are single-process observations around each isolated child; RSS is bytes on Darwin. These numbers are not host CPU increases or model-kernel timings. The separate generated-Go compile/replay happened after all six captures and has its own receipt.

## Pins

- Compiler source commit: `2519f27a8d7de834687bc32318c29662e75f5bd9`
- Native CLI SHA-256: `9cf07b11464080c97fc6a9cc7c52af5fdddbf759ea6e1c563962234a4da62d13`
- Runner source SHA-256: `999d3d5dc8dcdab63d9777365c31f3e266a933c9bbc6f02c415e3573af3e27f7`
- Runner binary SHA-256: `514e8dc873cb1cd40b72d9c6c32e06a5e6b3994c903e9cb7248480adf85dc0ab`
- Independent replay source SHA-256: see `independent-replay/independent-replay.json` and `replay.go`
- Model metadata / weights SHA-256:
  - fp32: `c5867ee872cbf0fb8227015bd3e94540415786437704ed01b66e6bb0b421c930` / `854da0f8fc0dcf5560d1f1b8838ea30081752020dbc2005772207217c5d1d17a`
  - ptq_ternary: `5766a32a9d70ef16c895e7cf8765d7884d59ab6241fcfa225ac2291731a77a4e` / `3c60ab7154f02f04a94b1bc68274e05c4ae86f10185b19dbe7251c26610a5c8b`
  - qat_ternary: `4746fc7cbd2864e9536b43d779c7ecc39aa35f8f6f64943c73b951f9e967c8a7` / `bd4444d5233f07c18168a4dfa0053afde180ce97e390a993e4e0fbb33fb88337`

The original pre-capture README is preserved byte-for-byte at `pre-capture/README.md` (SHA-256 `6cde5ab2341fb4ec4ae21bc0b17d289db6c452d5a037c74081dafb66eeb1e7e5`).
