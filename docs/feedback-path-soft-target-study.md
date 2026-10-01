# Finite path sets for continued bilingual Gooo judgment

Status: source and generated curriculum. GPU model training, Go parity and actual
native dogfood must be reported as separate subsequent observations.

The objective is useful partial Gooo construction from Korean/English judgment,
with multiple acceptable finite paths retained. This curriculum evaluates both
typed options against independently authored arithmetic expectations. It keeps
the original natural-language intention label separate from the labels tied for
the best finite behavior. Finite equivalence does not prove linguistic intent.

## Scope and grouping

The source is the existing five-family positioned structural curriculum:
references, assignment targets, operand order, branch layout and statement order.
Its 6,240 rows reuse 2,080 existing original instruction groups. They are feedback
views, not 6,240 new independent experiments. Source configuration, template,
program and intention groups retain their original disjoint train/calibration/
development-test partitions. The reused test split is development evaluation.

- 4,800 training views, 480 calibration views and 960 development-test views.
- Each split has complete, sparse and deliberately inconsistent finite contracts.
- 384/36/76 views respectively have two labels tied for the best finite score.
- All 6,240 generated inputs currently fit 512 bytes. The generator also retains
  longer inputs as explicit `context_declined` records and excludes them from
  training; a separate unit fixture exercises that branch without truncation.
- Inconsistent contracts give the same input two different expected values.
  No candidate can satisfy both; the unmet requirement remains in the denominator.
- Caller CI PASS/FAIL/UNKNOWN is synthetic, unverified context and never authority.

The complete original-label candidate is also checked against the independent
arithmetic oracle. This prevents a broken typed evaluator from silently becoming
the training target. Every row pins its plan, original text and feedback input.

## Planned offline tuning

`training/train_feedback_path_v1.py` fine tunes the existing 12,728-parameter
positioned model on a compiler-known pair of eligible labels. All paths tied for
the maximum finite pass count receive equal target mass. The objective is paired
soft-target NLL; it does not force an arbitrary unique answer for a finite tie.
Calibration selects checkpoints and temperature; the development-test split is
excluded from those decisions. The global confidence threshold remains 1.0 for
experimental bounded TDD use. Source/runtime/orchestration stay Go; Python is
limited to offline MPS training and export.

FP32, PTQ ternary and QAT ternary exports retain the existing Go structural ABI.
Ternary matrices use five base-three digits per byte (1.6 disk bits per weight),
with biases/scales/metadata measured separately. Go loads int8 tensors; packed
storage does not establish packed-compute acceleration or 1.58-bit total RAM.

Required observations before public weight publication: actual optimizer steps
and MPS/resource measurements, Python/Go feature/logit parity, compiler-pair set
selection and finite outcomes against the frozen parent, actual native codegen
with independent generated-Go execution, and an allowlisted public hash check.

```sh
go run ./cmd/feedback-curriculum --out /tmp/gooo-feedback-curriculum
```

The stored dataset and manifest are `data/feedback-path-v1`. Existing main,
source-bound deployment and frozen experiments remain separately recorded.
