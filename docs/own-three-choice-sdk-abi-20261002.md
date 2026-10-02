# Three-choice SDK implementation phase

The frozen [study protocol](own-three-choice-completeness-preregistration-20261002.md)
was published at `d74a8a455ceed5949fcbad482375405b4704dc9a` before collection
and optimization. Protocol SHA256:
`5d840f7b3379339d684b85b7896a895538bd1a030395a096af3e52c5fc781cd3`.

## Implemented interfaces

The separate `jointdecision.ThreeModel` ABI validates 768/24/8 dimensions,
eight complete mask labels, metadata up to 64 KiB and weights up to 128 KiB.
The existing four-label ABI retains its 64 KiB weight bound and prediction math.
Three complete canonical source-v3 parts retain their declared order and full
intent. Feature projection concatenates three 256-value vectors at `1/sqrt(3)`.

`PreparedPlan.NewThreeSession`, `Session.ReconsiderThree` and
`PreparedPlan.SearchThreeFeedbackBatches` rank eight complete alternatives and
reconsider actual committed failures. Each reconsideration makes one prediction
when more than one mask remains and the full context fits; the final sole mask
makes none. Seeded initial sampling is reproducible and keeps seed text outside
the input. Candidate construction remains bounded typed Gooo body assembly.

A disconnected model uses deterministic continuation. Unsupported arity records
every original input and makes zero predictions, including a fourth choice.
Overflow preserves full attempted input and individual part hashes, consumes
one bounded feedback round and continues with the unchanged frontier. Models
are immutable; request workspaces and sessions are independent. Nonblocking
locks return busy immediately; canceled rescoring preserves committed scores.

## Measured layout and verification scope

Contract tests check 74,624 FP32 weight bytes, 3,854 five-trit packed bytes,
18,624 decoded int8 matrix bytes, 128 FP32 bias bytes and eight separate scale
bytes. The fixed request workspace is 3,200 bytes. The shared Session struct's
joint-score array grows by 32 bytes (four additional float64 scores); these
figures describe arrays/tensors, not whole-process memory.

Valid projection and prediction each allocate zero heap objects in warmed tests.
Nonzero test matrices exercise every source channel, row-major strides, both
ternary scales, biases, temperature and all eight probabilities against separate
float64 arithmetic. Tests cover concurrent requests, invalid/numerically
overflowing predictions with atomic outputs, malformed metadata/weights,
unsupported arity, finite TDD calls, replay sampling, feedback overflow, pinned
model identity, cancel/busy behavior and complete candidate traversal.

Initial authored tests failed because an input declaration omitted `name=input`
and an expected value miscomputed three reversed subtractions. The corrected
ordinary arithmetic is `9-(3-(1-input)) = 7-input`; the tests now use the two
independently derived values 4 and 3 for inputs 3 and 4. Full SDK checks also
initially rejected altered extracted bytes under the old provenance manifest;
the updated source inventory is committed before extraction into the SDK.
No manifest check is bypassed.

This phase uses controlled test weights. It does not establish trained model
quality, natural-language completeness, native deployment or time savings.
The declared curriculum, actual frozen-teacher collection, fresh optimization,
student/native study and public model byte verification remain separate work.

The subsequent [native adoption phase](own-three-choice-native-adoption-20261002.md)
is now verified on compiler main, separately from these SDK-only measurements.
