# Prepared-candidate compiler: installed main verification

2026-10-03. Main [PR #1179](https://github.com/kimjooyoon/meta-ontology-go/pull/1179)
was normally squash merged as `8117fbaefac490a28e78c956bf1d40aed1372608` after the
six canonical checks and proof bundle passed. This clean revision was built with
Go1.27.1 and installed locally with public SDK v0.2.20-experimental.

Eight existing Korean/English requests from four arithmetic families were each
replayed with the original 16 KiB model and without a model. **16 generations,
8 model predictions, 32 compiled executions, 128/128 finite expectations** passed.
Every full selection record, generated Go body and model ranking matched the
frozen original native collection. Fresh CLI processes report `reused:false`;
retained reuse was measured separately in the [API](../order-prepared-native-20261003)
and [worker](../prepared-worker-native-20261003) observations. No weights changed.

## Promotion evidence

- Dev source: `9fc2634720f0a43ca7d72bac46aacba5c77822ce` (#1178 merged).
- Main candidate: `7fcf0a1f2bd55a20886935aec7706b9807acced8`.
- Previous main and candidate's sole parent: `98701e473bc4084a422c5fa7acaa6631118779e4`.
- Dev, candidate and installed main tree: `6d25e5ab78d3fb407c4a708042998a131134400d`.
- [Canonical CI 37103723095](https://github.com/kimjooyoon/meta-ontology-go/actions/runs/37103723095), attempt 1, PASS.
- Immutable proof artifact: `11267341721`, `ci-proof-37103723095-1`.
- Downloaded ZIP SHA256: `ead2d581610b0472a9c512cdd59d587a9b687cf1481161fc91102a3ca3785ec8`;
  matched the GitHub artifact digest before extraction.
- Independently verified proof: PASS; bundle digest
  `176efda1d182ee57cbbdfde355d486583671aa34809b89e47c59560584fe48c3`.
- Live dev/main refs, equal trees, exact candidate, open non-draft clean mergeable
  PR and all six same-run/attempt checks were reread before the normal merge.
  Promotion authorization is bound to the same bundle digest. No force/admin
  merge or branch-rule change was used.
- Installed binary SHA256:
  `9fa1c4da2d082c897bbba5ed5b9f203b6c944d47cd46961904524b671a30fb21`.

`native.zip` contains the source, recipe, generation, original comparisons,
independent native reports and summaries. Each report binds its compiler revision.
The build-info first line uses the binary basename in place of its local path.
Verify these publication files with `shasum -a 256 -c SHA256SUMS`.

The integrated `gooo body-path-stream` usability command is a subsequent change
in [PR #1180](https://github.com/kimjooyoon/meta-ontology-go/pull/1180); this installed
revision still uses the standalone worker for retained streaming.
