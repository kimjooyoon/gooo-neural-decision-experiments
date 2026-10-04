# Native record values in Gooo bodies

Observed on 2026-10-04 at clean compiler candidate
[`fa3c892791a2fd775a265c7524c9ee59adce45bf`](https://github.com/kimjooyoon/meta-ontology-go/commit/fa3c892791a2fd775a265c7524c9ee59adce45bf),
[development PR1223](https://github.com/kimjooyoon/meta-ontology-go/pull/1223).
Go 1.27.1, macOS arm64. The feature is now merged and installed at main
`b629a664`; [fresh installed observations and field completeness summaries](installed/README.md)
are retained separately from these candidate measurements.

## What a user can express

The source declares `Candidate { title, state }` and `Review { summary, reason }`.
Their fields use the existing required-string, one-value entity profile. A body
can construct a complete record, read a field, copy a record into a local, replace
that local with a complete value, return it, and compare two values of the same
record type. Scalar and record inputs can share an activity. Each activity has
one output and at most sixteen ordered inputs.

This fixture connects six activities:

```text
Score(Integer) ──> Propose(Integer, Text) ──> Candidate
                         │                       │
                         ├──> Echo(Candidate) ────┤──> Same(Candidate, Candidate)
                         └──> ReviewCandidate(Candidate, Boolean) ──> Review ──> Label
```

The graph carries actual values. For example, `Propose` creates a title and a
`ready`/`wait` state; `ReviewCandidate` reads that state and the caller's Boolean
to return `accepted`/`deferred`. Each delivery retains the source field identity,
declared field order and actual value. Native code uses value structs in declared
field order and stable names derived from the full source identity. JSON keeps
the original lower-case field names. This is a small form being passed between
desks: its title and status remain visible at every handoff.

## Own model and deterministic construction

The unchanged own model ranks three operand choices in **Score**. The other
bodies, including record construction and field reads, are written in Gooo.
The model proposed mask 6 first; it passed 0/6 selection examples. The next mask,
7, passed 6/6. Model search evaluated two of eight declared candidates;
deterministic search evaluated all eight. Both paths produced the same selected
Gooo source, native Go and driver across three consecutive generations and saved
replays.

The model is
[`asketeddy/gooo-three-choice-feedback-tiny-v1`](https://huggingface.co/asketeddy/gooo-three-choice-feedback-tiny-v1/tree/24238ca67048b36bc305731799985271bb752dd1/models/set-feedback/fp32),
frozen revision `24238ca67048b36bc305731799985271bb752dd1`. Its tensor payload is
74,624 bytes; metadata and weight digests appear in every model construction.
This observation updates compiler behavior and evidence using the existing
weights. It provides no new record-trained model or accuracy gain estimate.

## Finite observations

The main cohort consists of one six-activity graph, six authored runtime cases,
two modes, three fresh constructions per mode and six saved replays:

| Observation | Count |
| --- | ---: |
| Named output expectations met | 432/432 |
| Actual input slots checked | 648 |
| Producer-to-consumer deliveries checked | 432 |
| Record field observations checked | 1,152 |
| Native executions | 24 |
| Actual fresh model predictions | 3 |
| Actual predictions during saved replay | 0 |

Field observations include input and output positions. Repeated values can occur
at several positions; these counts describe trace coverage. The controls reuse
one fixture. Some runtime integers overlap selection examples.

Additional controls are separate in `additional-metrics.json`:

- Two fresh processes started together, one per mode: 72/72 named outputs, both
  completed. Process wall times were approximately 660ms and 802ms. This is a
  finite concurrency observation; arbitrary schedules need further measurement.
- Eight further runtime-only cases through each saved composition: 96/96 named
  outputs, 256 field observations, zero new predictions. Inputs include branch
  crossings, caller approval changes, emoji, quotes, tabs and a 1,000-byte title.
- Changing only one expected `state` from `ready` to `wait` produces **35/36**
  named matches. The trace retains the actual `ready` value and the failed
  expectation. At this scope, completeness is 97.22% of named expectations.
  Expected output-record fields also match 35/36 in this control. These finite
  denominators let a user locate the gap; they do not assign confidence to all
  possible programs.

## Time and memory

Three fresh requests per mode, in a fixed model-then-deterministic order with a
warm native build cache:

| Median | Own model | Deterministic |
| --- | ---: | ---: |
| Graph generation | 12.801ms | 11.774ms |
| Compiled graph runtime/build observation | 319.866ms | 311.529ms |
| Whole command wall time | 343.507ms | 334.904ms |
| Process user CPU | 0.15s | 0.15s |
| Process system CPU | 0.11s | 0.11s |
| Maximum observed resident memory | 83.39MiB | 83.17MiB |

Actual model prediction median was **26,875ns (26.9µs)**. Model loading is retained
separately in `.composition.model.setup_ms`, outside request-generation timing.
Saved controls retain the historical prediction duration under
`stored_prediction_ns`; their actual model-call count is zero.

Process resource measurements include native builds and child processes. They do
not measure a host CPU utilization increase caused by inference. The current
model reduces candidate evaluations while generation takes a little longer in
this sample. The dominant cost here is native build/execution, making reusable
compiled artifacts a useful next usability experiment.

## Reproduce

Build the clean compiler revision above with Go 1.27.1. From this research
repository's root, set `gooo_bin` to that executable, put Go 1.27.1 on PATH, and run:

```sh
"$gooo_bin" body-compose \
  --source publication/native-record-values-20261004/source.gooo.fixture \
  --cases publication/native-record-values-20261004/cases.json \
  --model models/own-three-feedback-v1/set-feedback/models/fp32/model.json \
  > /tmp/gooo-records.json
jq .composition /tmp/gooo-records.json > /tmp/gooo-record-composition.json
"$gooo_bin" body-compose \
  --source publication/native-record-values-20261004/source.gooo.fixture \
  --cases publication/native-record-values-20261004/cases.json \
  --composition /tmp/gooo-record-composition.json
```

Omit `--model` for deterministic construction. The CLI can also write a new output
directory with `--out`: inspect `realized.gooo`, generated Go, driver, composition
and runtime there. `go run ./cmd/native-record-observe --verify-json FILE
--compiler-source fa3c892791a2fd775a265c7524c9ee59adce45bf --mode model` checks the
captured six-case profile. Add `--saved` for a saved-composition observation.

The observer's default mode captures the twelve-row macOS resource cohort into
fresh public/private directories. `--controls` adds the separate controls to an
existing cohort. Its tests check the published observations and reject absent,
reordered or changed fields, wrong producer values and altered model identity.
The research CI reconstructs the graph and verifies fresh/saved actual field
delivery on Linux at the same compiler revision.

## Current boundaries and next work

Required string fields form the currently supported record profile. Optional,
many-valued and nested records require further entity-profile work. Body calls,
loops and field mutation remain outside this pure body grammar. Registered
domain `run` chains have their own runtime contracts; issue874 remains open.

Useful next steps are guided record-field choices declared inside `assembling`,
field-level expected/missing measurements in normal CLI output, and reuse of
compiled graph artifacts. These connect small model decisions to the language
while keeping the selected source available for the next construction.
