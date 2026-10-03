# Whole-candidate judge: installed main replay

On 2026-10-03, compiler [PR #1177](https://github.com/kimjooyoon/meta-ontology-go/pull/1177)
was normally merged as `98701e473bc4084a422c5fa7acaa6631118779e4`. This is the exact
tree of dev `1ee5a528c0543c15e2af28a7af1b152919762145`, with SDK v0.2.19.
A clean Go1.27.1 build was installed and exercised on Apple M4/macOS arm64.

## What ran

Eight frozen `new-template` requests cover all four arithmetic families and both
Korean and English. Each uses source order 0, desired order 1, budget 8, and both
deterministic and model routes. Each generated body was immediately compiled and
executed twice before proceeding to the next generation.

- **16 generations, 8 real local model predictions, 32 compiled executions.**
- **128/128 finite expectations passed**, counted once per generated result.
- Full model score/ranking arrays equal the original native observations.
- Generated source and the entire ordinary search record also match; no fields
  are removed from those comparisons. The separate prediction timer is outside
  that search record and is retained in each original generation receipt.
- Compiler source SHA and source binding are checked on every generation.
- Original 16 KiB weights are unchanged; additional training updates: **0**.

This is installation replay on previously observed requests. The larger
[native cohort](../order-judge-native-20261003) retains all budget-one failures,
cost measurements and source identities. No new speedup or broad accuracy claim
is derived from this smoke run.

## Merge evidence

The complete [main CI run 37100166540](https://github.com/kimjooyoon/meta-ontology-go/actions/runs/37100166540)
and [dev CI run 37099085714](https://github.com/kimjooyoon/meta-ontology-go/actions/runs/37099085714)
succeeded. Their immutable proof ZIPs were downloaded, matched to GitHub's
artifact digests, and independently verified using the clean compiler source's
proof verifier. Main's proof binds PR 1177, attempt 1, head
`3258262f45f2514dce17e772e627eecd901510d8`, base
`322ab8d02a3adfd4b37bcc7a7cd40dcb01dd7cf3` and a PASS promotion authorization.
Immediately before normal merge, live PR/ref identities, exact dev tree and sole
main parent were checked again. No protection setting changed.

The separate dev-only historical v17 compatibility certificate reported
`AXIS_MISMATCH`; the same failure is observable on predecessor run 37092454144.
GitHub's optional AI security check could not start because its monthly quota was
exceeded. These are separate from the successful canonical CI and are not counted
as passing. Research workflow 37099147973 has 16 passing jobs and one retained
historical arithmetic replay failure; its new whole-candidate job passed.

## Files and reproduction

`installed.zip` contains all source/recipe/case files, original generation and
execution receipts, complete selected-record comparisons, prediction comparisons,
request list and summaries. `manifest.json` binds the compiler binary, source,
weights, counts and CI identities. `buildinfo.txt` records a clean VCS build.
The two proof ZIPs and their artifact metadata retain the CI originals.

Verify the published file inventory with `shasum -a 256 -c SHA256SUMS`.
For a compact fresh run, follow the [Korean whole-candidate example](../../examples/whole-candidate-order)
using compiler `98701e47` and the published model. Its source and recipe are one
of these eight requests. Omit `--path-model` for the deterministic route.

### 한국어 요약

새 모델의 직접 연결을 main에 병합하고 깨끗한 설치본에서 재현했습니다.
한영 요청 여덟 개를 모델 사용·미사용으로 생성해 32회 실제 실행했고,
128/128 기대값이 일치했습니다. 모든 순위·선택 결과는 기존 관측과 같습니다.
준비물 재사용의 성능 개선은 [별도 후속 계획](../../docs/order-preparation-protocol-20261003.ko.md)입니다.
