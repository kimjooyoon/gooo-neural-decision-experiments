# Public model export verifier

`verify-public-export` checks a frozen v1 bundle against a separate, reviewer-approved allowlist. It rejects unlisted files, symlinks, digest changes, common credential and local-path patterns, unexpected run artifacts, invalid model files, and mismatched dataset, training, or external Go-audit bindings. It does not upload anything.

Build and run it from the repository root:

```sh
go run ./tools/verify-public-export BUNDLE_DIR ALLOWLIST_JSON
```

The allowlist uses schema `gooo/public-export-allowlist/v1`; every file entry contains a repository-relative `path`, lowercase `sha256`, and `role`. The allowlist must live outside the bundle. The v1 bundle has a fixed 17-file core:

| Role | Required path |
| --- | --- |
| `documentation` | `README.md` |
| `dataset_manifest` | `data/synthetic-ops-v1/manifest.json` |
| `synthetic_dataset` | `data/synthetic-ops-v1/dataset.jsonl` |
| `model_contract` | `model-contract.json` |
| `generator_source` | `cmd/dataset/main.go` |
| `training_source` | `runs/pilot-mps-20260930-v1/training-source.py` |
| `training_record` | `runs/pilot-mps-20260930-v1/preexecution.json` |
| `training_report` | `runs/pilot-mps-20260930-v1/public-training-summary.json` |
| `model_card` | `model-card.json` |
| `external_verification` | `runs/pilot-mps-20260930-v1/go-audit.json` |
| `python_parity` | `runs/pilot-mps-20260930-v1/go-parity.json` |
| `model_metadata` / `model_weights` | `runs/pilot-mps-20260930-v1/models/{fp32,ptq_ternary,qat_ternary}/{model.json,weights.bin}` |

The verifier pins the reviewed dataset, manifest, model contract, generator, training source, pre-execution record, and final Go audit hashes. It recomputes JSONL split counts and split digests; checks the manifest's dataset, contract, generator, class, and language bindings; binds the pre-execution record to the included source and data; and requires a filtered training summary that agrees with the external Go test scores. The summary schema is `gooo/tiny-ir-decision-public-training-summary/v1` and includes only the pre-execution, dataset, and source digests plus per-variant `test_correct`, `test_planned`, and `test_accuracy`.

The machine-readable `model-card.json` must use `gooo/tiny-ir-decision-model-card/v1` and state `model_origin: independently_initialized`, `laya_finetuned: false`, `task_scope: closed_eight_operation_typed_binary_ir_selection`, and `arbitrary_source_generation: false`. Its `external_go_parity_status` must be `PASS`; `external_verification_sha256` must match the exact included Go-audit bytes. The audit must report all 32 parity rows for each variant, stay within its declared error tolerance, bind the included model metadata and weights, and agree with all 256-row test scores.

The audit depends on reviewed source and saved run artifacts. Hashes establish byte identity, not who produced an artifact or what data was used. Pattern scanning can miss secrets or sensitive data, and cannot establish license rights, privacy of learned weights, model quality outside the frozen finite evaluation, or runtime correctness beyond the saved parity and test evidence. The task remains a closed eight-operation typed-IR selection problem; it is not arbitrary source generation and it is not a Laya fine-tune. A verifier pass is a bundle-integrity result, not permission to publish.
