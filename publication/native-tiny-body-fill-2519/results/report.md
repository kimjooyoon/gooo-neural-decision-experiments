# Native tiny_go body-codegen smoke results

## Scope and integrity

Six one-shot CLI cells ran sequentially: two fixed plans (arithmetic and Boolean) crossed with fp32, ptq_ternary, and qat_ternary. There were no warmups, retries, or external provider calls. Each cell made one local model prediction, preserved raw stdout/stderr and a receipt, and verified the expected compiler, executable, model metadata, and weight hashes before and after execution.

The native source pin is `2519f27a8d7de834687bc32318c29662e75f5bd9`. The CLI SHA-256 is `9cf07b11464080c97fc6a9cc7c52af5fdddbf759ea6e1c563962234a4da62d13`. The machine-readable capture details and raw hashes are in `capture-summary.json`; each cell's full evidence is in its named directory under this folder.

## Choices and finite-suite scores

Each plan has six explicitly listed cases. The arithmetic choices are `sum` (`input + 2`), `difference` (`input - 2`), and `product` (`input * 2`). The Boolean choices are `< 0`, `<= 0`, and `== 0`. Cases include signed int64 endpoints. These same cases score candidates and validate the final generated Go; they are not a held-out suite.

| Fixture | Variant | Decision recorded by receipt | Receipt candidate finite score | Final selected candidate | Final score |
|---|---|---|---:|---|---:|
| Arithmetic | fp32 | Deterministic fallback: `sum` (`OPERATION_NOT_OFFERED`) | 6/6 (100%) fallback | `sum` | 6/6 (100%) |
| Arithmetic | ptq_ternary | Model-applied `difference` | 0/6 (0%) | `sum`, after local finite-suite correction | 6/6 (100%) |
| Arithmetic | qat_ternary | Model-applied `difference` | 0/6 (0%) | `sum`, after local finite-suite correction | 6/6 (100%) |
| Boolean | fp32 | Deterministic fallback: `less_than` (`OPERATION_NOT_OFFERED`) | 5/6 (83.3%) fallback | `less_equal` | 6/6 (100%) |
| Boolean | ptq_ternary | Deterministic fallback: `less_than` (`LOW_CONFIDENCE`) | 5/6 (83.3%) fallback | `less_equal` | 6/6 (100%) |
| Boolean | qat_ternary | Deterministic fallback: `less_than` (`OPERATION_NOT_OFFERED`) | 5/6 (83.3%) fallback | `less_equal` | 6/6 (100%) |

The fallback receipt records the chosen fallback candidate, not the model's raw operation prediction. That raw label is absent from the response, so the four fallback rows do not measure model-choice accuracy. Only two rows expose a model-applied operation: both arithmetic ternary variants chose `difference`, which scored 0/6 before the compiler's local finite-suite correction. The corrected final candidate passed all six cases in each row.

## Timing and process observations

| Fixture | Variant | CLI active wall | Model load | Local decision | Body-codegen total | Child user CPU | Child system CPU | Child max RSS |
|---|---|---:|---:|---:|---:|---:|---:|---:|
| Arithmetic | fp32 | 501.513 ms | 0.522 ms | 0.080 ms | 2.687 ms | 10.217 ms | 13.225 ms | 16,564,224 B |
| Arithmetic | ptq_ternary | 10.868 ms | 0.181 ms | 0.050 ms | 0.920 ms | 5.438 ms | 3.654 ms | 16,433,152 B |
| Arithmetic | qat_ternary | 9.990 ms | 0.182 ms | 0.044 ms | 0.860 ms | 5.102 ms | 3.283 ms | 16,728,064 B |
| Boolean | fp32 | 9.668 ms | 0.209 ms | 0.038 ms | 0.791 ms | 4.685 ms | 3.293 ms | 16,842,752 B |
| Boolean | ptq_ternary | 10.860 ms | 0.198 ms | 0.047 ms | 0.949 ms | 5.518 ms | 3.731 ms | 16,777,216 B |
| Boolean | qat_ternary | 10.581 ms | 0.165 ms | 0.050 ms | 0.899 ms | 5.444 ms | 3.623 ms | 16,908,288 B |

CLI wall covers each child from process launch through exit, including startup and model load. The compiler-reported intervals are shown separately. CPU and max RSS are single-child observations; RSS is Darwin bytes. The first cell's 501.5 ms is retained as measured, with no warmup or cause assigned. These observations do not estimate host CPU increase, GPU work, or model-kernel throughput.

## Independent generated-Go replay

After the six native captures ended, Go 1.27 compiled and ran each generated program against the same six declared cases for its fixture. Result: 6/6 programs compiled, 36/36 cases observed and passed, 0 failures. The independent harness changed the generated package clause from `tiny_smoke` to `main` and appended a finite-case entry point; it preserved the generated activity function body. Full per-cell source, build, execution, and case evidence is under `../independent-replay/`; the top-level receipt is `../independent-replay/independent-replay.json`.

This verifies execution on the declared examples only. It does not establish unseen-case accuracy or behavior over all int64 inputs.
