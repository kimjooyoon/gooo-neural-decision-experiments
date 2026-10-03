# Complete public SDK replay — 2026-10-03

The research kernels now ship in
[SDK v0.2.15-experimental](https://github.com/kimjooyoon/gooo-decision-runtime/releases/tag/v0.2.15-experimental).
This appendix retains the existing local and Linux replay reports from its exact
release source `59c8d342da4475506b90954469aa201f85cadeb3`.

| Observation | Platform | Complete inputs | Actual predictions | Result |
| --- | --- | ---: | ---: | --- |
| [Local SDK report](arm64.json) | darwin/arm64 | 18,432 | 36,864 | PASS |
| [CI SDK report](linux.json) | linux/amd64 | 18,432 | 36,864 | PASS |

Each run covers four training arms, three weight variants, three input forms
and 512 views per condition, in expanded and compact layouts. Every feature,
hidden/logit/probability bit digest, first mask and full ranking matches the
frozen explicit-arithmetic observations. Optimizer updates: zero.

The reports name 48 model files, 36 frozen journals, the reference manifest,
the SDK extraction manifest and the actual source revision. The adjacent
`SHA256SUMS` binds both retained report files.
[Linux CI run](https://github.com/kimjooyoon/gooo-decision-runtime/actions/runs/37070241916)
also passed formatting, vet, unit and race checks. Its `sdk-full-input-replay`
artifact contains the original Linux report.

This stage measures numerical transfer into the public SDK on previously observed
tasks. Collection wall time includes loading, journal decoding and comparison;
it requires separate experiments to estimate inference latency or process RAM.
At this SDK replay stage the compiler used SDK v0.2.14. Native V4 generation and
execution were planned in the
[registered integration protocol](https://github.com/kimjooyoon/gooo-neural-decision-experiments/blob/daecfea3583614e006c960de263448a1645a9190/docs/full-input-sdk-native-protocol-20261003.md).
The later [native integration results](../../docs/full-input-native-results-20261003.md)
completed those 400 generations and 800 executions, with 816 model predictions
and 9,600/9,600 finite expectations. Current installed main `ed2c2cac` uses SDK
v0.2.20-experimental. The original SDK reports below retain their own v0.2.15
source and prediction counts.

Frozen model/data reference:
[complete arithmetic appendix](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/600dc282fb059918c0fd6588d955d3a56c380b58/publication/full-input-separate-arithmetic-20261003),
manifest SHA-256 `ce4ad854c0d7fcb3e7fa049658f40ea4b0393a9bff39a2ef21f64d83f221bc3a`.

The same appendix is available in the
[Hugging Face model repository](https://huggingface.co/asketeddy/gooo-shared-judgment-tiny-v1/tree/main/research/full-input-sdk-20261003).
