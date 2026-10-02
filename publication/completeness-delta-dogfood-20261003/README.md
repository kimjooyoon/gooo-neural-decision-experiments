# Actual own-model use of the Gooo completeness comparator

This is one integration observation on 2026-10-03. The clean compiler source is
[`e6506264bd711780b50559df572e32e29e28a382`](https://github.com/kimjooyoon/meta-ontology-go/commit/e6506264bd711780b50559df572e32e29e28a382),
with Go 1.27.1 on darwin/arm64. The explicit model is the public compact FP32
shared judge at HF revision `985999a89caba6a31cc7147f66ba29a5ce76a1d9`.

The compiler made **3 actual predictions**, generated the selected body, built
and ran it twice. All **24 supplied finite expectations** matched. Each
`completeness-delta` call then performed zero model calls and zero executions.

| Comparison | Observations |
| --- | --- |
| Generation → runtime | Exact parent continuation; 26 retained axes, 8 added axes, 2 UNKNOWN→observed transitions |
| Runtime → itself | Same registered scope; unchanged comparable counts |
| Prior frozen generation → fresh generation | Same registered scope; all 26 states unchanged, 16 comparable count deltas equal zero |

`permission_boundary` remains the first unresolved claim after execution.
Generation/runtime use different profiles, so that comparison preserves the new
observations with null numeric deltas. It does not invent an overall percentage.
The old generation and its original identities are included unchanged.

## Inspect and reproduce

The manifest binds nine complete input/output files. From this directory, with
the compiler built from the pinned source:

```sh
gooo completeness-delta --before model-generation.json --after model-runtime.json
gooo completeness-delta --before model-runtime.json --after model-runtime.json
gooo completeness-delta --before prior-generation.json --after model-generation.json
```

These commands reproduce the stored comparisons without loading a model. For a
fresh model invocation, supply the compact model's local `model.json`:

```sh
gooo body-codegen --json --activity ChoosePath --path-plan plan.json \
  --path-model /path/to/compact/fp32/model.json \
  --path-step-attempts 1 --path-feedback-rounds 7 --path-feedback-unfixed \
  input.gooo > fresh-generation.json
gooo body-execute --source input.gooo --path-plan plan.json \
  --generation fresh-generation.json --cases runtime-cases.json \
  --go-bin "$(command -v go)" > fresh-runtime.json
```

Fresh invocations have new resource observations and receipt digests. This
integration sample is separate from the frozen 96-generation compact benchmark
and adds no quality or performance claim. Public checks inspect the complete JSON
including decoded base64 parent receipts for private paths and credential patterns.
