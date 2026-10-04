# Installed main: ordered native input joins

2026-10-04, macOS/arm64. Compiler main
`1bef47fc1e7a4aca8ffb1cbd5a5aa7ded2353ce5`, tree
`11f8ff1f01700cb08c8ca36335085363e9acab1e`, Go 1.27.1 and SDK
v0.2.21-experimental. Development [PR1221](https://github.com/kimjooyoon/meta-ontology-go/pull/1221)
and main [PR1222](https://github.com/kimjooyoon/meta-ontology-go/pull/1222)
merged after their own complete CI runs and independently verified proofs.
Main CI37199114334 used snapshot `268c347b08d880d9657f8eae3e794ab306553940`
and proof artifact11302318970. That snapshot had the live dev tree and sole live
main parent. `main-verification.json` retains the exact tuple and artifact digest.

The CLI and body worker were rebuilt from clean merged main and installed
together. `installation.json` retains source, tree, Go/SDK versions and actual
binary digests with local paths removed. The staged CLI ran the new graph at
49/49 and separately replayed its saved composition at49/49. The worker shares
the installed source; the observations here use the CLI `body-compose` entry.

## Fresh observations on the installed executable

The same authored graph has seven activities, six edges and twelve input slots
per case. It joins repeated Integer inputs in different operand orders, mixes
Integer/Boolean/Text, partially binds an activity and compares independently
supplied Text inputs.

Two modes each have three consecutive fresh constructions and three separate
saved replays: twelve controls, **588/588 named output expectations**, 1,008
input-slot observations, 504 bound deliveries, twelve new builds and24 native
executions. The three fresh model requests made three actual predictions;
saved replay and deterministic construction made zero. Historical predictions
stored in a replay are recorded separately. Fresh selection expectations are
36/36 across six constructions, separate from the runtime output score.

All corresponding Gooo, function-Go and driver hashes match the preceding
measured candidate. Compiler identifiers now bind to installed main. Weights
are unchanged: the actual public FP32 tensor remains74,624 bytes. The model
checks two Left candidates (first0/6, second6/6); deterministic search checks
eight. Ordinary multi-input bodies follow their declared source programs.

| Control group, n=3 | Generation median | Full CLI median | Peak RSS range |
|---|---:|---:|---:|
| Model, fresh | 9.449ms | 316.959ms | 81.81–82.09MiB |
| Deterministic, fresh | 8.480ms | 312.954ms | 81.33–82.28MiB |
| Model, saved | 0 new generation | 311.461ms | 81.84–82.17MiB |
| Deterministic, saved | 0 new generation | 310.289ms | 81.77–83.56MiB |

These are new timings, collected in fixed model-then-deterministic order with
warm Go caches. Resource observations include native compilation/children.
`metrics.json` keeps each wall time, user/system CPU time and maximum RSS.
The host's before/during CPU utilization was not sampled. No whole-computer
utilization increase is attributed to this model, and the small repeated
samples do not identify causes of differences from earlier timings.

## Separate controls

`additional-controls.json` describes five requests outside the twelve above.

- Fresh model and deterministic requests started together and completed
  normally at49/49 each, **98/98** total and four native runs. Full wall times
  were590.885ms and743.220ms. Actual predictions were one and zero.
- Three additional input scenarios replayed both saved routes at21/21 each,
  **42/42** total and four native runs, with zero new inference. One Left input
  overlaps a selection example; these scenarios have no holdout designation.
- Changing one Difference expectation from-11 to11 leaves runtime completion
  and repeated output intact, while the score remains **48/49**. The actual
  value and failed expectation remain visible. This request makes zero predictions.

The counts describe one graph and its additional finite scenarios. They do not
estimate independent task accuracy. Record-valued bodies, calls, loops and
cross-invocation feedback remain further compiler work.

## Run again

From this directory, with the installed executable and the
[unchanged public model](https://huggingface.co/asketeddy/gooo-three-choice-feedback-tiny-v1/tree/24238ca67048b36bc305731799985271bb752dd1/models/set-feedback/fp32):

```sh
gooo body-compose --source source.gooo.fixture --cases cases.json \
  --model /path/to/model.json --out /tmp/gooo-installed-input-joins
gooo body-compose --source source.gooo.fixture --cases additional-cases.json \
  --composition model-0-composition.json
```

Omit `--model` for deterministic assembly. Use a new output directory.
[The Korean guide](https://github.com/kimjooyoon/meta-ontology-go/wiki/Native-Input-Joins)
explains numbered inputs, partial binding and how to read actual values.
