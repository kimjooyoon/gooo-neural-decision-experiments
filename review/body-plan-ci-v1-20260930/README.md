# Body plan evidence review

This review independently binds the frozen 128-plan cohort to the model-free native CI capture and the completed local Laya follow-up. The machine-readable receipt is [audit.json](audit.json); [main.go](main.go) and [laya.go](laya.go) contain the read-only verifier.

## Findings

The CI capture is from source revision `7d626b9ab0b503f5c3d571be09524d689271bd48` and cohort SHA-256 `e517b948a8c1d744cf832b5d711918dd2fef0b90cd951126f579ac54ada427a9`. The checked-in manifest binds 128 plans across eight families. Its eight arms reuse those plans; they are not 1,024 independent intents.

The verifier matched the selected-cell journal and final report for all 1,024 CI cells, checked the model weights against the independent Go audit and source manifests, and confirmed all 36,160 compiled-Go execution markers were unique and matched their frozen case results. Native source digests matched saved compiler replies. The CI capture reports zero unknown cases and no native generation failures. The workflow pins native compiler source revision `60cf7f49b0e302a6bebb42bc8da90f3ed19b2b82`; its transient executable bytes were not present in the downloaded artifact, so this review could confirm the workflow and report bindings but could not rehash those binaries locally.

The separate local Laya report has the same source revision and cohort. The audit checked 1,280 cells and all 45,200 compiled-Go case markers, with zero unknown cases. Its 128 raw exchange files contain exactly 222 completed exchanges, one for each declared hole. Each request matches the frozen hole text and allowed operation descriptions; requests contain no training or held-out cases. Replies, chosen operations, probability vectors, and selection receipts agree. The exchange bundle digest is recorded in `audit.json`. The final report carries runner, native compiler, and Go executable hashes and says their bytes remained stable through execution, but those transient binaries are not included in this local capture for direct rehashing.

The eight arms shared with CI have matching choices, scores, and emitted source digests. Probability values differ by at most `9.54e-7` across the two platform runs, within the audit's `1e-6` comparison tolerance; timing fields are excluded from this comparison.

| First-choice arm | Held-out correct |
| --- | ---: |
| Deterministic fallback | 384 / 1,280 (30.00%) |
| FP32 | 950 / 1,280 (74.22%) |
| PTQ ternary | 679 / 1,280 (53.05%) |
| QAT ternary | 916 / 1,280 (71.56%) |
| Laya multilingual | 945 / 1,280 (73.83%) |

Laya selected the authored gold operation for 144 of 222 holes. Its training-only finite search used 404 candidate evaluations, compared with 717 for the deterministic search control, a reduction of 43.65%. Both searches use the same bounded candidate enumeration, and all 1,280 Laya-search held-out cases passed. That final score is attainable without a model and does not measure a learned repair policy. Laya receives operation descriptions alongside each hole; the tiny models see hole text and classify among global labels. The first-choice rows compare these protocols on these plans, not model quality in isolation.

The raw request latency distribution contains 222 observations. Its ordinary median (average of the two middle values) is 40.911 ms; the upper middle observation is 40.914 ms. The maximum is 184.306 ms. For p95, `floor(0.95*n)` as a one-based rank gives 46.825 ms; nearest-rank p95 gives 48.029 ms. Both definitions are retained in the receipt because p95 depends on the order-statistic convention.

The process monitor records 761 samples over roughly 190 seconds and one end marker. Its process CPU counter rose 12.12 seconds; sampled RSS ranged from 453,520 KiB to 701,520 KiB, and the maximum rolling one-core CPU reading was 89.7%. The monitoring interval includes idle time before and after the 47.123-second report wall window. These are process observations over that larger interval, not per-request CPU, peak live request memory, or whole-host utilization.

## Audit limits

The Laya run's preserved pre-execution snapshot says `running` with zero completed cells and zero requests, as expected when it was first written. Its progress snapshot also says `running` even though it records 1,280 selected cells and 222 requests; neither snapshot is treated as the final result. The completed report and final selected-cell journal bind the completed run.

The native CI experiment ran at its pinned source revision. The frozen `internal/bodyplan/bodyplan.go` SHA-256 is `286fc65f177bc77a02b54889ef2f4928b60590b802d003b4c1c145a0b02583ae`; the current review worktree file is `2f1a00967786ceea9679ca3a2abf9f1bb8dd4af8c951abc1df1430149b21ea9b`. The verifier records both, plus the exact review-helper and imported-package source digests, and verifies those inputs did not change during the audit. It checked the saved CI and local execution markers against the current evaluator during review; this post-run review does not replace the frozen CI execution evidence.

This is a finite synthetic suite with authored expected-value formulas and plans that share semantics. It does not establish whole-domain or full-language correctness. The review helper checked saved evidence and loaded model metadata/weights only; it made no model prediction calls and did not run the native compiler or generated Go programs.
