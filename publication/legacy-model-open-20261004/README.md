# Legacy model files — 2026-10-04

## What a user can encounter

An older model loader checks a regular file, then opens its pathname again.
If the pathname becomes a FIFO between those operations, Unix open can wait
before descriptor validation. This matters when another local process replaces
a model bundle during a Gooo request. The current installed compiler
`d1bfd273ab4e21d0191548b066a27bcb77d7ed86` uses SDK v0.2.20; its common initial
metadata dispatch and whole-candidate judge opener are already nonblocking.
The later shared-model SDK read still carries this gap.

The actual CLI probe alternates hardlinks to an owned regular artifact and an
owned FIFO. The metadata arm establishes a wait on attempt 40: the sole writer
opens after a deliberate 500ms hold, then the joined CLI rejects the descriptor
at 505.242167ms. Exit is 1, stdout is empty and no result directory is created.
The elapsed interval includes the deliberate 500ms hold.
The weights arm has zero confirmed waits in 64 attempts. That finite observation
does not establish that its same open primitive cannot wait.

Across the 104 attempts, 36 ordinary completions make 36 actual shared-model
predictions and 72 native runs, matching 4,608/4,608 supplied expectations.
A separate ordinary control makes one prediction and two native runs with
128/128 expectations. Its first construction-plus-execution response is
1,245.668875ms; the model prediction receipt records 40,333ns. These are distinct
observation intervals from one request. Host CPU and model-only RAM are unobserved.
The repeated authored task and unchanged weights remain explicit.

## Correction and status

Research source [`ddcd1002`](https://github.com/kimjooyoon/gooo-neural-decision-experiments/commit/ddcd1002b14996f4e6b5577a227e065441a407c5)
adds one bounded reader used by operation/path and joint/shared model loaders.
On Unix it uses nonblocking, no-follow open, checks the opened regular inode
against the earlier Lstat identity, checks extent and reads at most the declared
bound plus one byte. Schema, digest, layout, finite-value and symlink checks remain.
Prediction math, fixed arrays, labels and weights retain their contracts.

The prior blocking primitive fails the controlled replacement-FIFO and symlink
regressions. The corrected helper and the existing decision/joint packages pass
their race tests. Windows arm64 has compilation evidence; runtime nonblocking
behavior there has not been observed. [SDK PR6](https://github.com/kimjooyoon/gooo-decision-runtime/pull/6)
passed exact-head CI37166324852 and merged as `061cd2d8`. The tested virtual-merge
tree equals the submitted candidate tree. Its complete arithmetic replay matches
18,432 frozen input rows using 36,864 actual predictions, with zero optimizer
updates. This metric covers agreement with the frozen arithmetic observations.
[SDK v0.2.21-experimental](https://github.com/kimjooyoon/gooo-decision-runtime/releases/tag/v0.2.21-experimental)
extracts 69 exact source/test files, retaining the previous provenance inventory.
Compiler adoption and a freshly installed comparison are follow-up stages.

The first hardlink controller failed because POSIX rename keeps a staging link
when both names already refer to the same inode. Its original source and error
are preserved. The revised controller joins both child and swapper before
removing its owned temporary directory. No timeout is used as proof of the gap.

## Read the saved evidence

From this repository's root with Go1.27.1:

```sh
shasum -a 256 -c publication/legacy-model-open-20261004/SHA256SUMS
mkdir /tmp/gooo-legacy-saved
unzip -q publication/legacy-model-open-20261004/installed-observations.zip -d /tmp/gooo-legacy-saved
go run ./publication/legacy-model-open-20261004/archive-readback publication/legacy-model-open-20261004/installed-observations.zip /tmp/gooo-legacy-saved
go run ./publication/legacy-model-open-20261004/readback /tmp/gooo-legacy-saved
```

Use a fresh extraction directory. These readers check all archived members and
recalculate saved process, model-call and native-case counts with int64 values.
They do zero inference, compilation, execution and training themselves.
