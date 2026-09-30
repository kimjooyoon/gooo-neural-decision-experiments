# Independent Go inference audit

Decision: **PASS**. The Go runtime matched all 96 saved Python parity vectors
within absolute tolerance `1e-4`; all 256 frozen test examples were observed
for each model. The audit used the already exported pilot bundles and made no
training or provider calls.

The machine-readable source of record is [go-audit.json](go-audit.json),
SHA-256 `f28434deee5356610848c5d5906411abed7d388ee28828c82b39a8bf90dadc9d`.
It binds dataset SHA-256
`af0a637320a8256cd6ebcdb3486a6885f2766441c90f0cb154c11599f3c8ca3d`, parity
SHA-256 `84119298650830740f4e6d4a3159b1163cef4c24f2e6f49a6dbdbfb3d1e6ebf6`,
the model metadata and weights hashes for each variant, and the Go runtime and
audit source hashes. Each model has 32 parity rows. Feature vectors matched
exactly; maximum absolute logit error was `3.82e-6`, and maximum probability
error was `8.17e-7`.

## Held-out test results

| Variant | Correct / planned | Accuracy | NLL | Brier | ECE (10 bins) | Accepted / correct accepted |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| FP32 | 256 / 256 | 100.00% | 0.0001343 | 0.00000512 | 0.0001330 | 256 / 256 |
| PTQ ternary | 245 / 256 | 95.70% | 0.11344 | 0.06919 | 0.01096 | 251 / 242 |
| QAT ternary | 242 / 256 | 94.53% | 0.10744 | 0.07456 | 0.03408 | 256 / 242 |

Test rows were balanced: 128 English and 128 Korean, 32 examples per label,
and 32 held-out templates with 8 examples each. FP32 scored 128/128 in each
language and 8/8 on every template. PTQ scored 118/128 English and 127/128
Korean; 29 templates scored 8/8 and its weakest scored 2/8. QAT scored
114/128 English and 128/128 Korean; 29 templates scored 8/8 and its two
weakest scored 2/8. The JSON report has each label and template row. The main
ternary errors were `and` and `or`: PTQ scored 26/32 and 28/32; QAT scored
20/32 and 30/32.

The confidence thresholds were fitted on calibration data. PTQ abstained on
5 test rows and was correct on 242 of its 251 accepted rows. FP32 and QAT
accepted all 256. These are test-split measurements; the test split was not
used to tune the threshold.

The typed bridge constructed and safely assembled a gold expression for all
256 rows in every variant. FP32 and QAT emitted 256 typed decisions. PTQ
emitted 251 and abstained at low confidence on 5. No type-mismatch abstention
occurred in these held-out rows; dedicated unit tests exercise that guard.
Expression assembly uses a fixed operation-to-operator table and validates
identifiers and operand types before producing text.

## Runtime and memory

Measurements below use the exported bundles on Apple M4 with Go 1.27. Hot
prediction was measured in-process after model loading. Cold CLI wall time
covers process start, model load, one request, and JSON output; each value is
one of three preserved child-process runs. Cold RSS is each child’s
`getrusage` peak RSS.

| Variant | Hot prediction | Allocations | Resident tensor bytes | Workspace / output | Cold CLI median | Cold CLI peak RSS median |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| FP32 | 7,860 ns/op | 0 B/op, 0 allocs/op | 50,912 | 1,248 / 80 B | 8.33 ms | 5.36 MiB |
| PTQ ternary | 9,336 ns/op | 0 B/op, 0 allocs/op | 12,896 | 1,248 / 80 B | 3.35 ms | 5.23 MiB |
| QAT ternary | 9,297 ns/op | 0 B/op, 0 allocs/op | 12,896 | 1,248 / 80 B | 3.24 ms | 5.33 MiB |

Ternary bundles use 12,672 resident int8 matrix bytes and 224 float32 bias
bytes. Their packed files are 2,759 bytes. FP32 matrices and biases occupy
50,688 and 224 bytes. Tensor totals exclude model metadata, Go object overhead,
and the temporary packed input used while loading; each ternary model also has
two float32 scale values (8 bytes) outside the tensor-array total. Each 80-byte
prediction struct contains 64 bytes of logits and probability arrays. Ternary storage is about
four times smaller than FP32 tensor storage, while this CPU run measured
roughly 19% slower hot prediction; these results do not show a speed benefit
from ternary weights.

All nine final cold CLI runs exited successfully and their raw request,
stdout, and stderr captures are under
[`go-audit-cli-cold-v3/`](go-audit-cli-cold-v3/). The FP32 process wall times
were 385.26, 8.33, and 3.96 ms; the first process was much slower than the
later two. PTQ times were 3.31, 3.40, and 3.35 ms; QAT times were 3.24, 3.14,
and 3.29 ms. The earlier v1 and v2 audit reports and captures remain preserved
as separate diagnostics; no run is omitted from the v3 record.
