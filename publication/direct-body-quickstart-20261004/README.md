# Direct Gooo body quickstarts — 2026-10-04

The existing source, recipe and finite-case files can go straight to
`gooo body-path-run`. The [Korean example](../../examples/whole-candidate-order)
shows the complete model and deterministic commands. No fixture-preparation
program is needed for its eight original expectations.

Clean compiler candidate `2420ad198480ca4592306df7db630a82de6d9ed4`, Go1.27.1
and SDK v0.2.21 made four constructions and eight native runs with **32/32**
expectations. Both modes generated the same Go, and each repeated construction
reused the native artifact while executing its current inputs twice.

We also downloaded only the compact shared FP32 pair from Hugging Face revision
`7c2781cbbcf81edba89a50d0423276c042b082de` using `hf download --local-dir`.
Both downloaded files are regular files. Metadata SHA256 is
`6478334e1e181d0854865d1da54b9c2203eaa0e488f2b05617c92e27f08b755f`;
the 8,288-byte weights SHA256 is
`43c503b224a78f23725101e58cdef1ec4647831329eea4de80f0a259a0253d31`.
They match the published compact FP32 pair used by the preceding studies.

From the compiler repository root, the following authored compound example
made two actual predictions, four native runs and **6/6** finite expectations:

```sh
hf download asketeddy/gooo-shared-judgment-tiny-v1 \
  --revision 7c2781cbbcf81edba89a50d0423276c042b082de \
  --include 'research/compact-runtime-20261003/models/fp32/*' \
  --local-dir ./gooo-models

gooo body-path-run \
  --source examples/body-codegen/typed-path-compound.gooo.fixture \
  --activity Combined \
  --path-plan examples/body-codegen/typed-path-compound-plan.json \
  --cases examples/body-codegen/typed-path-runtime-cases.json \
  --model gooo-models/research/compact-runtime-20261003/models/fp32/model.json \
  --repeat 2 --timing --out compound-model-results

gooo body-path-run --verify-timing --out compound-model-results
```

Use a fresh output path. This SDK profile checks regular, non-symlink model files;
keep `--local-dir` to materialize metadata and adjacent weights. Source/recipe
regular-file symlinks follow the compiler's separate input contract.
Omitting `--model` selects deterministic construction.

| Original observation | First response | Repeated response | Finite expectations |
| --- | ---: | ---: | ---: |
| Whole-candidate order model | 1,085.365750ms | 77.261875ms | 16/16 |
| Deterministic order example | 622.538250ms | 72.422792ms | 16/16 |
| Downloaded shared model, compound example | 884.413250ms | 70.435292ms | 6/6 |

Each response covers request decoding, construction and execution; initial model
preparation and saving/output are separate. These are six observed constructions
of existing authored tasks, with four actual predictions, twelve native runs,
zero new intents and zero training updates. The two model families have different
inputs and tasks. Host CPU and model-only RAM were unobserved in this quickstart.

`observations.zip` retains all 54 original files, including requests, timing,
source/recipe/cases, generation and actual runtime outputs. The reader checks
the exact candidate source, current saved Go, model calls and int64 case values.
It performs zero inference or native execution.

```sh
shasum -a 256 -c publication/direct-body-quickstart-20261004/SHA256SUMS
mkdir /tmp/gooo-direct-saved
unzip -q publication/direct-body-quickstart-20261004/observations.zip -d /tmp/gooo-direct-saved
go run ./publication/legacy-model-open-20261004/archive-readback publication/direct-body-quickstart-20261004/observations.zip /tmp/gooo-direct-saved
go run ./publication/direct-body-quickstart-20261004/readback /tmp/gooo-direct-saved
```
