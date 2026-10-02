# Full-input judgments: numerical portability follow-up

Status: the original platform difference is retained below. The subsequent
[explicit-arithmetic comparison](full-input-separate-arithmetic-results-20261003.md)
passes all 18,432 paired rows, including complete rankings and intermediate
arrays. SDK/native adoption remains next. Optimizer updates for both follow-ups: zero.

The original full-input model comparison is retained in
[the initial report](full-input-judgment-initial-results-20261003.md). Its Linux
replay passed the archive, source-target, training-journal, export-tolerance and
expanded/compact checks. Comparing the complete development summaries then found
different partial-completion curves. The exact comparator remains a failure.

## What the retained observations establish

The diagnostic Go reader pairs every input by source, full text, input hash,
finite target, model and condition. It makes zero new predictions.

| Paired development observations | Count |
| --- | ---: |
| Complete arm64/Linux row pairs | 18,432 |
| Different logits | 16,236 |
| Different probabilities | 15,524 |
| Different first selected mask | 0 |
| Different complete ranked order | 272 |

Maximum absolute logit difference is `1.430511474609375e-6`; maximum probability
difference is `4.470348358154297e-7`. The 101 differing summary leaves concern
partial-completion curves, with family differences in `chained_operands`.
First-choice completion, extra attempts to a fully passing path, and the reported
bilingual outcomes agree. The full changed-row list keeps each input hash and
both complete candidate orders. The small numerical deltas are measured facts;
the exact arithmetic operation responsible still needs a controlled check.

[The complete retained diagnosis bundle](../publication/full-input-platform-diagnosis-20261003/README.md)
includes every Linux observation, the 101 summary differences, all 272 changed
row identities and the closed archive inventory.

The first failed job skipped artifact upload. The next source revision added
complete mismatch diagnostics and unconditional artifact retention. That second
run preserved the actual Linux rows used here; training was not repeated.

- Initial local auditor: `7d21fa426f95b5a769375b74ab350dc6456b236d`.
- Retained Linux source: `8ea6a0f1e403a58d3eed46e7fe480ad317ba1758`.
- [First comparison failure](https://github.com/kimjooyoon/gooo-neural-decision-experiments/actions/runs/37058751906/job/111009821854).
- [Retained Linux replay](https://github.com/kimjooyoon/gooo-neural-decision-experiments/actions/runs/37059199749/job/111011318694).
- [Anonymous Hugging Face byte verification](../publication/full-input-initial-hf-verification-20261003.json)
  independently verified all 59 public files including the model card.

## Repair plan recorded before the subsequent collection

1. Inspect multiplication/accumulation and scaled-bias operations for compiler
   fusion differences. Use explicit float32 rounding in a separately identified
   arithmetic contract, so the existing observed arithmetic remains replayable.
2. Keep model weights, source features, full inputs, candidate masks, finite
   expectations and training checkpoints fixed. Preserve arithmetic identity in
   expanded and compact metadata, loaders, conversions and inference receipts.
3. Pair every new arithmetic result with the retained legacy result. Record
   hidden/logit/probability values and all candidate orders, with model-call
   counts separate from the zero-update training count.
4. Run arm64 and Linux checks against the same new arithmetic contract. Require
   exact discrete outcomes and complete ranking/partial-completion curves;
   keep numerical differences and observed limits explicit. Do not claim
   portability from the first-mask agreement alone.
5. Carry the observed order-alias counterexample and the remaining full-input
   protocol forward. Publish actual native compiler behavior before adoption.

This follow-up does not change the original cohort, model weights or published
initial observations. Its purpose is to make the execution contract precise
enough that tiny numerical differences do not silently change the work schedule.
