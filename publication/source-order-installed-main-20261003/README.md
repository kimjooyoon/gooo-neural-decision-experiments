# Source-bound plan export installed on main

Compiler [PR1174](https://github.com/kimjooyoon/meta-ontology-go/pull/1174) merged
to dev as `5eaea0f5bf5d96de29c63bf1726acb6c65b155fc`.
[PR1175](https://github.com/kimjooyoon/meta-ontology-go/pull/1175) promoted the
same tree to main `322ab8d02a3adfd4b37bcc7a7cd40dcb01dd7cf3`.
[Main CI run 37096927156](https://github.com/kimjooyoon/meta-ontology-go/actions/runs/37096927156),
attempt1, completed successfully. Immutable proof artifact11264942275 was downloaded
and independently verified, then the current source/target tuple and native
protection checks were rechecked before normal PR merge.

The installed compiler was built from the clean main revision above using
Go1.27.1 and SDKv0.2.18. Its binary SHA256 is
`e446e7ea1ab7f6d6257187a0189aeee0bfde8d02e162b1440b7a392a01a3f0f9`.

The installed binary reproduces all **48 source-plan exports** from the earlier
[descriptor preflight](../source-order-preflight-20261003). All 24 source-order
reversals remain distinguishable; local renaming16/16, commutative changes16/16
and language-channel changes24/24 retain their expected invariance. The export
stage invokes zero models, candidate tests, training updates or native runs.
`exports.zip` retains every source, recipe, bound context and observation.

Separately, installed condition-chain and constant-body smoke checks made four
generations and eight compiled executions: **26/26 finite expectations matched**.
Two actual model calls used the existing shared judge. Model and deterministic
arms generated identical bodies for each of those two functions. These are
installation checks; their timings are not an isolated performance comparison.

The new [whole-candidate judge](../order-judge-initial-20261003) is public in the
research runtime and on Hugging Face. Its own direct compiler/native invocation
remains the next integration stage. The installed change provides source-bound
`body-context --include-plan` for that work.
