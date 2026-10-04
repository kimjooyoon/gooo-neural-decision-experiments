# Sequential record updates in Gooo

This is a new language representation experiment after the frozen shared-field
study. Keep those weights, its measurements and its wording regression intact.
Implement local record field assignment and a separate `field_update` choice
whose ordinal counts only field assignment right-hand sides. Existing
`field_value` ordinals keep counting constructor fields.

Gooo owns sequential values, branch scope and record copying. A later update may
read an earlier updated field or a saved local. Native Go value structs and the
finite evaluator must agree. Updating a parameter, an undeclared local, an
undeclared field, a scalar receiver or a temporary value is outside this profile.
Use fixed sixteen-field string-header storage; avoid mutable record maps.

## Registered observation

Publish this plan before collecting results. Freeze six source shapes: ordered
cross-field reads, a saved record copy, a guarded update, a saved scalar field,
same-field concatenation, and a second copied local after an unrelated update.
For each shape keep Korean and English intents and budgets 1/2/8. Compare
deterministic order and the existing public QAT shared-field model: 72 graphs,
144 native executions. Each graph contains Select -> Label and four explicit
input cases. Count selection fields, compiled fields, named outputs, attempts,
prediction time, command wall/CPU and peak RSS separately. Preserve every failure
and partial result. Reconstruction and compiled execution make zero predictions.

## Source preparation diagnostic (before the valid-program comparison)

The initial observer at0c641c5 completed48 graphs;24 snapshot graphs were
rejected at type preflight because their saved locals appeared only in the
alternative expression. Preserve all72 initial captures separately. Repair the
snapshot baseline RHS from`"deferred"` to`saved.state` (record) or`saved` (scalar),
so every baseline and every alternative uses its declaration. The six shapes,
goals, case values, budgets and frozen weights remain the same. Run the full
72-tuple comparison with these valid baselines, retain both phases, and label
them separately. The144 native executions refer to the valid-program phase;
initial48 graphs also ran96 executions. This amendment is published before
the repaired comparison and does not count as another approach.

This tests frozen expression-only ranking on sequential programs. Its context
does not resolve local reaching definitions or learn cross-field dependencies.
Case outcomes choose a complete or partial program under the declared budget.
No tuning after the results, no independence claim for the program's semantics,
and no generalization claim from six authored shapes. This is one distinct
representation approach; 72 executions do not count as 72 approaches toward100.

Keep new raw captures below16MiB, share immutable model files, bound each native
command to60 seconds and retain process failures. Add positive and negative
source, evaluator/native parity, mixed constructor/update ordinal and saved
checkpoint tests. Publish language source/examples and the raw experiment on
GitHub; report whole-command costs without attributing startup to inference.
