# Source-ordered native input joins — 2026-10-04

Gooo bodies now receive several declared scalar inputs and connect them to
actual producer results. `Add(Integer, Integer)` keeps its two Integer slots
separate; `Difference` verifies the operand order. `Label(Integer, Boolean, Text)`
gets its Integer from Add and its other values from the caller. A repeated-Text
root compares two independently supplied strings.

Think of each activity as a small workbench: its numbered input sockets have
declared types, and a wire delivers the actual value from another workbench.
The optional own model helps assemble the Left body before these values join.
Gooo owns the socket order, body lowering, connections and finite checks.

## Implemented path

```gooo
activity Add(Integer, Integer) -> Integer computes "return input0 + input1"
bind Left.result -> Add.input0
bind Right.result -> Add.input1
```

- 1 input uses the existing `input`; 2..16 inputs use `input0` through `input15`
  in source order. Inputs are immutable; local `let` values can be assigned.
- Source-ordered references survive the bidirectional model and semantic IR,
  including repeated types that collapse into one unique PROV `used` fact.
- A port accepts at most one producer. An unbound multiple-input port uses the
  case key `Activity.inputN`; a single-input root keeps the activity name.
- Generated Go calls use that order. Native traces retain each slot's actual
  value, entity identity and producer identity.
- One build immediately runs the whole graph twice. Saved composition replay
  reconstructs the source/projection and performs zero predictions.

The own model still proposes three binary construction choices inside the
single-Integer Left assembly. Add, Difference, Label and Compare use their
declared pure bodies. This experiment extends the language around that model.

## Finite observations

Compiler source: `5af78bcbdf5c881ff7e2bf7034be5018b524ae04`,
[compiler PR 1221](https://github.com/kimjooyoon/meta-ontology-go/pull/1221).
The source and cases here are exact copies of its runnable example.

One authored graph has seven activities, six bound edges, six external input
slots and twelve total input slots per case. Seven cases cover repeated types,
noncommutative argument order, mixed types, partial binding, zero/false/empty,
Unicode comparisons, line endings and int64 wraparound.

| Control | Observed result |
|---|---|
| 6 fresh constructions: 2 modes × 3 consecutive checkpoints | 294/294 named output expectations |
| 6 separate saved replays | 294/294; zero new predictions |
| All 12 controls | 588/588 outputs; 1,008 input observations; 504 bound deliveries; 24 native runs |
| Actual own-model calls in these controls | 3; one for each fresh model construction |
| Fresh selection examples | 36/36 across 6 constructions; kept separate from runtime outputs |

The first model-ranked mask 6 passed 0/6 selection examples. The next checked
candidate, mask 7, passed 6/6. Deterministic enumeration began with mask 0 at 0/6
and checked all 8 candidates. The model path checked 2 candidates. Both emitted
the same source body, Go functions and input delivery driver through three
consecutive checkpoints. These are repeated controls of one graph; the counts
do not estimate independent tasks or universal intent accuracy.

`additional-controls.json` retains two simultaneous requests, three additional
input scenarios replayed through both saved routes, and an intentional wrong
Difference expectation. The wrong expectation stays visible as 48/49 while both
native executions agree. One additional Left input overlaps a selection example;
these controls have no holdout designation.

## Time and memory

Three fresh measurements per mode, fixed model-then-deterministic order and warm
Go caches, on macOS/arm64 with Go 1.27.1:

| Measurement | Own model | Deterministic |
|---|---:|---:|
|Median whole graph generation | 9.711ms | 6.396ms |
|Whole command, including native build/two runs | 330.474–565.592ms | 300.716–381.500ms |
|Process peak RSS | 81.08–82.64MiB | 82.17–83.44MiB |
|Evaluated Left candidates per fresh request | 2 | 8 |

`metrics.json` retains each wall time, user/system CPU time, peak RSS and native
runtime time. Generation includes source preparation and ordinary body checks;
it is a whole graph measurement, separate from individual prediction latency.
Peak RSS includes the command's build/execution resource observations rather
than only the neural tensor. The tensor remains 74,624 bytes. Host CPU utilization
was not sampled alongside this final series, so no whole-computer increase is
attributed to the model. These timings do not establish a general speed gain.

An earlier candidate (`0ab58d1b`) was measured while full tests/race checks ran
locally. Its source partition exceeded the compiler CI extractor's capacity.
The measured candidate `5af78bcb` separates plan, input delivery and preparation
files, but its preparation function still exceeded the extractor's rendered
capacity. Follow-up source `6b6c86f2` divides preparation into smaller functions.
The observations here retain their exact measured source; merge and installation
are pending. Earlier measurements remain private, and any installed-source
observations will be recorded separately.

## Use the model and rerun

The unchanged own model is
[gooo-three-choice-feedback-tiny-v1](https://huggingface.co/asketeddy/gooo-three-choice-feedback-tiny-v1/tree/24238ca67048b36bc305731799985271bb752dd1/models/set-feedback/fp32).
Metadata SHA256:`e9d7f4770d4d8402c64eea116028e6d6c049b1522f50f399c05f2c3571932a3b`;
weights SHA256:`f5af1e35288cbad938d8416dd51873d83dff369a0e0c6e7e16f6e1f75b7f429b`.
No new model weights were trained for these observations.

With a clean compiler built from the recorded source, run from this directory:

```sh
gooo body-compose --source source.gooo.fixture --cases cases.json \
  --model /path/to/model.json --out /tmp/gooo-input-joins
gooo body-compose --source source.gooo.fixture --cases additional-cases.json \
  --composition model-0-composition.json
```

Omit `--model` for deterministic assembly. Use a new output directory. The
compiler guide explains
[input keys and replay](https://github.com/kimjooyoon/meta-ontology-go/blob/5af78bcbdf5c881ff7e2bf7034be5018b524ae04/docs/native-body-composition.md).
The Go observer at `cmd/native-composition-observe` accepts `--input-joins` and
checks per-port values against actual producer outputs while collecting costs.

Next language work can use these explicit sockets to study small models that
rank permitted argument/connection choices. Record-valued bodies, calls, loops
and cross-invocation feedback still require dedicated compiler implementations.
