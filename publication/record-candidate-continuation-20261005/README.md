# Keep generating after an invalid record combination

2026-10-05. One authored unused-local example compares deterministic order and
the unchanged public shared-field QAT model with budgets4/8. Four generation
commands and four saved replay commands belong to one bounded observation.

## The language usability problem

Two individually valid field alternatives can jointly remove every use of a
saved local. The previous compiler stopped the complete search at that candidate.
`before-failure.json` retains the original compiled failure from sourcec330ee6d.
The baseline and each individual alternative had passed source preflight.

[Compiler PR1237](https://github.com/kimjooyoon/meta-ontology-go/pull/1237) records
that combination as `TYPECHECK_FAILED`, consumes its budget slot and continues.
Cases have not run for a type-rejected candidate, so its case/field denominators
remain zero. A previously evaluated partial implementation remains available.
Source or individual-alternative errors still fail preflight. Other generator
errors and cancellation stop the call.

The raw commands below used clean source
`664d5bab47af3faf95b8a9774941b807f15a5dde`, Go1.27.1 and immutableSDK24.
The follow-up2823a03c adopts Go1.27's typed error extraction API; its focused
regressions also passed. Merge and installation have their own source receipts.

## Actual code generation and native execution

| Candidate order | Budget | Attempted | Type rejected | Cases evaluated | Selection fields | Native named outputs |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Deterministic |4 |4 |1 |3 candidates |12/15 |6/14 |
| Deterministic |8 |7 |1 |6 candidates |15/15 |14/14 |
| Public QAT |4 |4 |1 |3 candidates |15/15 |14/14 |
| Public QAT |8 |4 |1 |3 candidates |15/15 |14/14 |

The partial run retains actual values and gaps. Its native record fields are
17/21; named outputs require their whole value to match. Complete runs retain
21/21 native record fields. Selection and native denominators are separate.
The Go completeness reader recounts each field from its actual/expected values
and now reports attempted, evaluated and type-rejected candidates separately.

The QAT ranking is5,4,7,6,1,0,3,2. Its third candidate,mask7, fails type checking;
mask6 then completes the supplied functionality. Deterministic order encounters
the same issue atmask3. Every model generation calls the unchanged public model
once. Native execution and all four saved replays make zero new predictions.
Each native response includes two fresh executions. No model fitting occurred.

## Cost and retained evidence

QAT prediction took36.708µs and20.084µs. The two complete generation/build/run
commands took319.90ms and328.11ms. Their command CPU ratios were81.5% and78.9%
of one core; maximum single-process RSS was81.28MiB and82.78MiB. These include
compiler/build/native work. Host CPU increase and GPU utilization are unmeasured.

All four commands ran in a fixed order. The first deterministic run took724.30ms,
the next522.15ms; both are retained. This ordered small observation does not
estimate a general speedup. Eight full JSON responses, four source files,
compiler identity, summary, original failure and regression logs are included.
The observation folder remains under1MiB of logical source/data before a later
installation follow-up. Check `SHA256SUMS` for the published files.

## Reproduce

Build a clean compiler checkout containing PR1237 with Go1.27.1. Download the
two QAT model files from HF revision48cb8c508d5cd2ff0066eade37df43cc8ea77e3a of
[the public shared-field model](https://huggingface.co/asketeddy/gooo-record-shared-field-tiny-v1).
From this research checkout on macOS or Linux:

```sh
go run ./publication/record-candidate-continuation-20261005/observe.go \
  --checkout /path/to/meta-ontology-go --compiler /path/to/gooo \
  --model /path/to/models/qat_ternary/model.json --out /tmp/gooo-continuation-study
go run ./cmd/record-completeness \
  --input /tmp/gooo-continuation-study/qat_ternary-b8-native.json
```

`observe.go` is a parameterized reproduction of the measured procedure. Output
must be a new directory. It binds the public frozen weight digest and records
the actual compiler source. The origin-aware model-input contract is a separate
[source representation study](../record-value-origins-20261005); new origin
weights still require balanced-goal training and native evaluation.
