# Keep the assembled graph and run the next inputs

Gooo can now keep one compiled value graph while applying an ordered series of
input/expectation suites. The first suite builds the executable; later suites
run that same executable on their current inputs. Construction consults the
optional tiny model once. Runtime makes zero model calls.

Like keeping a tool assembled on a workbench, the next piece of material changes
without assembling the tool again. Each execution still measures its own result.

## What was measured

[Candidate observations](candidate) bind clean compiler source
`38962397144d4418b75f571471225ff02d712617`, Go1.27.1 and the existing frozen own
models. [Compiler PR1227](https://github.com/kimjooyoon/meta-ontology-go/pull/1227)
adds `body-compose --case-series ... --repeat 2` and an owned graph executor.
The candidate was measured before main promotion and installation.

One authored two-node graph constructs a three-field record and then its label.
Three suites change Korean/English titles, states and reasons. The third suite
deliberately asks for a different label: its actual record still passes and its
label expectation misses. Repeating the suites twice retains scores
**4/4, 2/2, 1/2, 4/4, 2/2, 1/2**. The observer recomputes these counts from actual
values and checks current inputs against the corresponding suite.

There are 50 captured CLI invocations, 120 suite frames and 240 native executions:
eight fresh constructions, six saved retained requests and 36 saved stateless
requests. Fresh model requests make five judgments in total (three FP32 and one
per ternary profile); disconnected requests and saved/runtime requests make zero.
All 20 six-suite workloads retain **280/320 named expectations and 480/480 record
fields** in aggregate. Repeated controls measure this single source shape.

## Paired saved workload costs

Each pair uses the same saved construction, program bytes, ordered inputs and
expectations, with zero new model calls on both sides. The stateless workload
starts six CLI commands and builds six times. Retained execution starts one CLI
command and builds once. Startup count is therefore part of this comparison.
Three trials per mode alternate the workflow order.

| Saved workload median | Six stateless commands | One retained command |
| --- | ---: | ---: |
| FP32-origin wall time | 2,324.88ms | 509.65ms |
| FP32-origin CPU user+system | 1.46s | 0.38s |
| FP32-origin peak RSS | 82.75MiB | 82.73MiB |
| Deterministic-origin wall time | 2,328.73ms | 502.34ms |
| Deterministic-origin CPU user+system | 1.47s | 0.36s |
| Deterministic-origin peak RSS | 83.06MiB | 82.20MiB |

The wall reduction is about 78% for this six-suite CLI workload. Peak RSS remains
similar. CPU is process user+system time including builds and children; RSS is
the maximum process observation, with a maximum across the six stateless
commands. Whole-host utilization change is unmeasured.

Fresh FP32 prediction median is **18.834µs** across three constructions. Fresh
first-suite runtime median is **371.323ms**; the fifteen subsequent retained
frames have a **22.001ms** median. The initial disconnected request took
1,743.89ms end-to-end; the other two took 509.76/515.24ms. Original first and warm
observations remain in the data. These costs depend on the local cache/device.

The first saved FP32 trial separates a96.49ms build, a248.26ms first native
launch/execution and an8.32ms second execution of the same suite. This identifies
both build and first-execution wall cost. The underlying cause of the first-run
delay is unmeasured. Keeping the executable avoids repeating both stages;
the text reader shows build and both native times separately.

One fresh PTQ and one QAT request also retain the same generated program and
finite results. Each has 18,752 resident tensor bytes versus FP32's 74,624.
Prediction observations are 19.041µs/19.000µs. The `ternary_base3_5` matrices pack
five trits per byte, approximately 1.6 stored bits per matrix weight. Each weight
file is 3,854 bytes including FP32 biases; decoded tensor residency is the
separate 18,752-byte count. The earlier 2-bit wording is corrected here from the
actual metadata and files. 1.58 bits describes ternary information content. Weights and training are
unchanged. Equal finite results on this shape leave broader model quality open.

## Try the source and read the current result

From a compiler checkout containing PR1227, using an actual Go1.27.1 executable:

```sh
go run ./cmd/gooo body-compose \
  --source examples/body-codegen/record-field-assembly.gooo.fixture \
  --case-series examples/body-codegen/record-field-case-series.json \
  --repeat 2 --out /tmp/gooo-series-example
```

Add `--model /path/to/model.json` for compatible local ordering. The output
directory contains `composition.json`, `runtime-history.json`, `case-series.json`,
the selected Gooo source and executable Go source. `runtime` in stdout is the
latest history frame. Replaying `--composition` starts one new native build,
then reuses it within that request; it makes zero new predictions.

The Go observer accepts `--verify-json CAPTURE.json --compiler-source REVISION`
and `--saved` for a saved capture. Add `--text` for a concise per-suite summary
with output/field percentages, current model calls, build/reuse and actual mismatches:

```sh
go run ./cmd/composition-series-observe \
  --verify-json publication/retained-composition-20261005/candidate/fp32-0-fresh.json \
  --compiler-source 38962397144d4418b75f571471225ff02d712617 --text
```

Its native CI repeats four profiles and their
saved counterparts on Linux. The executor owns at most one compiled artifact;
program changes replace it and close releases it. Local tests cover changed
programs, current inputs, concurrent callers and cancellable waiting.

[Independent Linux readback](linux-readback) rechecks the eight native responses
from the first research CI job. [The next compact field curriculum](next-field-learning.md)
describes source-family splits and per-field intent; new training is subsequent
work. This change reduces repeated build cost in an existing language flow.
