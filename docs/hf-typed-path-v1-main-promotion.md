## Native integration is on main

[PR 1113](https://github.com/kimjooyoon/meta-ontology-go/pull/1113) merged
the direct structural model integration to native compiler main at
`9b8b900e32384b5397fb3d08dc545e22cc7530da` on 2026-09-30 22:51:48 UTC.
The protected promotion retained the exact dev tree with the previous main
parent. The six canonical checks and actual PR-bound promotion authorization
passed in [CI 36785977424](https://github.com/kimjooyoon/meta-ontology-go/actions/runs/36785977424).
The downloaded proof/append-only receipt independently passed the Go verifier
before the ordinary squash merge; no protection or required-review change was
made. Required approving reviews remain zero and Guardian is absent.

A clean main build with Go 1.27.1 then loaded the published FP32 structural
model directly, made three fresh local predictions, passed the fixture's three
explicit cases, type checking and replay, and recorded the actual main source
SHA. It made zero external calls. This main smoke is separate from the primary
32-cell probe and is not an additional independent arithmetic holdout study.
The main and dev trees were checked equal after merging. This records deployed
source availability; it does not claim a newly packaged native release binary.
Post-main push CI was queued when this note was written; the successful
authorization above comes from the actual pre-merge PR run.

Use native `body-codegen --path-plan [--path-model]`, documented in the
[Gooo compiler body-codegen guide](https://github.com/kimjooyoon/meta-ontology-go/blob/9b8b900e32384b5397fb3d08dc545e22cc7530da/docs/language/body-codegen.md).
For example, use `positioned-random/models/fp32/model.json` from this public
repository with the compound plan/fixture supplied by the compiler. Omitting
the model retains deterministic finite search. These are closed bilingual
structural choice models; they do not freely generate arbitrary program text.
All nine published weight bundles remain unchanged.
