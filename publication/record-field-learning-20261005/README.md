# Learning from actual Gooo field choices

Gooo describes a small construction space: conditions and local assignments,
three record fields, two permitted expressions per field, intent and examples.
Our own small model orders the eight possible combinations. The compiler
assembles those pieces and measures the resulting behavior. A useful analogy
is a drawing with numbered parts: the model proposes an assembly order, and
execution shows which parts fit the requested drawing.

This pilot adds actual field expressions to the model context and makes
ordinary conditional bodies easier to write. `if` can fall through without
`else`, including early-return guards and nested local updates. Every reachable
return, variable scope and assignment type is still checked by the compiler.

## What was executed

- Clean compiler source: `af811fe3b5dbb709edd61fe9de216f6af0850911`.
- Go SDK: `v0.2.22-experimental`; feature contract
  `triple_record_field_context_v1_joint_v1`.
- Published preparation/trainer source: `4441c221dfa1b13230d9469d56974b5d7742b174`.
- 768 source views: eight expression/body families × six field orders × eight
  first/second orientations × Korean/English. Whole-family split: 384 train,
  192 calibration, 192 held-out test. Exact cross-split contexts/features overlap
  zero times. The three roles are title copying, `ready` state and reason suffix.
- Fresh Go-seeded initialization, 240 FP32 plus 240 QAT optimizer updates on MPS.
  Calibration selects epoch and temperature. Test rows are first used in the
  separate Go audit. PTQ adds zero updates.
- 192 export/Go logit comparisons; largest absolute error `0.00000304`.
- 100 native constructions, each built once and run twice: 80 observations on
  four held-out source views with new inputs and 20 on paired counter-intents.
  Five profiles and budgets 1/2/4/8. These are repeated construction observations;
  the separate goal of 100 distinct experimental approaches remains open.

The compiler's finite selection examples determine completeness. The model
context carries complete alternatives and intent, with zero test inputs or
expected outputs. Inference occurs once before construction; native execution
and repeated execution perform zero model calls. Disconnected construction uses
deterministic ordering.

## Complete-mask ordering on 192 held-out views

| Export | First candidate | Within two | Within four | Warm Go prediction median | Weight file | Resident tensors |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| FP32 | 192/192 | 192/192 | 192/192 | 15.958 µs | 74,624 B | 74,624 B |
| PTQ ternary | 70/192 | 101/192 | 161/192 | 16.000 µs | 3,854 B | 18,752 B + 8 B scales |
| QAT ternary | 134/192 | 192/192 | 192/192 | 16.000 µs | 3,854 B | 18,752 B + 8 B scales |

The 768/24/8 network has 18,656 parameters. Five trits per byte store matrix
weights at about 1.6 bits each, with a theoretical information limit of about
1.58 bits. Go expands ternary matrices to int8; the packed file size and runtime
tensor memory are reported separately. Feature parsing creates temporary
objects, and total process memory exceeds tensor storage.

## Actual execution and the intent gap

At budget one on the four selected normal source views (48 observed record
fields), deterministic ordering matched 32/48 fields, the frozen ordinal model
36/48, and each fresh field model 48/48. At budget eight every profile reached
48/48. The four views are a smaller subset of the 192-view ranking audit;
PTQ's complete result here coexists with its worse ranking across that full set.

Then we kept the same candidate expressions and changed intent/examples to
copy the input state into the title, set `wait`, and prefix the reason. At
budget one, the fresh models matched 12/24 runtime fields, deterministic order
20/24 and the frozen ordinal control 16/24. Full budget eight recovered 24/24
for all profiles. Named output completeness also fell to 8/16 at budget one:
unchanged guard cases pass, while active transformations fail.

The current training repeats three normal intents. Its strong normal ordering
therefore establishes a narrow assembly shortcut. The counter-intent experiment
shows the next training requirement: paired, opposing goals over identical
alternatives, varied Korean/English wording and literal/copy/concatenation roles.
Keep these observed counter-intents as diagnostics and reserve fresh pairs for
subsequent evaluation. No weights or thresholds were retuned after seeing them.

## Resources and time

The actual optimization loops took 1.234 s total and 0.584 process CPU s.
The whole Python training process took 3.18 s wall and 1.99 CPU s, with peak RSS
429.36 MiB. Sampled MPS tensor peak was 1.98 MiB and driver peak 50.72 MiB.
Host GPU utilization and changes to whole-machine CPU utilization were not
measured. Allocation is not a utilization percentage.

Across the 100 constructions, maximum compiler-child peak RSS was 84.20 MiB,
and maximum generated-program peak RSS was 4.91 MiB. The two native runs per
construction consumed 1.522 process CPU s in aggregate. Outer construction
commands recorded 26.573 CPU s across all 100 invocations, including waited
child process work on this machine. All process identities, times and partial
values remain in the raw captures. Prediction medians above include feature
projection after train/calibration warmup; fresh CLI prediction timings are
retained separately. Whole-command timings include compilation/startup and
fixed profile order, so they do not isolate a model speedup.

## Run it

From the research repository root, use Go 1.27.1 and the clean pinned compiler:

```sh
unzip -q publication/record-field-learning-20261005/evidence.zip -d /tmp/gooo-field-pilot
GOWORK=off GOTOOLCHAIN=local go run ./cmd/record-field-curriculum \
  --audit-prepared /tmp/gooo-field-pilot/prepared \
  --models publication/record-field-learning-20261005/models \
  --output /tmp/gooo-field-audit-new
GOWORK=off GOTOOLCHAIN=local go run ./cmd/record-field-curriculum \
  --audit-prepared /tmp/gooo-field-pilot/prepared \
  --verify-native /tmp/gooo-field-pilot/native \
  --output /tmp/gooo-field-recount-new.jsonl
gooo body-compose --source publication/record-field-learning-20261005/demo.gooo.fixture \
  --cases publication/record-field-learning-20261005/demo-cases.json \
  --model publication/record-field-learning-20261005/models/fp32/model.json
```

Audit output must be a fresh directory. Numerical parity records are provided
beside the model directory, and the full archive also retains that layout.
The native study accepts `--native-prepared`, `--compiler`, `--compiler-source`,
`--frozen-model`, `--models` and a fresh `--output` to reproduce all 100 observations.

`evidence.zip` retains complete Go preparation inputs, source/context exports,
optimizer journals, all model variants, all prediction rows, and 100 raw native
captures with new cases. `SHA256SUMS` pins every published file. The size report
checks the 64 MiB uncompressed pilot budget. No external pretrained tensors
initialize these models. Our earlier Laya experiments motivated the local
decision approach; this release starts from the recorded own initializer.

The public model package is
[asketeddy/gooo-record-field-tiny-v1](https://huggingface.co/asketeddy/gooo-record-field-tiny-v1).
