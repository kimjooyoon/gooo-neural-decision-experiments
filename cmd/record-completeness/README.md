# Read finite record completeness

This Go command reads a full captured `gooo body-compose` JSON response. It
prints named-output matches, output-record field matches and the exact gaps.
The denominator comes from caller expectations in that execution. Input field
observations describe delivery and retain their separate trace counts.

If the response contains field assembly, it also prints the selection contract's
case and field ratios, evaluated candidates and remaining field differences.
Selection cases and actual runtime cases keep separate denominators. For example:

```sh
go run ./cmd/record-completeness \
  --input publication/record-field-assembly-20261004/candidate/budget-4-model-0.json
```

```text
Selection Select: cases 2/5 (40.00%); fields 12/15 (80.00%); evaluated candidates: 4
Selection case 0 Select.state: mismatch; actual="wait" expected="ready"
Named outputs: 6/14 (42.86%); unobserved: 0
Record output fields: 17/21 (80.95%); unobserved: 0
```

The complete output includes every remaining selection and runtime difference.
The optional JSON `record_selection` contains a separate score per activity.
The reader recomputes both reported selection ratios from expected/actual values.

```sh
go run ./cmd/record-completeness \
  --input publication/native-record-values-20261004/partial.json
```

```text
Named outputs: 35/36 (97.22%); unobserved: 0
Record output fields: 35/36 (97.22%); unobserved: 0
Case 0 Propose.state: mismatch; actual="ready" expected="wait"
```

The actual command also prints the complete changed `Propose` value. Use
`--json` for `gooo/finite-record-completeness/v1`: gaps retain case index,
activity name/ID, field name, actual and expected values. A field with no
expectation is `unobserved_expectation`; one with a known expectation and no
actual value is `unobserved_actual`. Missing expectations receive no match
credit. When a denominator is zero its percentage is `null` in JSON and
`unobserved` in the readable summary.

The reader checks the captured named totals against actual value comparisons.
Object member order and JSON Unicode escaping are presentation details. Exact
integers remain exact during comparison. Source activity order and the record
field declaration are required. These small summaries help turn a finite gap
into the next task: change a body, add an expectation or investigate an
unobserved value, then measure that same contract again.
