# Read finite record completeness

This Go command reads a full captured `gooo body-compose` JSON response. It
prints named-output matches, output-record field matches and the exact gaps.
The denominator comes from caller expectations in that execution. Input field
observations describe delivery and retain their separate trace counts.

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
