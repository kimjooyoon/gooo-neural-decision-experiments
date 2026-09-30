# Native tiny-go capture results — revision 2

The run used native source commit `e7dc8198600f27fa3d6fc07d2a8fd8b6a4af3b27` and CLI binary SHA-256 `5e7f75badf8079d15d971bc05f65c08516a9b79c4bb0539036ccb318c093f79c`. The frozen run-plan SHA-256 is `1c5a86b321eacf9549d03f4b7df2c94f8ea8171b7dbb04f384094895869c3a58`. The six tiny-go cells ran once in frozen order, followed by two no-model baselines. There were no warmups or retries.

## Capture accounting

- Planned CLI children: 8; CLI children that exited successfully: 8.
- Tiny-go local predictions: 6; external-provider calls: 0.
- Offline baseline local predictions: 0; external-provider calls: 0, known.
- The six tiny-go wrappers exited successfully. Both offline-baseline wrappers exited 2 after writing their raw capture and invocation receipt; their CLI children exited 0.
- The two wrapper errors are limited to the completeness validator reporting `offline baseline completeness scope is not deterministic and local` and `completeness receipt counts tiny_go as a Laya decision`. The raw CLI output and wrapper error files are preserved unchanged. No retry or rewrite was made.

## Tiny-go choice results

| Cell | Raw operation | Gold operation | Applied | Provider candidate score | Final TDD score |
|---|---|---|---:|---:|---:|
| arithmetic-fp32 | `or` | `add` | no | fallback `sum`: 6/6 | 6/6 |
| arithmetic-ptq_ternary | `subtract` | `add` | yes | `difference`: 0/6 | 6/6 |
| arithmetic-qat_ternary | `subtract` | `add` | yes | `difference`: 0/6 | 6/6 |
| boolean-fp32 | `add` | `less_equal` | no | fallback `less_than`: 5/6 | 6/6 |
| boolean-ptq_ternary | `equal` | `less_equal` | no, low confidence | fallback `less_than`: 5/6 | 6/6 |
| boolean-qat_ternary | `add` | `less_equal` | no | fallback `less_than`: 5/6 | 6/6 |

Raw-operation agreement with the fixture gold was 0/6. The model applied an operation in 2/6 cells; both applied choices scored 0/6, or 0/12 combined. The four deterministic fallback candidates scored 21/24; this is fallback performance, not model-choice accuracy. The raw operation mapped to an offered candidate in three cells and those candidates scored 3/18 combined. TDD-selected candidates passed all 36 finite cases.

These six cases are the declared candidate-scoring suite. They are reused for candidate correction and final checking, so the results are not a holdout or a full-domain correctness claim.

## Timing and process observations

Each row is one sequential child-process observation with no warmup or replication. CLI active wall time ranged from 6.209 ms to 463.786 ms; median was 6.618 ms across all eight rows. The first tiny-go invocation was the large startup observation. Tiny-go model-load timings were 0.121–0.576 ms, and local decision timings were 0.035–0.088 ms. The CLI's `bodycodegen_total_ms` excludes model-load time; `tiny_model_load_ms` is reported separately. Only `cli_active_wall_ms` spans the full child process, including model loading.

The runner recorded 37.972 ms child user CPU and 30.585 ms child system CPU in total. Maximum per-child RSS was 16,990,208 bytes. These are per-process observations from the eight individual children; they do not measure host-wide CPU increase or steady-state model-kernel latency. See `capture-summary.json` for each row's measurements and raw artifact hashes.

## Independent generated-Go replay

The six source-bound tiny-go outputs are queued for independent compile and execution. The two offline rows will remain unknown in the fixed 48-case replay denominator until their saved raw receipts are independently checked under the declared completeness contract. No source or raw capture is changed by that replay.
