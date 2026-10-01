# Native main bounded construction budget study

Freeze this plan and the Go runner before the first measured model/native call.
First-shot perfection is not acceptance. Study partial construction, iterative
progress, bilingual agreement and complete cost accounting without new training.

## Fixed inputs and arms

- Native main: `37fb287a9c888bd0262194f524dde16426eca3a9`, clean Go 1.27.1 build,
  SDK `v0.2.7-experimental`, retained NDJSON worker with one worker.
- Existing `studies/compound-path-v1/cohort.jsonl`, digest
  `aefd16b22076c05b113e29df3e7788b06e60e0c5c14e45cc75547bd1e582a15a`.
  72 existing views = three templates, four intention masks, two languages and
  complete/sparse/contradictory contracts. Twelve existing intention groups;
  this is not a new language holdout or 72 new independent experiments.
- Disconnected deterministic arm, plus frozen parent FP32 and existing feedback
  FP32/PTQ/QAT each with initial ranking alone or varying-coordinate feedback.
  These are nine arms. Model identities are checked against existing fixed pins.
- Candidate budgets 1, 2 and 4, one new candidate per step. Feedback arms allow
  three feedback rounds but only reconsider between committed attempts before
  the budget is spent. A one-candidate budget cannot make a feedback prediction.
- UNKNOWN caller CI hint bound to the exact main SHA. This is unauthenticated
  context, never authorization; preserve it even if source CI later passes.

## Execution and accounting

Budgets run 1/2/4 for even arms and 4/2/1 for odd arms. Existing row order is
forward for even arms and reversed for odd arms. Each budget has six processes
with twelve valid sequential requests plus an initial invalid-source request.
Expected totals: **1,944 valid native constructions, 162 zero-prediction source
rejections and 162 native processes**. Raw NDJSON and whole-process resource
sidecars are saved before inspection. Individual output remains bounded to the
existing 2 MiB controller cap. No parallel workers or Python runtime are used.

Only source, typed plan and declared finite cases reach the native worker.
Separate-input cases, authored intention masks and legal-maximum metadata stay
in the evaluator. Native receipts must bind source/document/cases/model, preserve
type/replay and writes-zero checks, and reconcile emitted function AST to the
selected typed body. Independently authored integer-state oracles check every
candidate and native finite actual, including failures. Every distinct selected
Go source is independently compiled and run over the union of finite/separate
inputs, including int64 boundaries, then checked for each observation's mask.

Same-arm candidate/committed-curve prefixes must match across budgets. Initial
and feedback arms must agree at budget one; feedback cannot happen before the
first committed attempt. Feedback receipts preserve first failure, CI hint and
constant-coordinate skip evidence. The original full-budget audit keeps its
maximum-required check; the new partial audit verifies best-so-far completion.

## Measures and limits

Report finite cases passed, restricted finite legal-space attainment, evaluator
separate-input outcomes, intention-mask agreement, bilingual agreement, candidate
attempts and actual initial/feedback/skip counts. Independently enumerate the
four-path maximum after construction; partial native search need not reach it.
No all-input or arbitrary-language completeness claim follows from this maximum.

Committed curves contain only actual attempts. For a fair budget-position mean,
carry the terminal value forward after early complete termination; label this
padding explicitly and retain the actual attempt count. It is neither added
execution nor wall-time area. Compare feedback against initial ranking at the
same model, cases and candidate budget. First-shot metrics are observations.

Request round-trip excludes startup/constructor work. Whole-child wall/CPU/RSS
includes setup, one source rejection and twelve valid requests. Record p50/p95
request/native time, all raw outliers, CPU per valid request, one-core CPU ratio
and peak RSS. This does not measure controller resources, host CPU increase,
maximum-size inputs or long-stream memory. Fixed order does not establish causal
speedup. No new training, GPU work, upstream Laya call or checkpoint promotion.

Publish all partial and negative results on GitHub and a bounded public HF
appendix; preserve earlier weights, card, preregistrations and raw evidence.
