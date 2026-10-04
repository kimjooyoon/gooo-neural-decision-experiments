# Frozen ternary field-assembly pilot

Two actual native graph constructions use clean compiler
`cb316851c8701577fdb5d0d426651e68de3a4b88`, Go 1.27.1 and SDK v0.2.21.
The graph and seven runtime cases are supplied beside the observations. The
existing own-model PTQ and QAT weights are unchanged; no training is included.

| Profile | Resident tensor bytes | Initial prediction, one call | Attempted candidates | Selection fields | Compiled named outputs |
| --- | ---: | ---: | ---: | ---: | ---: |
| PTQ ternary | 18,752 | 39.708µs | 7 | 15/15 | 14/14 |
| QAT ternary | 18,752 | 35.834µs | 8 | 15/15 | 14/14 |

The FP32 cohort uses 74,624 resident tensor bytes. Ternary tensors here use
**2-bit packing**; 1.58 bits is the theoretical information content of three
states. The model tensor reduction is about 74.9%. Prediction workspaces, Go
runtime, source analysis, native build and process memory have separate costs.
There is one request per profile; these timings and candidate counts describe
this pilot, with cross-source model quality and broad speed improvement still
unmeasured. The field-ordinal context is the same frozen integer-model transfer
documented in the main field-assembly study. Runtime/replay makes zero model calls.

Metadata/weight identities:

- PTQ: `2bc62124e2ca72e2effe1e770842b7137f8968e2cd1238f139b7d1d503beb1bc` /
  `942e11299fecb9f2c399024375ac5d7aa114df2e3d2070a625872d391542b82d`.
- QAT: `fc5446ea7ca31ba2eb4ff29957cec5614df9820cfd110bc1f51a79f996325c43` /
  `2d75a95d03f060177c1f8feff7fde582b4dc70cd3000074a51b27afbc74b3e3a`.

Read the actual field differences and named-output counts with the research Go
`cmd/record-completeness` command. All three frozen profiles are in the public
research repository's `models/own-three-feedback-v1/set-feedback/models`.

The field observer also identifies these exact FP32, PTQ and QAT metadata/weight
pairs, reports the profile and resident tensor bytes, and recomputes the finite
counts from actual values. For example:

```sh
go run ./cmd/record-field-observe \
  --verify-json publication/record-field-assembly-20261004/quantized-pilot/field-ptq_ternary.json \
  --compiler-source cb316851c8701577fdb5d0d426651e68de3a4b88
```

The research CI constructs FP32, deterministic, PTQ and QAT bodies at all four
budgets and executes saved compositions with zero new predictions. Its fresh
Linux captures are separate from the two local pilot timings above.
