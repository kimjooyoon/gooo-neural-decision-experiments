# Complete finite enumeration with two surviving condition paths

2026-10-03, darwin/arm64, Go 1.27.1. Uses the clean `bf51d65a` compiler and the
same frozen own model, source, three-choice recipe and seven native cases as the
[paired condition-chain pilot](../condition-chain-recipes-20261003). The earlier
pilot remains unchanged. This follow-up runs one request per arm.

The original search stops after the first passing candidate. Here the existing
observation API inspects all eight declared combinations before search. The
request permits one round, reuses probe outputs and asks for direct projection
only if a unique candidate remains. No oracle activity is supplied.

| Observation, each arm | Result |
| --- | ---: |
| Declared / observed combinations | 8 / 8 |
| Unobserved combinations | 0 |
| Type-rejected candidates | 0 |
| Candidates rejected by the three selection cases | 6 |
| Surviving masks | 0 and 6 |
| Supplied probe inputs | 7 |
| Inputs separating the surviving pair | 0 |
| Candidate evaluation attempts in observation | 38 |
| Oracle evaluations / model predictions in observation | 0 / 0 |
| Fixed output matrix | 16 KiB |

Both receipts report `NO_DISTINGUISHING_INPUT` and resolution `AMBIGUOUS`.
Generation continues through the existing search. The deterministic arm makes
zero model predictions; the model arm makes one actual joint prediction. Both
emit the same source. Two generations and four compiled runs meet **14/14**
finite expectations, including eight selection-disjoint expectations. Training
updates are zero.

This distinguishes two useful questions: all declared construction choices have
been inspected, and the supplied observations still leave two possible paths.
The seven inputs include both int64 endpoints and the clamp boundaries. Equal
outputs on those inputs are a finite observation. No full-domain equivalence
proof was performed. An unknown expectation was never manufactured, and the
compiler did not label the result as uniquely resolved.

The original pilot's seven unattempted search combinations remain an accurate
record of that earlier procedure. This follow-up's complete enumeration is a
separate observation with an additional 38 candidate evaluations per request.
Process CPU/RSS and paired speed are not collected in this follow-up; the
generation and native receipts retain their own bounded operation timings.

## Reproduce

From this research repository, using the clean compiler at the pinned commit:

```sh
gooo body-codegen --json --activity Clamp \
  --path-plan publication/condition-chain-recipes-20261003/recipe.json \
  --path-observation publication/condition-chain-observation-20261003/observation.json \
  publication/condition-chain-recipes-20261003/source.gooo
```

For the model arm, add `--path-model` with
`publication/full-input-separate-arithmetic-20261003/models/compact/bag-original/fp32/model.json`.
Save the generation JSON, then pass it to `body-execute` with the same source and
recipe plus `publication/condition-chain-recipes-20261003/cases.json` and a Go 1.27.1
`--go-bin`. Exact invocations follow the earlier collector's `observe` function
with the observation option added. All four receipts and the request are here;
`SHA256SUMS` covers each file except itself. Source/model digests are also retained
in the preceding pilot and these source-bound receipts.
