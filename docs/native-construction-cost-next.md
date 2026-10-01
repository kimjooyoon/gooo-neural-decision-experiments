# Next local construction-cost study

This is a plan, not an implemented native service or measured speedup. Native
main 6f69eb116336b4728db6f94a992130ea56003a48 already supports explicit varying-coordinate
feedback. Its six-case bilingual pilot observes roughly 0.330–0.352 ms median
bounded search inside roughly 5.9–6.1 ms whole compiler processes. Tiny prediction
count is one cost dimension; source preparation, emission and process startup
still need independent measurement. First-shot accuracy is not acceptance.

## Controlled arms

| Arm | Construction boundary | What can be compared |
|---|---|---|
| Fresh native process | Current CLI, model loaded per request | Actual whole compiler and emitted-Go baseline |
| Retained model, fresh source/plan | Proposed bounded native worker | Same native contract and per-request source/type/replay evidence; startup/model-load amortization measured separately |
| Prepared Go SDK session | Existing in-process SDK | Candidate construction/receipt costs only; cannot substitute for full native emission results |
| Disconnected native | Deterministic fallback without a model | Same source, typed paths, partial-case denominator and actual emitted-Go checks |

Use models/weights unchanged and publish the new work as a separate appendix.
Order repeated arms in both directions and retain cold/warm results explicitly;
do not interpret six observations as an untouched language benchmark. Include
Korean/English paraphrases, ambiguous intentions, contradictory and sparse tests,
referenced assignments, nested conditions and scheduling-equivalent expressions.
Distinct new structural templates require authored independent state oracles;
translated contracts and repeated policy arms do not count as new experiments.

## Evidence and bounded state

Source bytes, original typed plan and finite suite bind each request. Any retained
model identity must match metadata/weight digests. A preparation cache would need
source SHA, plan SHA, suite SHA, compiler revision and explicit mode; no result
may inherit another request's intent, progress, failure or CI hint. Mutable session
state is request-owned. Model storage/decoded arrays are immutable, shared only
through supported runtime contracts. Actual native memory/worker bounds must be
measured before a low-RAM or parallel-throughput claim.

Keep emitted function AST reconciliation, type/replay checks, no-source-write
evidence and deduplicated actual Go execution against separate integer-state
oracles. Report functional partial completeness, intention ambiguity, committed
candidate count, initial/feedback prediction count, cancellation/decline reasons,
source preparation/emission times, cold/warm whole wall/CPU and max RSS separately.
An increase in any dimension is retained. One-core process CPU is not host CPU
growth; packed storage size is not decoded resident memory or packed compute.

Exercise cancellations before inference, during inference and between committed
batches; aborted candidates cannot consume committed-mask counts or contaminate
the next request. Concurrent workers need isolated receipts, bounded queues and
bounded shutdown. Reproduce all records offline without predictions or execution.
CI checks source changes and replay evidence; local candidate feedback consumes
finite failures immediately. PASS/FAIL/UNKNOWN CI context stays source-bound and
unauthenticated rather than becoming an authority input.

## Model scope

Own tiny models choose within declared Gooo typed coordinates using bounded
Korean/English context. New work should expose construction completeness and
iteration cost rather than rely on one-shot arbitrary text generation. Existing
broader checkpoint regressions, raw UNKNOWN hints and negative certificate results
remain frozen. Model tuning and any GPU work require a separately preregistered
curriculum/compute record; none is part of this deployment pilot.
