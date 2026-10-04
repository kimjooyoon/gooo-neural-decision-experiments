# Source-owned Gooo body assembly

Observed 2026-10-04 on macOS arm64 / Go 1.27.1, compiler candidate
`0f7ec4c0135be3dca24df59f8f6ee60bd307f538`, SDK v0.2.21-experimental.
[Compiler PR #1215](https://github.com/kimjooyoon/meta-ontology-go/pull/1215)
adds an activity's `assembling` block through syntax, semantic IR and execution.

## Authoring change

The baseline body, Korean/English intent, permitted paths, finite expectations
and budget live together in Gooo. The regular body command discovers the contract.
The same source supports an explicit small model or deterministic search, followed
by immediate compiled execution.

`source.gooo.fixture` has two activities: `Qualified` selects a different local
before conditional updates; `Clamp` lowers an `else if` chain. Each declares three
binary choices and eight combinations. `external.gooo.fixture` removes only their
assembly blocks. Its full plans come from the real compiler context export and
retain identical alternatives, intent, cases, activity IDs and budgets.

## Finite results and ranking quality

Two activities × two authoring forms × model/deterministic × four requests gives
**32 constructions**, **128/128 selection expectations**, **288/288 independent
runtime expectations** and **64 fresh native runs**. In each command, requests 2–4
reuse its owned artifact and perform fresh replay and two runs. The 16 model
requests made 16 joint predictions; deterministic requests made zero. All inference
was local.

The same activity/mode produces identical model input hashes, candidate counts
and generated Go digests across authoring forms. Model/deterministic routes may
choose different declaration or branch presentations with the measured finite
behavior preserved.

| Activity | Model first candidate | Model candidates | Deterministic first candidate | Deterministic candidates |
| --- | ---: | ---: | ---: | ---: |
| Qualified | 0/5 | 3 | 1/5 | 2 |
| Clamp | 1/3 | 4 | 3/3 | 1 |

These are two authored tasks repeated under controls. The model ranks worse than
deterministic order here. Finite testing recovers the requested behavior within
the budget. First failures and rank costs are retained for future model work.
The final 100% describes the supplied cases.

## Timing and resources

Warm medians use three responses after the first request per command. Response
includes construction, replay, native tool/artifact checks, two runs and original
artifact saving. Initial input/model setup and final stdout are excluded;
generation is a contained phase and must not be added again.

| Activity / route | Source response | External response | Source generation | External generation |
| --- | ---: | ---: | ---: | ---: |
| Qualified / model | 28.573ms | 26.654ms | 2.443ms | 2.041ms |
| Qualified / deterministic | 27.656ms | 25.247ms | 1.839ms | 1.492ms |
| Clamp / model | 26.950ms | 25.983ms | 1.422ms | 1.203ms |
| Clamp / deterministic | 25.991ms | 24.268ms | 1.161ms | 0.786ms |

The embedded contract adds source-owned preparation/checking. These small
sequential samples show that cost alongside order/cache noise. First responses
were 317–499ms across eight commands and included native builds. Joint prediction
medians were 19.77–20.48µs. The authoring change has no observed speed/quality gain.

`resources.json` retains eight macOS `time -l` command observations. For the two
source/model commands, CPU time/wall time was 62.1% and 81.0% of one core; maximum
RSS was 82.48–83.48MiB and reported peak footprint 15.20–15.73MiB. These include
loading and native compilation/execution. Model resident tensors were 74,624 bytes.
Whole-host CPU change and model-only process RAM were unobserved. Rounded CPU
totals and the two tasks constrain comparisons.

## Reproduce

The unchanged own bundle is
[`models/own-three-feedback-v1/set-feedback/models/fp32`](../../models/own-three-feedback-v1/set-feedback/models/fp32).
Metadata SHA256: `e9d7f4770d4d8402c64eea116028e6d6c049b1522f50f399c05f2c3571932a3b`.
Weights SHA256: `f5af1e35288cbad938d8416dd51873d83dff369a0e0c6e7e16f6e1f75b7f429b`.
This is the independently trained Gooo joint judge, using existing weights with
zero new GPU training and zero Laya calls in this observation.

```sh
go run ./cmd/source-assembly-observe --gooo /path/to/gooo \
  --compiler-root /path/to/meta-ontology-go \
  --private-out /tmp/gooo-assembly-raw --out /tmp/gooo-assembly-public
```

Use fresh directories. This reproducer handles the fixed public fixtures;
the real compiler parses Gooo and validates plans. `--summarize-only` re-reads
retained observations with zero model/native calls. Raw command directories
support `gooo body-path-run --verify-timing --out <directory>`.

Public files contain metrics, resources, build identity, source/plans/cases and 32
generation records. Original runtime envelopes and resource logs stay local due to
local tool paths. Published runtime counts are diagnostic extracts; reproduction
supplies new inspectable runtime records. An initial extractor used wrong JSON
field names. Extraction was corrected from unchanged original observations and
now has explicit field-name regression tests.
