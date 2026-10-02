# Fresh three-choice own models: training and actual Go kernels

The fresh common initializer, 10,739 source-bound states and every exported
feature/weight were prepared at `e0b256a9e561260672c5956834a9da10a39d11a0`.
The complete training-source CI run `36976206925` passed all eleven jobs,
including Linux feature/initializer regeneration. Offline MPS optimization
completed all three arms and **4,800 actual updates**, with no pretrained,
Laya, frozen teacher or previous student initialization.

Go auditor `08e15cfa03d82a02a35081fc0fe504d508a55690` then independently loaded
all nine exports. It reconciled all 4,800 update journal rows, all 600 epoch
receipts and joint calibration epoch/temperature choices. Actual Go calls were
**432 numerical parity predictions** and **18,018 separately counted warm and
allocation probes**, totaling 18,450. All nine models passed; invalid arity and
overflow probes changed no committed output and performed no kernel prediction.

## Calibration findings retained without discarding variants

| Arm | FP32 passing-set NLL | PTQ passing-set NLL | QAT passing-set NLL |
|---|---:|---:|---:|
| Uniform initial | 1.822441 | 1.837039 | 1.798077 |
| Passing-set initial | 1.819579 | 1.836282 | 1.802863 |
| Passing-set actual feedback | 1.811903 | 1.838565 | 1.798459 |

These values use the original 512-view calibration partition and all tied
passing masks. Actual-failure training improves this FP32 calibration objective,
but QAT uniform initial is slightly better than feedback QAT, and all PTQ
variants regress against their FP32 parents. All nine are retained. This does
not establish fewer SDK candidates, better development behavior or a new
default: the preregistered full SDK/native study is still pending.

## Measured costs and separate memory scopes

- Total six training-loop intervals: **27.561 seconds**. These intervals include
  optimizer work, calibration, synchronization and host-side receipt writes;
  they exclude some setup and final export work.
- Process CPU during those intervals: **60.09–76.77% of one core**, separately
  recorded per stage. GPU utilization and host CPU causal change were not measured.
- Sampled MPS tensor allocation peak: **32,118,528 bytes** (30.63 MiB).
- Sampled MPS driver allocation peak: **84,623,360 bytes** (80.70 MiB).
- Process lifetime RSS peak: **1,169,063,936 bytes** (1.09 GiB). This includes
  Torch, loaded corpus and transient optimizer/export state; it is not per-model
  inference RAM or a resettable per-arm measurement.
- Actual Go warmed prediction with full feature projection: **16.505–16.874 μs**
  across these nine probes, zero additional heap allocations per valid call.
- Maximum absolute Python-export/Go numerical error: **1.081e-6**, below 1e-5.

| Runtime quantity | FP32 | Packed ternary |
|---|---:|---:|
| Stored weight file | 74,624 bytes | 3,854 bytes |
| Resident decoded tensors | 74,624 bytes | 18,752 bytes |
| Additional matrix scales | 0 bytes | 8 bytes |
| Caller-owned request workspace | 3,200 bytes | 3,200 bytes |

The matrices pack five trits per byte, nominally **1.6 stored matrix bits**.
Biases, padding, scales, metadata, decoded runtime memory and process RSS are
separate. Ternary packing reduced storage; this kernel probe does not demonstrate
an inference speedup over FP32 or an overall codegen speedup.

## Complete compressed evidence

The training bundle retains all **642** original and supplementary files:
full Go feature matrix, exact row weights, fresh initializer, all stage updates,
all epoch receipts, all nine model pairs, all parity rows and both independent
audits. It contains 46,201,328 decoded bytes compressed to 5,758,737 bytes.
Every decoded member has already passed SHA-256, ZIP CRC, mode, inventory and
privacy checks. The earlier source/teacher bundle separately retains every
original native export and failed-prefix observation.

Next work preserves the full original protocol: 11,264 SDK sessions, calibration
policy selection, 640 actual Gooo generations and independent Go compilations,
and continued public byte verification. No language-completeness or native
deployment claim is inferred from this training/kernel phase.
