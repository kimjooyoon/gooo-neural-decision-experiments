# Source-recipe projection reuse: installed main verification

2026-10-03. [Main #1183](https://github.com/kimjooyoon/meta-ontology-go/pull/1183)
passed all six canonical jobs and the source-bound proof, and was normally
squash merged as `3bee55daf8063b0daee3d543e57a528975bc63ee`. Its clean Go1.27.1
build with SDK v0.2.20-experimental is installed locally. Recipe expansion
reuses the checked original projection from the same request, with source,
fallback, type and replay checks retained.

## Installed behavior

| Route | Generations | Model calls | Native executions | Passed / finite cases |
| --- | ---: | ---: | ---: | ---: |
| Eight existing English/Korean tasks, model and deterministic CLI | 16 | 8 | 32 | 128 / 128 |
| Integrated stream, two requests per route | 4 | 2 | 8 | 32 / 32 |
| One-command Go example, two requests per route | 4 | 2 | 8 | 32 / 32 |
| Total | **24** | **12** | **48** | **192 / 192** |

The eight tasks span the four existing arithmetic families at budget 8. Every
full selection record, generated source and model ranking (timing excluded)
matches the original frozen collection. Stream and Go-example sources also
match their earlier observations. Both retained model routes reuse preparation
on the second request and still predict anew. Deterministic routes make zero
predictions. New training updates: zero.

The Go example's model response intervals are 7.853/2.209 ms and native command
intervals 391.159/302.285 ms. Deterministic intervals are 6.089/2.112 ms and
290.917/290.203 ms. These installation checks use known requests in sequence;
they do not provide a controlled timing comparison with the preceding release.

The separate [paired decoder study](../source-recipe-projection-reuse-20261003)
measures 8,192 decodes: pooled median 0.371→0.292 ms, allocated bytes
728,580→571,800, all 4,096 full document/plan pairs matching. It preserves all
751 slower timing pairs and the preceding comparison without a wall improvement.
Its 512 native generations retain the original budget-1 partial outcomes.

## Exact promotion evidence

- Dev: `f3fb64910eb181bf3cfb9cceb8ea1331ae008703` (#1182 merged).
- Candidate: `629882f4b02430f5aa3e401d13c3a8091f4ec9fb`.
- Previous main and candidate sole parent: `cb2892cb583e693190bd68f75eb4df223819f053`.
- Dev, candidate and installed main tree: `c3c109eab504504898897472c927c88dde284f15`.
- [Main CI 37106995282](https://github.com/kimjooyoon/meta-ontology-go/actions/runs/37106995282),
  attempt 1, whole workflow PASS.
- Immutable artifact `11268926263`, `ci-proof-37106995282-1`.
- Downloaded ZIP SHA256
  `fd818d1c20a34db06b53adb05dfc5a0dded2414a9405126043f596e21462bb93`, matched GitHub metadata.
- Independently verified proof PASS; bundle digest
  `8142ac38397c0629f95e8dd332995398d7f3e8c6b1518c1e7d13d4e8a38a28ae`.
- Live refs, tree, parent, clean non-draft open PR and exact PR/head/base/run/attempt
  were rechecked before the normal merge. Promotion authorization binds the same digest.
- Installed binary SHA256
  `e4ae04d3d1ecaf8d4e547035e0cc714e41be380e28b5887d669d19c346444e4b`.

`native.zip` holds original generation and runtime reports, comparisons and
per-route summaries. `summary.json` is derived from those summaries. Separate
resource files cover the one-shot and stream checks; they include compiler/child
work. Build information uses a basename instead of the private path.
Verify publication files with `shasum -a 256 -c SHA256SUMS`.

[Run the example](../../examples/whole-candidate-order)
· [One-command source and Linux check](../order-example-immediate-20261003)
· [Next retained-executable protocol](../../docs/retained-native-execution-protocol-20261003.ko.md).
