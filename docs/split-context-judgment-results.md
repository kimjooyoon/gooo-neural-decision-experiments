# Separate Gooo context and bilingual intent: measured regression

이번 실험의 목표는 첫 응답의 정답률보다, Gooo 구조와 한글·영어 의도를
판단한 뒤 유한한 테스트 계약으로 조립을 이어가는 비용을 측정하는 것입니다.
입력 채널을 분리했지만 추가 경로 시도는 늘었습니다. 모델과 실패 기록을
실험 자료로 공개하며 기본 모델로 채택하지 않습니다.

## Input contract and actual training

The preregistered `split_context_intent_ngrams_v2` ABI keeps 256 fixed feature
values: 64 context values and 192 intent values in four position buckets. The
last literal `intent: ` separates the channels; without it, all input is intent.
Each active channel is normalized independently and scaled by the fixed square
root of the number of active channels. Changing one active channel leaves the
other bit-identical. ASCII folding and byte ngrams remain bounded conveniences,
not a Korean/English parser or ontology reasoning. Empty, invalid UTF-8 and
over-512-byte inputs are rejected, without truncation.

Source `93e3c7f2c394bceecd34db9c428044461fb59c9a` was published before training.
The matched positioned-v1 and split-v2 arms start from the same random state and
seed, with the same optimizer and finite soft targets, zero consistency penalty.
No old v1 weights are reinterpreted as v2. Existing 1,040 English/Korean pairs
reuse 640 Gooo program groups: 800 train, 80 calibration, 160 development pairs.
Program/template groups are disjoint across splits, but this repeatedly observed
development set is not an untouched benchmark. New independent intentions: zero.

MPS performed **560 actual optimizer steps**, 140 FP32 and 140 QAT per arm. PTQ
exports each FP32 model without training. Four training loops totaled 1.267
seconds; v1 FP32 ran first and includes startup. This fixed order and single run
do not establish comparative throughput. Sampled MPS allocated peak was
2,033,920 bytes and driver peak 53,166,080 bytes. Process lifetime peak RSS reached
527,450,112 bytes, about 503 MiB. Host CPU/GPU utilization was not measured.
Checkpoint and temperature selection use calibration only.

## Judgment and continuation

Go evaluated 160 bilingual pairs / 320 views per cell. The original own-feedback
parent and disconnected deterministic arms are additional descriptive references;
the controlled comparison below is between identically initialized v1 and v2.

| Model | Agreement /160 | Same wrong intent /160 | Complete before /320 | Additional legal attempts | Prediction median µs |
|---|---:|---:|---:|---:|---:|
| v1 FP32 | 91 | 0 | 251 | 69 | 11.042 |
| v2 FP32 | 57 | 0 | 217 | 103 | 11.791 |
| v1 PTQ ternary | 116 | 26 | 224 | 96 | 12.667 |
| v2 PTQ ternary | 137 | 60 | 177 | 143 | 13.042 |
| v1 QAT ternary | 88 | 1 | 246 | 74 | 12.459 |
| v2 QAT ternary | 59 | 7 | 205 | 115 | 12.875 |

Separating channels increases attempts in every matched variant here. Greater
agreement or lower bilingual divergence can still mean both languages choose the
same wrong intended body: v2 PTQ illustrates this. Channel capacity, normalization
and the objective are possible follow-up variables; this comparison does not
identify a cause. Raw timing is a single local observation, not causal speedup.

After the separately authored full contract, every cell reaches 320/320 complete
views and 3,840/3,840 finite cases with at most one additional legal path per view
and zero additional model calls. This is exhaustive evaluation of two existing
typed-arena paths. The tests and legal paths supply the finite completion; the
model supplies a ranking. This does not measure general intent correctness or
arbitrary body synthesis. These Go judgment audits execute no native compiler
and no emitted-Go processes.

The judgment capture has 2,880 actual predictions across nine model cells.
Separate full-curriculum audits have 5,856 each, including numerical parity and
original positioned-random parent comparisons. That parent differs from the
own-feedback parent in the bilingual judgment capture. Total local predictions:
14,592, excluding unit tests and CI. Both 96-row Go/Python parity fixtures pass,
maximum observed errors 7.45e-8 for v1 and 6.90e-8 for v2. The fixed toy feature
fixture also verifies Go/Python channel encoding.

FP32 tensors occupy 50,912 bytes. Each ternary export has 2,759 packed disk bytes,
12,896 decoded int8 tensor bytes, 8 matrix-scale bytes and a 1,248-byte workspace.
Five trits per byte encode 1.6 disk bits per matrix weight; theoretical log2(3)
is about 1.585. This is decoded int8 inference, not packed compute or total RAM.
The valid Go feature path allocates zero heap objects in its allocation test.

## Actual compiler compatibility probe

One actual read-only invocation of clean compiler main
`1e01c96c54f2f8dd43334b8f580af93ffaea24df`, SDK `v0.2.8-experimental`, supplied
the v2 FP32 metadata to the existing source-bound body-codegen path. Source
binding passes, then model loading returns exit 1 / `FAIL_CLOSED`:
`feature version is outside the closed model contract`.

The original capture and outcome are retained. Observed model predictions: zero;
repository writes: zero; generated Go: none. This is expected compatibility
rejection before inference, not successful v2 native adoption. The existing
compiler passes caller intent to its initial ranking path; automatic construction
of the new context channel from typed IR remains future integration work.

## Reproduce and use

Source, raw predictions, tests and offline training are in the
[public research repository](https://github.com/kimjooyoon/gooo-neural-decision-experiments).
The six weights and selected immutable evidence form an experimental appendix
in the [public model repository](https://huggingface.co/asketeddy/gooo-feedback-path-tiny-v1).
Existing core weights and model card retain their meaning. These are independently
trained own tiny models; upstream Laya weights are not used in this training.

From this research checkout, with Go 1.27.1:

```sh
go run ./tools/audit-feedback-model --models runs/split-context-judgment-mps-20261001/v2/models --output /tmp/split-v2-parity.json
go run ./tools/audit-bilingual-judgment --models runs/split-context-judgment-mps-20261001 --arms v1,v2 --out /tmp/split-context-fresh-judgment
```

The output directory must be fresh. Runtime, evaluation and publication tooling
use Go; Python is offline training/export only. The native default remains the
existing optional-model policy, including deterministic disconnected execution.
New ABI support must be explicitly adopted before native v2 use.
