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
mode and fetches Git history. Its result is recorded separately when available.

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
