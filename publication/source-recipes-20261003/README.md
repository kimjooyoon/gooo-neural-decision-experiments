# Source-derived recipes: paired native observation

2026-10-03. Compiler [`3a52232d`](https://github.com/kimjooyoon/meta-ontology-go/commit/3a52232de15c3c6ccaf6489d4d8da3a42913591f),
SDK v0.2.17, Go 1.27.1, darwin/arm64. The clean compiler build reports
`vcs.modified=false`; the manifest binds its executable and model hashes.
Implementation [PR #1168](https://github.com/kimjooyoon/meta-ontology-go/pull/1168)
is awaiting its own CI and promotion at publication time.

## What changed

A Gooo body supplies its typed base. A small recipe identifies operand, branch,
local-reference, assignment-target or adjacent-statement choices. This pilot uses
three operand-order choices, one bilingual intention view, an unchanged own
model and three repeats per condition. It compares the recipe with an equivalent
full document under identical source, model and tests.

- 24 generations and 48 actual compiled runs; 12 paired expanded documents and
  emitted sources match exactly.
- Compact JSON: **1,538 → 569 bytes**, a **63.0%** reduction. Both inputs are
  compacted with the same JSON rule, preserving the same bilingual intentions.
- Six actual joint-model predictions; each ranks all three choices in one call.
  The other 18 requests make no prediction. Training updates: zero.
- Independent runtime expectations: **84/144** across all conditions. Source
  oracle/direct resolution: **72/72**. Search with only the initial example:
  **12/72**, equally for recipe and full document, with and without the model.

The initial case is `10 -> -3`. Several combinations satisfy it. A passing initial
test therefore gives limited information about `7-input` on other inputs. The
declared reference activity adds a discriminating observation; unique-candidate
resolution then constructs the body directly. These are finite results on one
authored task, with repeated requests rather than independent task samples.

## Whole generation process, three samples per cell

| Input and mode | Median ms | Median peak RSS MiB | Process CPU, one core % | Runtime passed/total |
| --- | ---: | ---: | ---: | ---: |
| Full, disconnected search | 7.910 | 16.797 | 88.10 | 3/18 |
| Recipe, disconnected search | 7.903 | 16.953 | 88.24 | 3/18 |
| Full, own-model search | 8.143 | 17.328 | 86.78 | 3/18 |
| Recipe, own-model search | 8.598 | 17.641 | 86.53 | 3/18 |
| Full, disconnected resolution | 8.232 | 17.391 | 86.41 | 18/18 |
| Recipe, disconnected resolution | 8.949 | 17.609 | 87.38 | 18/18 |
| Full, model requested + resolution | 8.139 | 17.250 | 87.89 | 18/18 |
| Recipe, model requested + resolution | 8.691 | 17.578 | 88.16 | 18/18 |

Resolution skips model loading and prediction, including when a model is
requested. Recipe expansion is included in process time and adds source
projection/validation work. The own-model search medians differ by 0.455 ms;
this pilot demonstrates shorter authored input with the same construction,
while showing the extra processing cost. It supplies no causal speed claim.

CPU is child-process user+system time divided by wall time. It can describe the
process's use of one core, and does not measure whole-computer utilization or
utilization increase over idle. Peak RSS covers the whole generation process,
not only weights or fixed arrays. Compilation/runtime process measurements are
also retained in `processes.json`; the table concerns generation.

## Reproduce and inspect

Collector source: [`1486ef8`](https://github.com/kimjooyoon/gooo-neural-decision-experiments/commit/1486ef8).
Use a clean build of the pinned compiler and its source checkout:

```sh
go run ./cmd/source-recipe-observe \
  --compiler /path/to/gooo --compiler-sha 3a52232de15c3c6ccaf6489d4d8da3a42913591f \
  --compiler-source /path/to/meta-ontology-go --go-bin /path/to/go1.27.1 \
  --out /path/to/new-observation
go run ./cmd/source-recipe-read --dir publication/source-recipes-20261003
```

The independent reader checks generation/compiled-run counts, paired source and
document hashes, actual prediction counts, per-case outcomes, embedded-parent
privacy and process accounting. It calls no model, compiler or network provider.
`independent-reading.json` retains its output. Raw generations, runtime receipts,
source, both plans, observation request and six runtime cases are included.

The first collector wrongly expected three predictions because it counted three
choices. The pinned joint head correctly made one prediction, and that assertion
stopped collection after five completed generations. The original collector
commit and captures remain in
[`source-recipes-initial-accounting-20261003`](../source-recipes-initial-accounting-20261003).
Those captures are excluded from this 24-generation result. The correction
changed accounting only; source, compiler, model weights and intentions stayed
the same.
