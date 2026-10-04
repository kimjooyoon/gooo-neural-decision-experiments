# Assemble record field values in Gooo

An activity can declare alternatives for the expressions inside a record
constructor. The compiler checks the types, an optional local model orders the
combinations, and finite cases measure how much of the declared result each
combination completes. The selected body is emitted as ordinary Go.

Think of the source as a tray with labelled spaces. Each field has two permitted
parts. The model suggests an order for trying the parts; the cases show which
spaces contain the expected values. A partly filled tray remains useful evidence.

## Put the choices and expectations beside the body

The complete [source example](candidate/source-budget-8.gooo.fixture)
declares a three-field `Candidate` record, two typed inputs, conditional local
replacement, and a connection to a text-label activity. Its selection block is:

```gooo
activity Select(Candidate, Boolean) -> Candidate computes `
let copy = input0
if input1 && copy.state != "ready" {
    copy = Candidate{title: "draft", state: "wait", reason: "deferred"}
} else {
    copy = input0
}
return copy
` assembling {
    choice "title" field_value at "0" alternative "input0.title" intent "Keep the original title."
    choice "state" field_value at "1" alternative "\"ready\"" intent "활성화된 후보를 준비 상태로 만든다."
    choice "reason" field_value at "2" alternative "input0.reason + \":accepted\"" intent "Append the accepted marker."
    value_case "[{\"title\":\"Hello\",\"state\":\"queued\",\"reason\":\"check\"},true]" -> "{\"title\":\"Hello\",\"state\":\"ready\",\"reason\":\"check:accepted\"}"
    attempts "8"
}
```

`at` is the zero-based occurrence of a constructor field's **value expression**
in baseline source order. A choice retains that expression as `value_first` and
permits the supplied `alternative` as `value_second`. The alternative follows
the body's lexical scope and pure-expression rules. It must have the declared
field type. A field choice can use a local or parameter field, a text literal or
a text expression. A mismatched Boolean expression receives a type diagnostic.

`value_case` takes a JSON positional input array and a complete expected output
record. Inputs may mix Integer, Boolean, Text and declared records. Every record
field is required. Input and expected JSON are normalized into the assembly
contract, so formatting changes keep the same observations.

## Generate, inspect and execute

From a checkout with Go 1.27.1:

```sh
go run ./cmd/gooo body-context --include-plan --activity Select \
  examples/body-codegen/record-field-assembly.gooo.fixture
go run ./cmd/gooo body-codegen --json --activity Select \
  examples/body-codegen/record-field-assembly.gooo.fixture
go run ./cmd/gooo body-compose \
  --source examples/body-codegen/record-field-assembly.gooo.fixture \
  --cases examples/body-codegen/record-field-assembly-cases.json \
  --out /tmp/gooo-record-field-assembly
```

Use `body-codegen --path-model /path/to/model.json`, or
`body-compose --model /path/to/model.json`, to attach a compatible local model.
Omitting the model orders masks from zero upward. Model workspaces and search
observations belong to each request; retained immutable weights can be shared.

`body-context` exports the exact field IDs, alternatives and Korean/English
intent without evaluating cases or calling a model. Saved generation includes
the planning baseline and selected `picked` labels. `body-realize` reconstructs
the selected source and finite observations with zero new predictions.
`body-compose` compiles and executes the connected activities, preserving actual
record values, field IDs and producing-activity identities at each connection.

The retained NDJSON worker supports parallel field-body construction. Its scalar
`--execute` envelope does not carry record inputs yet; use `body-compose` for
native record graph execution.

## Read partial completion

The report's `record_assembly` keeps every attempted mask and its case/field
counts, plus the selected candidate's expected and actual values. Selection
maximizes matching fields, then whole matching cases, then the earlier candidate
in the ordered search. Search stops at full finite case completion or the budget.

The example's five selection cases contain 15 expected fields. With
`attempts "1"`, the baseline matches **2/5 cases and 6/15 fields**. With eight
attempts, the declared alternatives reach **5/5 and 15/15**. These are distinct
completeness dimensions in the shared receipt. An unfinished candidate carries
`PARTIAL_FINITE` and a progress receipt; its checked Go and Gooo source are still
available for inspection and execution.

The separate seven-case runtime file scores both connected activities, producing
14 output expectations per execution. Five inputs overlap selection cases; two
add text escaping and another unchanged-record path. Keep interpreted selection
counts and compiled runtime counts separate when reporting results. These counts
describe the supplied cases; extending the input domain requires more evidence.

## Current model transfer

The existing frozen three-choice model can order exactly three field choices.
The named `gooo/record-field-ordinal-context/v1` contract maps the source choices
to ordinal binary decisions. Its full text carries stable field IDs, both
expressions and intent; expected outputs and case inputs are excluded. This is
an experimental transfer from an integer decision model. Field-specific model
training remains subsequent work.

Other choice counts, a different model family, or an oversized complete context
record a deterministic decline with zero predictions. An encoded request makes
one initial local prediction. Finite checks then compare candidates in that
order. The field profile currently keeps that initial order; failed-case
re-ranking remains available in the separate Integer assembly profile.

The first development run completed all 15 expected fields, but the frozen model
ranked the successful combination last and spent the same eight attempts as
deterministic search. Repeated measurements should report prediction cost,
attempt count and field completion together before attributing a speed benefit.

## Bounded implementation

- 1..6 disjoint binary field choices and 1..64 attempts.
- 1..128 typed selection cases; 1..16 activity inputs.
- Required string record fields, using the existing 16-field record profile.
- Source up to 128 KiB; each JSON value up to 32 KiB; each supplied text up to
  1,024 UTF-8 bytes.
- Pure body conditions, expressions and whole-local assignments follow the
  ordinary typed body profile. Calls, loops and nested record fields require
  subsequent language work.

Native Go records are value structs. The finite evaluator uses a fixed array of
16 string headers with a nominal record identity. Copying a local copies those
headers independently; strings retain ordinary immutable storage.

The next learning target is reusable field decisions: source-owned alternatives,
small bilingual intent and the observed field improvements. A useful model
should complete more fields under the same attempt budget across new source
shapes. Prediction latency and memory belong beside that result.
