# Order-feature isolation: Linux replay receipt

Observed on 2026-10-03. Collector source:
[`75d3c30e7598a266a9dbed433759f497eebb7687`](https://github.com/kimjooyoon/gooo-neural-decision-experiments/commit/75d3c30e7598a266a9dbed433759f497eebb7687).
The implementation move is commit `684620c2527e3b783f2c918121f8a5ab2bd8990b`.

The [Linux CI run](https://github.com/kimjooyoon/gooo-neural-decision-experiments/actions/runs/37087469678)
passed `separate-arithmetic-replay` and `path-observation-evidence`.
Downloaded immutable artifacts:

| Artifact | ID | Compressed bytes |
| --- | ---: | ---: |
| separate-arithmetic-linux-replay | 11260897452 | 7864290 |
| path-observation-independent-readings | 11261311849 | 16840 |

The arithmetic collection made 73,728 real model predictions over 18,432 inputs
using the four existing arithmetic/layout lanes. The unchanged comparator
accepted the original computation inventory and observed zero changed explicit
hidden values, logits, probabilities, first masks, full rankings or finite
outcomes against the published arm64 collection. Legacy arithmetic retains
272 changed full rankings and 38 changed finite outcomes.

`platform-comparison.json` is copied byte-for-byte from the downloaded artifact.
Its SHA-256 is
`6dfb76e59d52679638eebbfd88dfc86703fe30163ae18f3e7453be6b33cfc25a`.
Comparison adds zero model calls. The complete collection stays in the CI
artifact; this publication adds only its compact comparison.

The order replay made 64 real frozen-model predictions over 32 authored pairs.
Every published counter, input digest, reference body/output and prediction
matched locally after deleting only each view's `prediction_ns`. It preserves
the original scope: synthetic structural headers, four arithmetic families,
two language views and four wrappers. New-feature training and native execution
remain zero. See the [original preflight](../intent-order-preflight-20261003).

Both replay jobs use zero optimizer updates. The historical
`full-input-initial-replay` job still reports the original legacy platform
difference. See the [failure cause and source isolation](../../docs/intent-order-isolation-20261003.md).
