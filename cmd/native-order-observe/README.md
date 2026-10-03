# Frozen order features in native Gooo construction

Pre-collection definition, 2026-10-03. Four authored two-operation families
(add/multiply, subtract/multiply, negate/add, square/add), Korean/English views
and both requested orders make 16 intention/source contracts. These follow the
earlier synthetic feature preflight, now with real compiler-projected context.

Each source contains a mutable local and two assignments. One root-order choice
swaps those assignments; two operand-order choices cover their binary expressions.
Commutative operations deliberately retain semantically equivalent alternatives.
Eight structural masks therefore do not mean eight distinct functions. Report
root-order agreement and native functional behavior separately.

Use the released SDK18 main compiler and three arms: deterministic, frozen compact
positioned-original FP32, frozen compact bag-original FP32. The two trained models
also differ in their learned weights; this is an operational comparison of existing
models, not a causal isolation of feature encoding. No new training, checkpoints
or threshold tuning. Each arm runs with a one-attempt and eight-attempt budget.
The 96 generations make 64 actual joint predictions and 192 compiled executions
if collection completes. A functional failure is retained and is not an execution
infrastructure failure.

Three selection cases use inputs -2, 0 and 3. Eight native cases use -3 through 4;
five are selection-disjoint. The reference is ordinary authored arithmetic for
the requested order. Compiler model context excludes those expected outputs.
There is no task holdout, paraphrase-generalization or training-independence claim.
One sample per cell makes timing descriptive. Arms rotate deterministically and
budgets alternate. Existing caches and first-process outliers remain observable.

Keep source/recipe/cases, all generation/native receipts, complete model input
digests, feature digests, model identities, first masks/probabilities, selected
masks, candidate counts, finite case results, process wall/CPU/RSS and all errors.
CPU is relative to one core; host utilization change is unmeasured. Model-pair
feature equality and prediction equality are computed from real source-derived
inputs. No automatic retraining follows these results.

```sh
go run ./cmd/native-order-observe --compiler /path/to/clean/gooo \
  --compiler-sha 729482ed6ff6c3fd2a5664db415ced48e14e122b \
  --source-revision COMMITTED_COLLECTOR_SHA \
  --go-bin /path/to/go1.27.1 --out /path/to/fresh/evidence
```

The native programs have no external dependencies or effects; each generation
and execution has a 90-second process deadline. Native execution uses the
compiler's existing source-bound bounded runner. Expected raw publication size
is under 16 MiB. A collector/source/model identity error stops with files retained.
