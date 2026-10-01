# Retained Gooo construction cost: preregistration

This protocol freezes before execution. It uses the real native compiler and the
new `gooo-body-worker`, with identical clean source revisions and SDK 0.2.7.
It is a construction cost pilot, not a new language accuracy benchmark.

## Fixed workload and controls

Reuse the six contradictory compound views: three authored structural templates,
one existing numeric configuration, mask-zero intentions, Korean and English.
Run them once forward and once in reverse. There are 12 valid requests per arm.
These repeated/translated views are not independent new experiments.

Use five fixed arms: disconnected, parent FP32, feedback FP32, feedback PTQ and
feedback QAT. Metadata/weights remain frozen. Ternary storage is 1.6 bits per
weight with decoded int8 arrays; packed compute is not used. Each model-backed
request uses step 1, max attempts 4, feedback rounds 3 and the explicit varying
coordinate optimization. CI hint status is UNKNOWN and its source is the measured
native revision. No post-CI relabeling, training, GPU or upstream Laya HTTP calls.

For each arm compare:

1. Twelve fresh native CLI processes, model loaded after each source binding.
2. One retained-model process, sequential requests; await each result before
   sending the next. Setup precedes request clocks and is measured separately.
3. One retained-model process with four workers, with the producer and consumer
   active together. Queue time is part of per-response latency.

Order these modes fresh/retained-1/retained-4 for even arm indices, reversing mode
order for odd indices. Before each retained batch submit one invalid source;
require a rejected receipt with zero predictions, followed by all valid requests.
Expected totals: 180 verified constructions, 10 source rejections and 70 native
processes. Four model-backed arms make predictions; the disconnected arm does not.

## Evidence and measurements

Save preregistration tuple and binary hashes before execution. Save every original
fresh JSON response or worker NDJSON stream and its timing/resource sidecar before
inspection. Startup is spawn to constructor-record arrival. Constructor setup is
local model load/identity work. Each request freshly prepares and binds source and
plan; no preparation cache, intent cache or candidate state is retained.

Record full process wall, user/system CPU and max RSS; record retained enqueue to
response latency by sequence, native plan/binding/search/emission times and model
loads. Whole worker metrics include startup and the rejected request. Fresh
response clocks include process startup/model load; retained sequential clocks
exclude one startup/load. Parallel response clocks include queueing and should not
be presented as sequential latency. Compare whole-process throughput separately.
The model resident tensor count is separate from process max RSS. These child CPU
measurements are not host CPU growth or GPU utilization. Controller serialization,
memory and CPU are outside the child measurements.

Require exact source, document, suite, original CI hint and metadata/weight
identity. Reconcile per-request progress/feedback/failure chains, actual native
function AST versus selected typed program and finite pass denominators. Require
same candidate sequence, selected body and native actuals across all three modes
for the same arm/view. Source rejection and earlier request results must not
contaminate subsequent construction. Partial finite results are valid outcomes;
first-shot accuracy is not an acceptance metric.

Deduplicate emitted Go source only for actual execution: run every distinct body
against the independently authored state oracle on the union of declared and
separate inputs, including int64 extremes. Keep the actual observations and later
audit them offline without predictions or subprocess execution. Publish negative
or increased costs alongside decreases. This fixed small workload does not support
a causal/generalized speedup, all-input correctness or broad language judgment.

## Operational scope

The native worker's race/lifecycle tests cover cancellation, blocked input/output,
bounded backpressure, source mismatch and concurrent request isolation. A native
CI modernizer failure, if observed, is retained separately; canonical source CI
and deployment proof remain mandatory. No human review or Guardian is introduced.
Publish an additive GitHub/Hugging Face appendix; previous captures, model card,
weights and raw UNKNOWN CI hints remain immutable.
