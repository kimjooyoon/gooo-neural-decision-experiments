# Native Integer local repair

The frozen `studies/native-integration-gaps/integer-literal-local.gooo` fixture
failed in native compiler source `60cf7f49b0e302a6bebb42bc8da90f3ed19b2b82`:
a bare constant local became Go `int` instead of the activity's `int64`.
`native-result-before-fix.json` preserves that actual failure.

Compiler correction source: `6a8011bf144abdd5634c7656f977835dbdd90c4e`.
The new source-pinned compiler emits `var result int64 = 5`. Its native JSON
receipt, including typechecking and route equivalence, is
`native-result-source-bound.json`. Its executable SHA-256 is
`7f738bee6039b7aaccbcb3bd8efa0230ffb2de2a91e1f6b25ed5ce28e0de3fe0`.
The old study and source-pinned compiler are unchanged.

The first local build disabled VCS stamping. Its `native-result.json` correctly
records `UNBOUND_LOCAL_SOURCE` despite successful generation, so it is retained
as incomplete provenance rather than counted as a source-bound receipt. That
build's executable digest is
`fe399851bda4e6d8abd009e6da6721437b0291e92317ad75715a9379ba1849c7`.
The separate corrected build used `-trimpath -buildvcs=true` from a clean tree.

This is a deterministic compiler correction; no model/provider calls occur.
The implementation and regression tests are in
[native PR 1105](https://github.com/kimjooyoon/meta-ontology-go/pull/1105).
