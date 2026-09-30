# Independent review: tiny typed-IR decision model

Review date: 2026-09-30. This is a read-only review of the frozen synthetic dataset, training source, model contract, and current Go runtime. The review did not run PyTorch training, download a model, upload an artifact, or modify files owned by the training or runtime contributors.

## Findings for the pilot owner

### P1 — The trainer does not bind its input to the frozen dataset manifest

`training/train_pilot.py` accepts only `--dataset`. It hashes the selected JSONL bytes into `preexecution.json`, but it does not load the adjacent `manifest.json` or compare the data hash, split hashes/counts, contract hash, or generator-source hash. A different dataset can therefore pass the current row-label and split-overlap checks and be trained as though it were the frozen pilot input. The recorded hash identifies what was read; it does not establish that those bytes match the reviewed manifest. Bind future training invocations to the frozen manifest.

Root confirms that the completed v1 invocation was separately launched only after checking the exact dataset and source hashes, so the missing in-script manifest check did not admit a different dataset in that run. It remains a direct-invocation and future-run integrity gap; root plans a separate v2 training entry point rather than changing frozen v1 outputs.

The committed data currently matches its manifest exactly. Independent verification found 2,048 rows (1,536 train, 256 calibration, 256 test), all eight labels balanced, 1,024 English and 1,024 Korean rows, and exact matches for dataset, split, contract, and generator-source SHA-256 values.

### P1 — Split-safety checks rely on `assert`

The trainer uses `assert` for the MPS requirement, the epoch and batch bounds, valid labels/splits, and cross-split prompt/template disjointness. Running Python with optimization enabled disables these checks, including the leakage check. Use explicit exceptions for these preconditions so the training invocation cannot silently skip them.

No leakage was found in the frozen dataset: example IDs and prompts are unique; template groups are disjoint across train/calibration/test; all label/language groups have the declared split counts; and an independent exact-token scan found no gold label in prompt text. The generator also assigns all variable configurations from one template to the same split.

## Training and evaluation behavior

- FP32 is trained with float32 master weights. PTQ is exported from the selected FP32 checkpoint after quantizing its two matrices. QAT trains the same two-layer architecture with a straight-through ternary weight approximation. Biases and activations remain float32. This is accurately described as a ternary-aware MLP, not a BitNet Transformer.
- Checkpoint selection uses calibration NLL. Temperature and confidence threshold are also fitted on calibration predictions from the exported artifact. The test split is read only after those choices and is not used for training or calibration. Reusing calibration for checkpoint selection and calibration-parameter fitting can make calibration metrics optimistic; the independent template-held-out test remains the reported generalization measure.
- The task is intentionally narrow: eight binary IR operation labels over generated English/Korean prompts. The test set has two held-out templates per label/language group, with eight variable-name configurations per template. It does not test unseen operation families or arbitrary source generation.
- Python and Go input guards match at 1–512 UTF-8 bytes. The frozen prompt set has no empty inputs and a maximum length of 155 bytes. The final saved Go audit is `PASS`: it independently compares 32 rows for each of FP32, PTQ, and QAT, with 32/32 selected labels per variant and feature, logit, and probability differences within the recorded `0.0001` tolerance. The audit binds the frozen dataset, Python parity rows, and all six model files by SHA-256. On the separate 256-row test split, Go reports FP32 256/256, PTQ 245/256, and QAT 242/256. This verifies the saved inference artifacts against the finite parity set and test split; it does not establish behavior outside that evaluation.

## Quantization and reported sizes

Five ternary values are represented as one base-3 byte, so the matrix code storage is 1.6 physical bits per weight (with partial-byte padding); `log2(3) ≈ 1.585` is only the theoretical alphabet entropy. Biases remain float32, and per-matrix scales and metadata are stored separately. The README and training report distinguish these values and also separate packed file size from decoded resident memory. Keep that distinction in the model card and HF description.

The training timer named `gpu_training_wall_seconds` includes each epoch's calibration forward pass, transfer to CPU, and calibration NLL scoring, in addition to optimizer work. `process_peak_rss_bytes` is a process-lifetime high-water mark and is shared/cumulative across the sequential FP32 and QAT runs. MPS allocation fields are sampled once per epoch, so they are sampled maxima rather than true peaks. These can be reported, but should not be described as isolated per-variant GPU-only time or an exact memory peak.

Root's v1 run record reports test results of FP32 256/256, PTQ 245/256, and QAT 242/256. The measured 2.62 s FP32 and 1.02 s QAT durations include the calibration and CPU work above. Sampled MPS allocation was about 2 MB, sampled driver allocation about 53 MB, and process RSS about 442–445 MB. These are pilot observations, not isolated kernel throughput or exact peak allocations.

## Publication and safety

The training pre-execution record contains source and dataset hashes, versions, seed, and budgets; the generator's frozen manifest states synthetic provenance. The final Go audit binds its exact inference evidence to the dataset, parity rows, and model artifacts. These saved records do not by themselves prove data provenance, license rights, model quality beyond the finite evaluation, or privacy of learned weights. A deny-by-default verifier is provided in `tools/verify-public-export/`; it pins the reviewed v1 inputs, recomputes the dataset split counts and hashes, checks the pre-execution/report/card bindings, verifies all three model files through the Go loader, and requires the final Go audit to bind every included inference artifact. It also scans common credential, local-path, and identifier patterns. The full public bundle has not yet been run through this verifier: its filtered summary, model card, README, and reviewer-approved external allowlist must be present first.

The public description should state that weights are independently initialized and not fine-tuned from Laya. It should not describe this pilot as a general natural-language compiler, as a 1.585-bit model, or as proof of correctness.

## Independent data checks

The frozen `data/synthetic-ops-v1/dataset.jsonl` SHA-256 is `af0a637320a8256cd6ebcdb3486a6885f2766441c90f0cb154c11599f3c8ca3d`. Manifest, split, contract, and generator-source digests matched exactly. Independent counts were 2,048 rows, 2,048 unique IDs, 2,048 unique prompts, 1,536/256/256 rows by split, 256 rows per label, and 1,024 rows per language. Cross-split text and template intersections were zero. The longest prompt is 155 UTF-8 bytes. The manifest asserts no sensitive data; the reviewed source generates only hand-authored templates plus generated variable names, with opaque hash-derived row/template IDs.
