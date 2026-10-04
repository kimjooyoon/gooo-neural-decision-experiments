# Arithmetic after a bounded model-loader change — 2026-10-04

Research CI37166559740 stopped with `pinned file differs: model_test.go` before
comparing its calculated results. The default strict source check still rejects
that difference; its original CI log and a fresh default-mode error are retained.
[Versioned source-delta comparison](../../docs/arithmetic-source-delta-20261004.md)
checks each recorded source inventory against its own immutable Git revision.

Clean research source `695f5bc85e57938e1a7a86feeb11bf63d6a14e7a` made **73,728
actual model predictions over 18,432 paired input rows**, with zero optimizer
updates. The local arm64 collection took 6.025306s and kept 10,616,382 output
bytes before its report. It uses the original frozen text, finite targets,
weights and four legacy/explicit-arithmetic lanes.

Compared with original arm64 source `250c935642b59f4f3122360c90286e5557af3da0`,
all 18,432 explicit feature/hidden/logit/probability observations, rankings and
finite completion curves match. Legacy observations also retain zero differences
in this same-platform comparison. The v2 report records
`computational_sources_equal: false` and fourteen source changes, including the
new imported file-reader dependency and the versioned comparator itself.
This is an arithmetic replay of existing data. Native loader behavior and
generated-program behavior have separate observations.

Focused race tests verify historical Git source binding, invalid/missing source
and complete added/removed/changed inventories. The Linux CI requests the new
mode and fetches Git history.

The `separate-arithmetic-replay` job of source-bound CI37168510833 finished
SUCCESS. Artifact11289922451 was downloaded and its archive SHA256
`00d944c221608fc53e748ed8c798fcf9bf9c342864f3df73e4f66b44d95102b6`
matched GitHub's metadata. Independent local consumption of every Linux row
also passed. All 18,432 explicit observations retain zero value, ranking or
finite-outcome differences. Legacy arithmetic keeps its original **272 ranking
differences and 38 finite-outcome differences**, with unchanged first masks.
The whole source-bound workflow subsequently finished SUCCESS. This arithmetic
archive independently covers its completed arithmetic job and source `695f5bc8`.

`linux.zip` preserves the complete retrieved Linux collection, its original
CI comparison and model files. `linux-independent-comparison.json` records the
independent consumption. Collection counts are 73,728 actual predictions per
platform; comparison makes zero predictions. Weights and optimizer state remain
fixed throughout.

`arm64.zip` contains the exact complete new collection. The member reader checks
every file's bytes and closes the archive inventory; comparison does no new
inference or native execution.

```sh
shasum -a 256 -c publication/arithmetic-source-delta-20261004/SHA256SUMS
mkdir /tmp/gooo-arithmetic-current
unzip -q publication/arithmetic-source-delta-20261004/arm64.zip -d /tmp/gooo-arithmetic-current
go run ./publication/legacy-model-open-20261004/archive-readback publication/arithmetic-source-delta-20261004/arm64.zip /tmp/gooo-arithmetic-current
mkdir /tmp/gooo-arithmetic-original
unzip -q publication/full-input-separate-arithmetic-20261003/arm64.zip -d /tmp/gooo-arithmetic-original
go run ./tools/audit-separate-arithmetic --mode compare --source-delta \
  --local /tmp/gooo-arithmetic-original \
  --remote /tmp/gooo-arithmetic-current/arithmetic-delta-arm64 \
  --output /tmp/gooo-arithmetic-source-delta.json
```

Use fresh paths and a checkout containing both source revisions. Source pins,
protocol, complete models, paired inputs and all original arithmetic conditions
must validate for the comparison to proceed.
