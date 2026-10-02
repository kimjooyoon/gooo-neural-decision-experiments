# Independent native receipt reader

`main.go.txt` is Go source compiled in the pinned `meta-ontology-go` checkout,
where its internal completeness and runtime packages are available. It makes
zero model calls and executes zero generated programs. The collector records
this file's digest before observations begin.

The reader checks all 400 original generations and runtime records, their source
and parent bindings, model/context identity, initial/feedback chains, actual
prediction counts, 800 completed runs, and all 9,600 finite expectations. It
keeps per-mode process, load, prediction, build and execution measurements apart.
Finite failures remain counted. The output includes the first unresolved
completeness dimension and records the known-task scope.

## Reproduce the registered observation

Use the released SDK v0.2.15 in a clean compiler revision with the V3/V4 dispatch.
Use Go 1.27.1 for the collector, compiler, and its build children. Put compiled
tools outside source checkouts. From this clean research checkout:

```sh
go build -o "$TOOLS/full-input-native-observe" ./cmd/full-input-native-observe
"$TOOLS/full-input-native-observe" \
  runs/own-three-full-input-native-20261003 \
  "$TOOLS/gooo" "$GO1271" "$COMPILER_REVISION" \
  publication/full-input-separate-arithmetic-20261003
```

The output directory must be fresh. The collector inventories retained study
bytes, requires 4 GiB free, binds all 48 model files to the registered public
manifest, and retains the original error prefix with no automatic retry. Every
generation is followed immediately by a build and two executions. No other local
source builds run during collection. The ordinary Go build cache is retained.

After collection, copy `main.go.txt` to a temporary `.go` file in the compiler
root, run it with the absolute capture directory and exact compiler revision,
and remove only that temporary copy. Record its JSON output as
`independent-consumption.json` alongside the original observations. Source code
and raw evidence are retained separately from any summary or publication archive.

This extends the fixed protocol in
[full-input-sdk-native-protocol-20261003](../../docs/full-input-sdk-native-protocol-20261003.md).
The older collector and older 96-generation publication remain byte-frozen.
