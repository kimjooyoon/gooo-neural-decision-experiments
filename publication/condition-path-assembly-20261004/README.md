# Typed condition and body assembly in Gooo

Observed on 2026-10-04 with Go 1.27.1 on darwin/arm64. The compiler candidate was built from feature source `a7dfad39f10ba6591ea013763d2c1e87f8e50322`, SDK `v0.2.21-experimental`. Its SHA256 was `9992cbc47a48e02a123032df810941d173b4ea360b9707a6076636cebf5eb84b`.

This study asks a narrow question: can a small local model help choose among typed Gooo body paths, and does finite validation recover when its first choice is wrong?

## Language change

The source recipe accepts common conditions written with `>`, `>=`, `!=`, and boolean `!`. The compiler normalizes them into the existing typed comparison and equality structure:

- `a > b` becomes `b < a`
- `a >= b` becomes `b <= a`
- `a != b` becomes `(a == b) == false`
- `!condition` becomes `condition == false`

The Gooo SDK operation set does not change. The public compiler example combines conditions, local values, assignment, branches and return. See the compiler [source-path recipe guide](https://github.com/kimjooyoon/meta-ontology-go/blob/main/docs/source-path-recipes.md) and [feature PR #1210](https://github.com/kimjooyoon/meta-ontology-go/pull/1210). The exact dev tree is being promoted to `main` in [PR #1212](https://github.com/kimjooyoon/meta-ontology-go/pull/1212).

## Choice experiment

The source computes `first = input + 2`, `second = input * 2`, chooses one as a branch base, applies a condition-controlled adjustment, then returns a sum. The recipe exposes three binary choices: local reference, branch layout and independent declaration order. The full candidate set has eight combinations.

This nondefault intent requires the second local as the branch base. The model proposed mask 0, the default local. The needed mask 1 ranked third in its predicted probabilities. Finite TDD evaluated masks 0, 4 and 1, then selected the intended typed body. Deterministic search evaluated masks 0 and 1. Both modes passed all **5/5** declared selection cases and emitted the same digest, `sha256:2398a64cb7c6880e4198804a25982375c0f70a8d02c50adafd8524ad214d4b3b`.

Separate compiled execution passed all **11/11** runtime cases in both modes. Six of those inputs were outside the selection suite; that says nothing about whether training had seen them. The four saved runtime receipts include both requests per mode.

The first authored oracle gave input 5 the expected value 25. Evaluating the documented arithmetic gives 24, and the generated function returned 24. The initial 4/5 partial result and its incorrect expectation are retained in `initial-oracle.json` and `initial-oracle-error-generation.json`; the corrected oracle is documented in `oracle-correction.json`. Corrected selection results are 5/5 for both modes.

## Completeness and resource measurements

We report each measured boundary separately. This observation has a typed source-equivalence witness, 5/5 finite selection cases and 11/11 compiled runtime cases. It has **no aggregate completeness score**. These finite observations do not prove all `int64` inputs, unstated natural-language intent, other tasks or other Gooo syntax.

Ten fresh `gooo body-codegen` processes per mode were measured in alternating order. Native Go compilation and execution were excluded from these direct codegen measurements.

| Median across 10 processes | Tiny model | Deterministic |
| --- | ---: | ---: |
| Whole codegen process wall time | 11.087 ms | 8.981 ms |
| Process CPU time (user + system) | 10.098 ms | 8.609 ms |
| Process peak RSS | 21,037,056 B | 20,201,472 B |
| Process CPU / wall, relative to one core | 94.69% | 95.11% |
| Local model prediction | 21.275 µs | 0 |

The model adds about 0.80 MiB median peak RSS and 1.489 ms median process CPU on this fixture. Both modes use about one core while their short process is active; these percentages are not whole-host CPU utilization. The model's prediction itself is small relative to total process time. In this experiment it also caused one extra candidate evaluation.

With `body-path-run --repeat 2 --timing`, both modes passed 11/11 and ran two native executions per request. One cold response took 1,418.228 ms with the model and 1,295.898 ms deterministically; the repeated, artifact-reusing responses took 28.255 ms and 26.700 ms. These are single paired observations and include different native build/cache stages. They do not establish a stable speed difference.

The compact model weights were unchanged; optimizer updates were zero. The model helped express a selectable alternative, while the finite test oracle determined which body met the declared cases. For this fixture, deterministic search evaluated fewer candidates and finished faster.

## Reproduce

Clone `meta-ontology-go` and `gooo-neural-decision-experiments` side by side, then run these commands from the compiler repository root with Go 1.27.1:

```sh
study=../gooo-neural-decision-experiments/publication/condition-path-assembly-20261004
model=../gooo-neural-decision-experiments/models/own-three-feedback-v1/set-feedback/models/fp32/model.json
go1271="$(GOTOOLCHAIN=go1.27.1 go env GOROOT)/bin/go"
"$go1271" version
"$go1271" build -trimpath -o /tmp/gooo-condition ./cmd/gooo

/tmp/gooo-condition body-codegen --json --activity Qualified \
  --path-plan "$study/recipe.json" --path-model "$model" \
  "$study/source.gooo.fixture" > /tmp/condition-model-generation.json

/tmp/gooo-condition body-codegen --json --activity Qualified \
  --path-plan "$study/recipe.json" \
  "$study/source.gooo.fixture" > /tmp/condition-deterministic-generation.json

/tmp/gooo-condition body-path-run --source "$study/source.gooo.fixture" \
  --activity Qualified --path-plan "$study/recipe.json" \
  --cases "$study/runtime-cases.json" --go-bin "$go1271" \
  --repeat 2 --timing --model "$model" --out /tmp/condition-model

/tmp/gooo-condition body-path-run --source "$study/source.gooo.fixture" \
  --activity Qualified --path-plan "$study/recipe.json" \
  --cases "$study/runtime-cases.json" --go-bin "$go1271" \
  --repeat 2 --timing --out /tmp/condition-deterministic
```

The `measure-direct.go` helper records each codegen subprocess's wall time, user and system CPU, peak RSS and prediction time. Use a new output directory for each repetition. `SHA256SUMS` covers the retained inputs, receipts, measurements and helper source.
