# Full-input Gooo judgments: first training and Go observations

Updated 2026-10-03. This is the first result of the
[published representation × phrasing protocol](https://github.com/kimjooyoon/gooo-neural-decision-experiments/blob/main/docs/full-input-judgment-preregistration-20261003.md).
The four fresh 2,072-parameter judges completed 6,400 local MPS updates. All
twelve FP32/PTQ/QAT exports have actual Go numerical and compact-storage checks.
The initial quality comparison covers the original 512 development views and
two authored, evaluation-only wording variations of those same source tasks.

## What changed

The earlier model was sensitive to introductory wording. Here the full original
instruction stays in the input. We compare two representations: the existing
positioned byte fragments (V3) and whole-text fragment counts (V4). Each is
trained either on original source/feedback states or on five neutral phrasing
forms of each state. This separates the phrasing treatment from the
representation treatment; V4 changes both position encoding and hash collisions.

Think of the model as a small assembly assistant reading a work order. The old
assistant sometimes chose different parts when an introductory phrase moved the
same instruction down the page. Counting fragments across the whole page helps
on this cohort, while losing information about which operation comes first.
Both effects matter when deciding how to develop the language and model together.

## Original complete development inputs

Each of the 512 views has 16 authored finite expectations. “Complete” means the
first selected path meets all 16. “Extra attempts” sums the additional candidates
needed when eight paths are tried in the model's fixed probability order.
Pair measures use 256 Korean/English views of the same source tasks.

| Model | First path complete /512 | Extra attempts | Both languages valid /256 | Same path, both wrong /256 |
| --- | ---: | ---: | ---: | ---: |
| positioned-original FP32 | 113 | 1,469 | 1 | 0 |
| positioned-original PTQ | 80 | 1,439 | 22 | 98 |
| positioned-original QAT | 89 | 1,512 | 0 | 1 |
| positioned-varied FP32 | 266 | 528 | 75 | 9 |
| positioned-varied PTQ | 104 | 1,194 | 34 | 142 |
| positioned-varied QAT | 238 | 617 | 92 | 47 |
| bag-original FP32 | 368 | 186 | 180 | 68 |
| bag-original PTQ | 94 | 1,128 | 37 | 167 |
| bag-original QAT | 287 | 371 | 101 | 43 |
| bag-varied FP32 | 320 | 326 | 132 | 48 |
| bag-varied PTQ | 111 | 1,057 | 34 | 122 |
| bag-varied QAT | 257 | 569 | 74 | 25 |

The fresh positioned-original control reproduces the earlier 113/512 and 1,469
extra attempts. Bag-original FP32 reaches 368/512 (71.9%) and 186 extra attempts
on these complete inputs. Its 68 same-wrong pairs show why agreement alone is
insufficient. The common language families, authored alternatives and finite
expected values are unchanged; the development tasks have been observed before.

PTQ weakens all four FP32 models in this comparison. QAT recovers part of that
loss, with different effects across representations and phrasing treatments.
All variants remain in the public record. The compact artifacts contain 8,288
weight bytes for FP32 or 446 for ternary variants, using the same fixed tensor
shapes as the previous shared judge.

## Two additional complete-input conditions

These additions were fixed before training and excluded from the optimizer and
calibration. The prefix adds `For this task, ` / `이 작업에서는, ` around the
entire original instruction. The suffix adds ` That is the complete instruction.`
/ ` 이것이 전체 지시입니다.`. Complete original text and feedback are retained.

| FP32 model | Original complete /512 | New prefix /512 | New suffix /512 |
| --- | ---: | ---: | ---: |
| positioned-original | 113 | 125 | 243 |
| positioned-varied | 266 | 110 | 233 |
| bag-original | 368 | 358 | 368 |
| bag-varied | 320 | 269 | 173 |

More training forms did not consistently improve these conditions. For example,
bag-varied has the lowest original calibration passing-set NLL among the FP32
arms, yet falls to 173/512 on the new suffix. Positioned-varied agrees across all
256 bilingual pairs under the new prefix, while 201 pairs choose the same wrong
path. The full report preserves every variant, family and language breakdown.

## A representation limit kept in the evidence

The two work orders below contain identical 2/3-byte fragment multisets:

```text
Step: add one. Step: multiply by two. Step: end.
Step: multiply by two. Step: add one. Step: end.
```

With the same valid source header, V4 maps them to exactly equal features and V3
distinguishes them. Their reference arithmetic differs: for inputs `[-2, 0, 3]`,
`(x+1)*2` gives `[-2, 2, 8]`, while `x*2+1` gives `[-3, 1, 7]`.
This feature probe records zero trained predictions and zero native executions.
It identifies an order distinction that this V4 representation cannot learn from
intent alone. Future language/model work must preserve such distinctions through
the source representation or a richer intent representation.

## Preparation, optimization and actual Go calls

- Go reconstructed 10,739 original source/teacher states into **49,599 full forms**.
  Every form fit the declared bound; original state weights were distributed
  across five forms for varied training and restored for original training.
- Independent Go preparation replay recomputed **76,184,064 float32 feature
  values**, source identities, targets, weights and all full texts exactly.
- Four FP32 and four QAT stages each made **800 actual updates**, with a journal
  entry for every update. Calibration retained the same original 512 views.
- The initial Go audit reconciled all **6,400 update records** and every epoch's
  temperature/checkpoint selector. It made **624 expanded export-parity calls**,
  **624 compact parity calls**, **6,144 calibration calls** and **18,432 development
  calls**: **25,824 actual Go predictions** in total.
- Expanded and compact complete workspaces/predictions match exactly on the
  parity vectors. The largest export-to-Go absolute error was
  **1.8477439880371094e-6**, below the fixed `1e-5` tolerance. Go replay also
  recomputed exported calibration NLL and the PTQ temperature choice.

No new Gooo native generation is counted in this initial audit. Existing native
execution evidence belongs to the earlier model editions. V4 loading is present
in the research runtime; publishing the SDK extraction and measuring actual
compiler generation/execution are subsequent protocol steps.

## Resource observations

The Go launcher started one local MPS optimizer process and waited for its exit:

| Measurement | Observation | Scope |
| --- | ---: | --- |
| Wall time | 42.180 seconds | Input validation, all four arms and exports |
| Process CPU time | 21.016 seconds | Same optimizer process |
| Average CPU / one core | 49.825% | CPU time ÷ wall time |
| Process maximum RSS | 1,809,678,336 bytes | About 1.69 GiB, optimizer process |
| Maximum sampled MPS allocated bytes | 154,424,832 | Largest recorded training-stage sample |
| Maximum sampled MPS driver bytes | 1,093,353,472 | Largest recorded training-stage sample |
| Automatic retries | 0 | One completed optimizer invocation |

The CPU percentage describes the optimizer process. Whole-machine CPU change and
GPU utilization percentage were not sampled. MPS allocation samples describe
training memory; inference and complete code-generation timings require their
own measurements. The existing model/runtime results remain dated separately.

## Evidence and reproduction

Public bundle:
[GitHub](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/full-input-initial-study-20261003)
and [Hugging Face research appendix](https://huggingface.co/asketeddy/gooo-shared-judgment-tiny-v1/tree/main/research/full-input-initial-20261003).

- `bank.zip`: full preparation, both raw feature arrays, full text rows,
  rejections, original initializer, independent replay and storage preflight.
- `training.zip`: all 800 epoch records, 6,400 update records, twelve expanded
  exports, numerical vectors, stage reports and process measurements.
- `initial-audit.zip`: twelve compact exports, all 18,432 development observations
  in lossless journals, complete input texts and the initial comparison report.
- `sources.zip`: exact authored Gooo corpus, frozen teacher states/report/audit,
  preregistered protocol and six offline optimizer source files.
- `manifest.json`: every public file and decoded archive member with SHA256 and
  byte extent. The Go packager reopens each archive and checks every digest/CRC.

Training/preparation source:
[`d23bd7e`](https://github.com/kimjooyoon/gooo-neural-decision-experiments/commit/d23bd7e9e9fa3f197ffb5ee8d10872ad982e29ca).
Initial Go auditor:
[`7d21fa4`](https://github.com/kimjooyoon/gooo-neural-decision-experiments/commit/7d21fa426f95b5a769375b74ab350dc6456b236d).
The protocol includes preparation and training commands. The Go audit is:

```sh
go run ./tools/audit-own-three-models --full-input-experiment \
  --models runs/own-three-full-input-training-20261003 \
  --dataset runs/own-three-composition-curriculum-fixed-20261002/dataset.jsonl \
  --teacher-curriculum runs/own-three-teacher-curriculum-20261002 \
  --output runs/own-three-full-input-initial-audit-20261003 \
  --source-revision "$SOURCE"
```

The audit accepts a fresh output directory. Its required source/teacher digests
and the training source are fixed in Go. Model loading uses explicit V3/V4
metadata and retains all caller input.

## Remaining work in the same study

The [subsequent Linux replay](full-input-numerical-portability-followup-20261003.md)
has completed its collection and exposed 272 complete candidate-order differences.
Its exact comparison remains failed; the report retains every observation.
The [versioned arithmetic follow-up](full-input-separate-arithmetic-results-20261003.md)
now reproduces every hidden/logit/probability vector and complete ranking across
both platforms on these 18,432 rows. Next, complete all split/form comparisons,
exact feature-conflict inventories, timing and allocation measurements,
and source-bound native Gooo generation followed immediately by compiled execution. Preserve
the complete input, deterministic zero-prediction continuation and unresolved
completeness frontier. Model adoption will use these additional observations.
