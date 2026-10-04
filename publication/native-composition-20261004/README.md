# Native composition of Gooo activity bodies

2026-10-04, macOS arm64. Clean compiler candidate
`c00bd714b4aa8cefc68c2b4d2ccca54fb079b3ed`, Go1.27.1, SDK v0.2.21-experimental;
compiler [PR1219](https://github.com/kimjooyoon/meta-ontology-go/pull/1219).
`build.json` records the actual executable identity. These observations precede
the main promotion and retain this candidate revision.

## One connected program

The source declares seven activities and five typed binds. Two activities use
source-owned integer assembly; their result flows through a range limiter,
an Integer -> Boolean judgment, a Boolean -> Boolean inversion and an integer
fanout. An independent Text root feeds a Text -> Boolean activity. Seven cases
cover ordinary/boundary values, exact int64 maximum, empty text, Korean and emoji.
Each case supplies both root inputs and seven named output expectations.

The compiler retains one optional local model across both assembly activities,
emits the graph's typed driver, builds it once and immediately runs it twice.
The actual model is the unchanged own `set-feedback` FP32 model: metadata
`e9d7f4770d4d8402c64eea116028e6d6c049b1522f50f399c05f2c3571932a3b`, weights
`f5af1e35288cbad938d8416dd51873d83dff369a0e0c6e7e16f6e1f75b7f429b`,
resident tensor bytes74,624. There was no new training or Laya inference.

The first model candidate remains weak: Assemble0/6 and Clamp1/3. Bounded tests
selected a successful body after two and four candidates respectively. Both
model and deterministic routes satisfy the supplied graph expectations, while
their Clamp implementations and resulting source/code hashes differ. Each
route has a fixed source/code/driver after three consecutive constructions.

## Controls and finite counts

There are **six fresh generations** (two modes × three stages) and **six separate
saved-composition replays**. This is one authored graph with repeated controls.

- Fresh generation selection:54/54 supplied activity cases across six constructions.
- Twelve independent graph controls:588/588 supplied runtime expectations.
- Actual intermediate values588 and declared edge deliveries420, in canonical order.
- Twenty-four fresh native runs after twelve builds; both executions agree per control.
- Six actual joint predictions:two per model generation, zero for deterministic
  construction or saved-composition replay.

The saved replays retain earlier model/selection receipts. Their stored prediction
counts are recorded separately from actual calls in that replay. No additional
generation accuracy, model loading or selection is attributed to saved replay.
The observer rejects missing runtime, trace, compiler and prediction-count fields.
The generation selection suite and graph runtime cases overlap; these are finite
behavior observations with no held-out or universal accuracy claim.

## Process time and memory

| Mode / control | Generation median | Full CLI median | Compiled replay phase median | Process max RSS range |
| --- | ---: | ---: | ---: | ---: |
| Model, fresh construction |13.527ms |374.219ms |354.409ms |81.91–83.30MiB |
| Model, saved replay |0ms new generation |469.090ms |453.724ms |81.03–82.52MiB |
| Deterministic, fresh construction |13.011ms |346.143ms |322.474ms |82.50–82.98MiB |
| Deterministic, saved replay |0ms new generation |337.886ms |322.000ms |81.66–82.34MiB |

Each median has three fixed-order samples, with existing Go build cache and
concurrent CI/development activity. Saved replay was slower for the model group
in this small observation; no latency improvement is concluded. Generation time
covers graph checks, source lowering, model loading/ranking, bounded tests and
emission. The compiled replay phase includes source/selection reconstruction,
one native build and two graph executions.

BSD process user/system CPU time and maximum RSS include children, including the
native Go build. Full-command CPU medians were69.32–75.11% of one core based on
these rounded time samples. Whole-computer CPU utilization increase was not
observed. The model's74,624 tensor bytes are distinct from process/build memory.
Per-child build and native-run CPU/RSS observations are retained in each JSON.

## Reproduce

```sh
go run ./cmd/native-composition-observe \
  -gooo /path/to/clean/gooo \
  -compiler-source c00bd714b4aa8cefc68c2b4d2ccca54fb079b3ed \
  -source publication/native-composition-20261004/source.gooo.fixture \
  -cases publication/native-composition-20261004/cases.json \
  -model /path/to/own-three-feedback-v1/set-feedback/models/fp32/model.json \
  -private-out /tmp/gooo-composition-raw \
  -out /tmp/gooo-composition-public
```

Both output directories must be new. The observer measures macOS BSD time;
compiler graph unit/native tests also run in Linux CI. `model-0.json` is a fresh
generation and native execution; `model-0-replay.json` is its separate saved
replay. Each `*-composition.json` can be supplied to `body-compose --composition`.
Checkpoint sources become the next stage's input. Full generation observations
and original expected/actual traces are published; local resource logs remain
in the private measurement directory. Files contain fixed public examples.

[Compiler usage and bounds](https://github.com/kimjooyoon/meta-ontology-go/blob/agent/native-body-composition-20261004/docs/native-body-composition.md).
Record-valued bodies, joins, calls, loops and cross-invocation feedback remain
further language work. The seven-stage example advances actual-body composition;
it does not close the complete domain handoff scope of compiler issue874.
