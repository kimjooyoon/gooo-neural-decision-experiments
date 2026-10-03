# A small order feature preflight

This experiment asks whether 64 directed clause-edge counters can distinguish
some operation orders that the frozen global fragment representation aliases.
The counters use 128 bytes per intention, retain beginning/end edges and consume
every clause rune. The collector validates source framing separately. Full original text is retained
by the caller; old V3/V4 features, model dimensions and loading contracts stay
unchanged. The new feature is an auxiliary experimental API with no trained head.

Before collecting, the fixed cohort contains four arithmetic pairs: add/multiply,
subtract/multiply, negate/add and square/add. Each has Korean/English views and
four whole-instruction wrapper forms: 32 permutation pairs, 64 original-model
calls. All use the same synthetic valid source header, explicitly isolating
intent representation. Authored arithmetic for inputs -2, 0 and 3 establishes
that reversed operations can describe different behavior. No native compilation
or execution is included in this feature preflight.

Record exact old-feature equality, positioned-feature equality, experimental
counter equality and both real frozen-model predictions for every pair. Count
distinguishable pairs without changing examples after collection. Timing belongs
to the frozen model call; a separate benchmark measures feature construction.
The pretrained model never consumes the experimental counters, so prediction
equality is evidence of the old representation's limitation. A new accuracy
claim requires a separately versioned trained model and actual program tests.

The retained negative control `A B A C A D` / `A C A B A D` has identical directed
edge multisets. Finite hash buckets create additional possible collisions.
Punctuation and clause wording also affect the sketch. It has no language parser,
learned semantic vocabulary or general order-preservation guarantee.

Run from a clean source revision, with a new output directory:

```sh
go run ./tools/audit-intent-order \
  --source-revision "$(git rev-parse HEAD)" \
  --output publication/a-new-intent-order-run
go test ./internal/intentorder -run '^TestOrderSketch' \
  -bench '^BenchmarkSemanticIntentOrderSketch$' -benchmem -count 3
```

An adoption decision will use finite separation, remaining collisions, memory
and observed cost. Any later training comparison will keep old features as a
control and evaluate completed Gooo bodies. This is a small representation
experiment motivated by Gooo's existing operation-order counterexample.

The implementation now lives in `internal/intentorder` and takes raw bounded
UTF-8 intention text. The collector uses the unchanged semantic encoder for its
separate source header. This isolates an untrained experiment from the frozen
model's computation package. The original publication pins its earlier source
and wrapped-input benchmark. Existing records remain unchanged; CI compares every
counter and frozen-model output after moving the experiment.
