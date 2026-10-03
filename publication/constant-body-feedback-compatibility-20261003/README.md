# Same feedback examples, revised body compiler source

The constant-input fix changes the body compiler's source bytes. Research CI
[37091072181](https://github.com/kimjooyoon/gooo-neural-decision-experiments/actions/runs/37091072181)
passed the complete race suite and reproduced the 6,240 feedback rows exactly,
then failed the historical manifest comparison at its body-compiler digest.
That failure is retained; the older numerical-portability failure also remains.

The full local reproduction using the revised source produced an identical
`dataset.jsonl`, SHA256
`570d1d73bfea32662b869e5cfbf77e6d04df838b703191b076be58c1d2180dbf`.
Every manifest field also matches except this one deliberate source change:

| Source | SHA256 |
| --- | --- |
| Original `internal/bodyplan/bodyplan.go` | `6ebd90bde6e97ffc93dede1ae741a262cb56408194ca3696e9ced530dd109a07` |
| Constant-input fix, research `5cda16c` | `562374bc20db45063db1b934564aedfb234c3b819617eb999948ca822385be5e` |

The original [manifest](../../data/feedback-path-v1/manifest.json) and dataset
remain unchanged. This directory adds the [new source manifest](manifest.json).
No data or model weights are duplicated. Regeneration performs zero model
predictions and zero training updates.

CI now compares the regenerated data with the original exact dataset and the
complete new manifest with this versioned file. It additionally checks that
substituting the original body-source digest reproduces every original manifest
field. Thus changes to rows, targets, other source files, counts or scope still
fail. The generated Linux manifest is retained in the
`feedback-curriculum-revised-source` artifact.

Linux [run 37091975068](https://github.com/kimjooyoon/gooo-neural-decision-experiments/actions/runs/37091975068)
at `9786b591f47ab705231e141221a0b97f15323820` passed `runtime-and-artifacts`,
including the exact dataset and new manifest comparisons and the original-field
compatibility check. Downloaded artifact `11262771495` has the same manifest
bytes as this directory. Fifteen jobs passed; the historical
`full-input-initial-replay` numerical comparison remains the workflow's one
failure. The earlier failed run is preserved.

This is a source compatibility observation for the existing authored corpus.
The new constant-body behavior has separate SDK/compiler regressions and an
[own-model native pilot](../constant-recipes-20261003).
