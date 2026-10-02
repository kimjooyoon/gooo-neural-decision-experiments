# One useful probe, then native Gooo generation

Observed 2026-10-03 KST with Go 1.27.1 on darwin/arm64.

- SDK source: [`e3b3e4f15f0afd6aa610f9528349b93263830351`](https://github.com/kimjooyoon/gooo-decision-runtime/commit/e3b3e4f15f0afd6aa610f9528349b93263830351).
- Native compiler: clean main `fc0e99c4efb12dc525d574d2c4b9b0e1101c376b`, SDK v0.2.15.
- New API: the SDK development source above; it is not part of released v0.2.15.
- Compiler input: exact `body.gooo`; activity `Adjust`.

Both `input-2` and `2-input` satisfy the initial observation `2 → 0`.
The SDK ranks probe inputs `[2,3,0]`; index 1 distinguishes the candidates.
The declared independent specification `2-input` provides the new observation
`3 → -1`. Two surviving candidates become one. There are eight evaluator attempts
before the new observation and seven after it, with zero model calls or training
updates in this example. The output matrix is 16 KiB; this is not process RAM.

The completed example emitted Gooo. The installed clean main compiler generated
Go from it, then a native Go test executed inputs `[-7,0,2,3,9]` and matched
`[9,2,0,-1,-7]`. All five logged expectations passed. This is one deliberately
small mechanism example. The five checks are a supplied finite suite, and do not
constitute unseen-task accuracy or an all-input proof. No latency/CPU/RSS
benchmark or model-quality improvement is claimed for this observation.

## Reproduce

At the pinned SDK source:

```sh
go run ./examples/probe-ranking
```

Using the pinned compiler and this directory:

```sh
gooo body-codegen --json --activity Adjust body.gooo
```

For the captured Go and explicit finite checks, copy `generated.go.txt` and
`generated_test.go.txt` into a temporary directory as `generated.go` and
`generated_test.go`, then run `go test -json generated.go generated_test.go`.
The archived `native-tests.json` is the original successful output; timestamps
and execution time will vary. `generation.json` is the original compiler JSON
from a relative source path, avoiding private host paths.

## Inspection and limits

`example.json` retains both complete ranking records, their source/case/probe
digests, every candidate output and the oracle observation. It reports candidate
disagreement, not learned information gain. A caller with no existing oracle
must leave the expected value unresolved.

Local SDK vet and full unit tests passed; the pathplan and executable example
also passed race tests. The initial example at `88786e5` used labels outside the
closed operand-order vocabulary and failed before producing a result. The
published correction uses `layout_forward/layout_reverse` and adds a test that
actually runs the example. The library's earlier focused tests had passed; this
integration failure remains part of the record.

[Exact corrected-source CI](https://github.com/kimjooyoon/gooo-decision-runtime/actions/runs/37079292516)
checks formatting, vet, tests, race and existing frozen model replay. That CI's
model replay is a separate activity from this zero-model-call example.

[Research rationale and next steps](../../docs/agile-language-research-20261003.ko.md).
