# Validation and actual public-model use

The original frozen research source is b7ce25b4cccf128a9bbc12968cdab88ff266159a.
[Run37226752241](https://github.com/kimjooyoon/gooo-neural-decision-experiments/actions/runs/37226752241)
passed all26jobs; source PR17 merged to research main172d366a. Model weights and
training remain unchanged after the diagnostics. Later usability/installation
receipts keep their own source identities.

## Numerical, field and native observations

`linux-ci.zip` preserves the original successful Linux replay artifact, including
360freshly generated graphs,720native executions and the Go model audit.
Its SHA256 is3dc4281c2024fca7e88d2c1089699c80bf9941fe973728603ea2723d59979eeb.
The verifier checks its digest,120fixed NumPy/Go logits, model quality counts,
source/case/selection identity, current producer input delivery, actual values,
active/unchanged/required-changed field counts and process completion.

For a fresh local replay, extract the **parent** `evidence.zip` once into a pilot
folder, then read the Linux ZIP in memory:

```sh
go run ./validation/verify-linux/main.go \
  /path/to/extracted-pilot ./validation/linux-ci.zip /path/to/new-validation.json
```

The Linux verification uses ZIP members directly and writes one report. The
original extraction for the Mac comparison is separate. The saved observation
is `linux-validation.json`. `observer-diagnostic.json` retains the initial
observer's family7 field-name mistake and its correction using stable field
metadata. That correction changed the observer; model/training were retained.

## Public weights and worker observations

`worker-trials` retains the original first-launch observation and all three
registered repeats against compiler d515db9d. Every trial uses the same pinned
public weights, six actual native controls and32mixed-goal worker requests with
two workers. The first response arrives while stdin remains open. This checks
an input-EOF barrier within the60second observation window.

`public-dogfood/main.go` accepts original publication, exact public download,
extracted pilot's `native` directory, compiler, worker, fresh output, exact clean
compiler revision and HF revision. It compares every supplied publication file.
At the original measured revision this was21files; later documentation records
are included when a later complete revision is supplied.

```sh
go run ./validation/public-dogfood/main.go \
  ./publication /path/to/pinned-public /path/to/pilot/native \
  /path/to/gooo /path/to/gooo-body-worker /path/to/new-observation \
  EXACT_COMPILER_SHA EXACT_HF_REVISION
```

The worker observations measure body construction and finite selection fields.
Compiled graph execution is measured by the separate six `body-compose`
controls. `installed-public-dogfood.json` records the actual installed main;
`installation.json` records clean source, Go/SDK versions and binary digests.
The dev/main proof receipts carry their independent verification and artifact
digests.

## Reading a proposal

`read-model.go` retains JSON output by default. `--text` shows names, original
intent, candidate expressions, probabilities and the proposed expression. Its
tests cover all8mask orientations and incomplete declared shapes. The captured
public example is `field-choices.txt`.

## Storage and scope

`storage.json` records the original64MiB local pilot target,37.91MiB original
pilot,38.32MiB later Linux replay and combined76.22MiB logical raw size. The
combined observation exceeds that first target. Preserve the registered plan,
both observations and the compressed archives. Core timing, warm worker timing,
full command timing and process/host/GPU resource scopes remain distinct.

The benchmark uses three authored roles and eight authored body families;
all eight required mask combinations occur in training. New wording and
cross-field dependencies are continuing language work. The four repeated32
requests and finite graphs describe the measured completion boundary.
