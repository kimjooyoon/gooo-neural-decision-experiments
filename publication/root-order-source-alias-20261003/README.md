# The source side of an order decision can alias too

2026-10-03 follow-up to the [native order study](../native-order-20261003).
Compiler is clean main `729482ed6ff6c3fd2a5664db415ced48e14e122b`, SDK18,
Go1.27.1/darwin-arm64. The unchanged compact positioned-original FP32 model
is used in both requests. Model metadata digest is
`84cb1196b0b28f034d46294a8f3115763ffe7d65f27e2bbb904285b07332d127`;
weights digest is `43c503b224a78f23725101e58cdef1ec4647831329eea4de80f0a259a0253d31`.

The same instruction asks to multiply by two, then add one. In `add-first.gooo`,
the source adds then multiplies, so the requested root order is reverse. In
`multiply-first.gooo`, the source already follows the instruction, so the
requested root order is forward. The unchanged recipe lets the two assignments
and each binary expression's operands swap. Addition and multiplication commute,
so those operand swaps do not alter this example's arithmetic.

| Source order | Required root choice | Proposed root choice | Native expectations met |
| --- | --- | --- | ---: |
| Add then multiply | Reverse | Reverse | 8/8 |
| Multiply then add | Forward | Reverse | 0/8 |

Both source-bound `order` context records have **identical source feature bytes
and complete local model input**:

- Source feature SHA: `ffc5758aa37a73f7aac7223996438d4346b955d06d17a9a641b94534ef705827`.
- Complete root input SHA: `bcc30991bf789bc05e0701fc8c2424aa269e010f475df3d87d9b0ebd2f5b747c`.
- Both proposed full masks are 7; both root choices are `schedule_reverse`.

The complete three-part joint input can differ in the other operand parts. The
published shared architecture evaluates each part with the same local eight-unit
judge and sums the selected local scores. It has no learned interaction between
those parts. This observation establishes the local source-order alias and the
two actual outcomes; it does not assert that every bit of both full joint input
or probability vectors is equal.

Two generations, two actual predictions, four compiled executions, 8/16 finite
expectations met, zero training updates. The recipe has one selection input,
`0 -> 1`, and a one-candidate budget. Eight native inputs range from -3 through 4;
seven per request are selection-disjoint. Every failure and source binding is
preserved. No process comparison, performance gain or full-domain score is claimed.

## Next small language/model step

The native study established that retaining natural-language positions changes
features but did not improve these frozen weights. This counterexample adds a
necessary source-side check before contrast training: opposite source-relative
answers must receive distinguishable descriptions of the permitted alternatives.

A bounded next prototype should describe the two changed assignment operations
in source order, along with the complete instruction. It can use a small fixed
array over the existing typed plan. First test source reversal, renamed locals,
equivalent commutative operands and unsupported shapes. Preserve current feature
versions and weights; publish a separately versioned input/training experiment
only after the intended distinctions and limits are observable. This prototype
and new training are not implemented by this evidence publication.

## Reproduce

For each source file, run the pinned compiler's `body-codegen --json --activity
Compose --path-plan recipe.json --path-model` with the existing
`publication/full-input-separate-arithmetic-20261003/models/compact/positioned-original/fp32/model.json`.
Pass the saved generation and same source/recipe to `body-execute` with
`--cases cases.json` and Go1.27.1. Invocation semantics are the same as the
[native collector](../../cmd/native-order-observe), with this explicit source
pair and one-case recipe. `SHA256SUMS` covers all files except itself.
