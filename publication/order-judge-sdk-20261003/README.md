# Released Go SDK: complete whole-candidate replay

The new `orderfacts` and `orderjudge` packages are published in
[v0.2.19-experimental](https://github.com/kimjooyoon/gooo-decision-runtime/releases/tag/v0.2.19-experimental).
[PR #4](https://github.com/kimjooyoon/gooo-decision-runtime/pull/4) was normally
merged at `69b2f6f017d1af314aa27522891ac8970595aaca`.

The package extraction uses a separate 12-file inventory pinned to research
`ada8a3fd728a9288b5d7060982bcb3801e8bccfa`. Only imports and gofmt are relocated.
The existing 64-file SDK inventory and V3/V4 implementations remain unchanged.

Both macOS arm64 and Linux amd64 executed **160 requests, 960 actual searches,
480 actual predictions and 320 baseline SDK searches**. Complete records match
the original research run after removing only `SearchNS` and
`Ranking.predict_ns`. This compares predictions, selected bodies, failures,
descriptors, aliases and receipts, not just aggregate scores.

Canonical records SHA256:
`f0f6864734aed872cd84766979abde26733f12b505072db0dda6078f9398377f`.

- Pull-request CI: [37098090895](https://github.com/kimjooyoon/gooo-decision-runtime/actions/runs/37098090895), SUCCESS.
- Push CI: [37098088108](https://github.com/kimjooyoon/gooo-decision-runtime/actions/runs/37098088108), SUCCESS.
- Tested PR head: `4a9dd5d3b1b0ce8a966725740b3247a29a080069`.
- GitHub PR merge checkout: `5ad20243cfda78bb0b04aa1a90eab2e8339d41f6`.
- Head, merge checkout and merged main tree: `15d2a6f9a673b3e487ce7aa140e7b5a74a9cbe27`.
- Immutable Linux artifact: `11265052293`, downloaded bytes SHA256
  `9e4b2ef7f43e8336f7a0147d03a4e009e00419a78ef5996e7371b82c0aa87715`.

`linux-artifact.zip` retains the actual artifact; `artifact.json` records its
GitHub identity. `local-manifest.json` and `local-race.log` retain local source
identity and package coverage. The compiler imports this public released module
with no local replacement. [Native compiler observations](../order-judge-native-20261003)
follow this SDK replay without training or weight changes.

The research repository's earlier [whole workflow 37097293126](https://github.com/kimjooyoon/gooo-neural-decision-experiments/actions/runs/37097293126)
finished with 16 passing jobs and the retained historical arithmetic comparison
failure. Its new whole-candidate job passed; the entire research workflow is not
reported as green.
