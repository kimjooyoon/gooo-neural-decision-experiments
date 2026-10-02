# Where bilingual Gooo judgments lose the intended path

The shared model's first-choice gap has a strong connection to input phrasing.
On the same 512 development views, the frozen shared FP32 model satisfies all
finite expectations in **113 views with the original development prefix** and
**480 views with the bare instruction**. Its weights, original Gooo source,
allowed paths and finite targets stay fixed. This is a controlled input-sensitivity
observation on an already observed cohort.

The input resembles a work order attached to a drawing. Adding “While composing,”
before the work order can cause the current small reader to choose another
assembly. A useful next model should keep reading the complete work order as its
wording changes.

## What was run

Protocol and collector were published before new predictions at research revision
`75da63a0171a60180eacc3d997af4d7e5e8e35ba`. The collection reconstructs the exact
frozen 3,072-view source corpus: eight families, 24 configurations, eight requested
paths, two languages. Program groups remain 1,024 training, 256 calibration and
256 development. All 1,536 bilingual pairs have identical source and full finite
targets. Every view has 16 finite expectations over eight permitted paths.

Five complete input forms were evaluated:

| Form | English example prefix/suffix | Korean example prefix/suffix |
| --- | --- | --- |
| Original | Existing split-specific text | Existing split-specific text |
| Bare | Instruction body | Instruction body |
| Calibration prefix | `Decision request: ` + body | `구성 요청: ` + body |
| Development prefix | `While composing, ` + body | `함수를 구성할 때, ` + body |
| Development suffix | body + ` While composing.` | body + ` 함수를 구성할 때.` |

The original input is retained alongside every constructed form. The experimental
transformation removes only the exact wrapper authored by this frozen curriculum.
It preserves all three source headers and the complete instruction body. All 64
source-feature coordinates in each part are independently checked for equality.
Production Gooo continues to preserve and consume the caller's complete input.

Six existing exports—dense/shared × FP32/PTQ/QAT—made **92,160 actual Go model
predictions** across 15,360 constructed inputs. Shared exports use the compact
representation. The original development totals exactly reproduce the previous
six-model study. This collection performs zero optimizer updates and zero native
program executions. Budget curves evaluate each static ranking against the
existing complete finite targets.

## Development results

First-choice counts have denominator 512; paired-language counts have denominator
256. “Extra” is the sum of ranked attempts after the first candidate until a
finite passing candidate is found.

| Export | Original complete | Bare complete | Suffix complete | Original extra | Bare extra | Original EN/KO disagreements | Bare disagreements | Bare same-wrong pairs |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Dense FP32 | 95 | 239 | 208 | 1,572 | 640 | 256 | 148 | 34 |
| Shared FP32 | 113 | 480 | 233 | 1,469 | 38 | 255 | 32 | 0 |
| Dense PTQ | 74 | 98 | 94 | 1,595 | 1,419 | 45 | 100 | 123 |
| Shared PTQ | 80 | 283 | 247 | 1,439 | 550 | 136 | 140 | 32 |
| Dense QAT | 82 | 218 | 184 | 1,584 | 668 | 240 | 113 | 62 |
| Shared QAT | 89 | 349 | 214 | 1,512 | 281 | 255 | 128 | 14 |

Shared FP32's bare result is 248/256 English and 232/256 Korean views. Both
languages choose a valid path in 224 pairs; all 224 choose the same valid mask.
With the original development prefix, English chooses mask 0 in all 256 views;
Korean chooses mask 7 in 191 views. Under the suffix intervention, 13 pairs choose
different masks that both satisfy the finite contract. This distinction matters
when measuring language agreement: several paths may meet the same expectations.

Bare shared FP32 completion is 1,920/2,048 training views, 480/512 calibration
views and 480/512 development views. Adding the development prefix to the
training views reduces their first-choice completion to 442/2,048. Thus the
format intervention affects source configurations seen during training too.
All five forms, languages, splits, families, mask histograms and budget curves
remain in the full JSON report.

## What the evidence supports

The source/target pairing checks passed. Across the audited forms, no exact
feature-vector bucket had disjoint finite passing sets. This rules out that
specific contradiction within the observed corpus; other representation losses
remain possible.

The v3 encoder hashes 2/3-byte intent fragments into four relative-position
buckets. Prepending a phrase changes both the fragments and their position
buckets, followed by normalization. The experiment exposes sensitivity to that
combined change. Separating position, added words, normalization and finite model
capacity requires further controlled comparisons.

The bare instruction form is also the form used in the original training split.
Its 480/512 result measures behavior after this explicit input intervention.
Published quality under the original inputs remains 113/512. The recorded
regressions, quantized variants and both-wrong language agreement remain visible.

## Cost and reproduction

One complete local collection took **5.59 seconds wall time**, with 5.91 user and
0.51 system CPU seconds, and **177.69 MiB peak process RSS**. These include source
and oracle reconstruction, six model loads, comparisons and 48.9 MB of raw journal
writes. Go runtime threads can accumulate CPU time concurrently. The observation
describes this audit process; prediction-only latency, model-only RAM and the
whole-host CPU increase require separate measurements. GPU work was zero.

The public evidence archive contains the exact dataset, six small models, all
constructed input strings and predictions, the frozen protocol, collector source,
independent reader and process observation. Its manifest binds every member.
The independent reader reconstructs inputs and source features, runs every Go
prediction again, and recomputes all 30 sets of finite and bilingual summaries.
Numerical replay uses a declared absolute tolerance of 0.00001 and exact masks.
Replay predictions are counted separately from the original collection.

From the research repository, verify and extract the published bundle, then run
the independent reader:

```sh
go run ./tools/package-shared-three --mode verify-bundle \
  --output publication/bilingual-wrapper-audit-20261003
unzip -q publication/bilingual-wrapper-audit-20261003/evidence.zip \
  -d /tmp/gooo-wrapper-audit
go run ./tools/verify-wrapper-audit --bundle /tmp/gooo-wrapper-audit \
  --output /tmp/gooo-wrapper-verification.json
```

Both destinations must be fresh. To reproduce collection, check out the frozen
collector revision in a clean checkout and run `tools/audit-bilingual-wrappers`
with `--dataset <bundle>/dataset.jsonl`, `--dense-models <bundle>/models/dense`,
`--shared-models <bundle>/models/shared`, a fresh `--output` directory and the
exact `--source-revision` above. Go 1.27.1 is the recorded toolchain.

## Next model and language work

The next comparison will retain full caller text and source binding while varying
training phrasing and the positional intent representation. It will keep the
2,072-parameter shared judge, matched initialization and bounded GPU work so
changes can be attributed more clearly. FP32 and ternary behavior will be reported
together, with original, varied-format and language-pair metrics kept separate.

Gooo's declared paths and deterministic continuation remain the execution
contract. The language can use these measurements to describe when a suggestion
helps, how much search remains, and where the current reader needs more work.
