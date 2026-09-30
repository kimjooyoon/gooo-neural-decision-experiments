---
license: mit
language:
- en
- ko
tags:
- gooo
- typed-ir
- metaprogramming
- ternary
- experimental
- golang
---

# Gooo IR operator tiny v1

A 12,728-parameter experimental classifier for eight typed binary IR operations:
`add`, `subtract`, `multiply`, `less_than`, `less_equal`, `equal`, `and`, `or`.
It maps a bounded English/Korean instruction to probabilities. The Go bridge
validates operand types and identifiers before constructing a closed IR node
and expression. It does not generate arbitrary source or a complete program.

**Origin:** independently initialized MLP, not Laya weights or a Laya fine-tune.
Laya is the research baseline for future encoder/head experiments.
All training material is generated public synthetic templates. Code, data and
these independently trained weights are MIT licensed.

## Evaluation and completeness

2,048 examples: 1,536 training, 256 calibration, 256 test. Entire prompt templates
are kept in one split; the test has 32 templates with eight variable configurations
each, 128 English/128 Korean examples and 32 examples per operation. The effective
independent intent/template coverage is narrower than 256 rows. No test data was
used to fit weights, checkpoint selection, temperature or abstention thresholds.

| Variant | Correct / test rows | Test accuracy | Packed weights | Resident tensor arrays | Hot Go prediction |
| --- | ---: | ---: | ---: | ---: | ---: |
| FP32 | 256 / 256 | 100% | 50,912 B | 50,912 B | 7.86 us |
| PTQ ternary | 245 / 256 | 95.70% | 2,759 B | 12,896 B | 9.34 us |
| QAT ternary | 242 / 256 | 94.53% | 2,759 B | 12,896 B | 9.30 us |

Go matched all 96 Python parity rows within absolute tolerance `1e-4`.
The frozen Go audit contains label/language/template breakdowns, calibration
and selective acceptance, model/input/source hashes, typed IR checks and latency.
PTQ accepted 251/256 and was correct on 242 accepted rows. QAT accepted all 256
despite 14 errors; confidence is not a guarantee of semantic correctness.
Boolean conjunction/disjunction in held-out English templates caused most
ternary errors. Unseen operation families, free-form prompts, ambiguity,
malicious inputs and real developer workloads have not been evaluated.

All hot prediction variants used **0 B/op and 0 allocations/op** in this Apple
M4/Go 1.27 measurement. Ternary was approximately 19% slower than FP32.
Tensor totals exclude metadata, object headers, loader temporaries and, for
ternary, eight additional scale bytes. One reusable worker workspace is 1,248 B;
the prediction struct is 80 B. Cold CLI median peak RSS was about 5.2–5.4 MiB
over three processes per variant. FP32's first startup took 385 ms; its median
was 8.33 ms. PTQ/QAT medians were 3.35/3.24 ms. These small samples are not a
deployment latency guarantee.

## Quantization

The ternary alphabet has theoretical `log2(3) = 1.585` bits. Packing five trits
per byte uses **1.6 physical bits per matrix weight**, plus biases, scales and
metadata. The Go runtime decodes once to contiguous int8 matrices. Resident RAM
is not 1.58-bit packed storage. QAT used floating-point master weights with
straight-through ternary training; this is a small MLP, not a BitNet Transformer
or an 8-bit activation implementation. PTQ is derived from FP32; QAT is a
separate training arm. This pilot did not show QAT outperforming PTQ.

## Use with Go

Get the runtime from [the public GitHub repository](https://github.com/kimjooyoon/gooo-neural-decision-experiments).
Download a variant's `model.json` and sibling `weights.bin` from
`runs/pilot-mps-20260930-v1/models/`. Then run:

```sh
go run ./cmd/gooo-decision --model runs/pilot-mps-20260930-v1/models/fp32/model.json <<'JSON'
{"schema":"gooo/tiny-ir-decision-request/v1","text":"Operands: a, b. Combine both values by summation.","left":{"name":"a","type":"Int"},"right":{"name":"b","type":"Int"}}
JSON
```

The runtime rejects invalid UTF-8, empty instructions and instructions above
512 UTF-8 bytes. It never silently truncates input. Model loading rejects digest,
shape and encoding mismatches. A type mismatch or calibrated low confidence
produces an abstention. Deterministic ordinary Gooo compilation remains the
fallback path; this experimental bridge is not integrated into production Gooo.

## Training resources and provenance

Training used PyTorch 2.14 on Apple M4 MPS, ten GPU cores, 16 GiB unified RAM,
120 epochs per FP32/QAT arm and batch size 256. Measured loop time was 2.62 s
for FP32 and 1.02 s for QAT, including calibration and CPU scoring. The first
arm includes warm-up/cache costs; this is not evidence QAT training is faster.
Process-lifetime peak RSS was approximately 422–424 MiB. End-of-epoch sampled
MPS allocation/driver maxima were 2.06/53.18 MB. GPU utilization percentage,
continuous GPU memory peak and kernel-only timing were not measured.

This Hub bundle contains an explicit 17-file allowlist: synthetic data and
manifest, model contract/generator, frozen training source/preexecution, filtered
summary, machine card, Python parity, external Go audit, and three model pairs.
Tokens, private repository material, local filesystem paths and hardware
identifiers are excluded. SHA checks and pattern scans are integrity/hygiene
checks, not proof of privacy or quality outside this finite experiment.

Research sources: [Laya](https://huggingface.co/convaiinnovations/laya),
[BitNet b1.58 paper](https://arxiv.org/abs/2402.17764),
[PyTorch MPS](https://docs.pytorch.org/docs/main/notes/mps.html).
