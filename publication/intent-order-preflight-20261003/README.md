# Keeping a small trace of instruction order

2026-10-03. The current V4 model counts short text fragments. Two instructions
can contain the same fragments while requesting different operation orders.
This preflight adds an experimental **64-counter directed clause sketch** beside
the existing representation. It uses **128 bytes** per intention and changes
no existing weights, feature contract or compiler route.

## Observations

Four authored arithmetic order pairs — add/multiply, subtract/multiply,
negate/add and square/add — have Korean/English views and four wrapper forms.
That gives **32 permutation pairs**, with 64 real frozen-model predictions.
All use a fixed synthetic valid source header to isolate intention features.
The references list Gooo body text and arithmetic outputs at `[-2,0,3]`.

| Observation | Pairs |
| --- | ---: |
| Existing V4 fragment features are identical across the two orders | 32/32 |
| Frozen V4 model returns identical predictions across the two orders | 32/32 |
| Existing V3 positioned features distinguish the two orders | 32/32 |
| Experimental clause counters distinguish the two orders | 32/32 |

The frozen model is compact bag-original FP32, with the same metadata and weight
hashes as the observation-loop pilots. Actual prediction time ranges from
10.417 to 31.417 µs, with a median of 12.209 µs across 64 calls. This is inference
on controlled feature inputs. **No new feature model has been trained or called,
and no native Gooo program is compiled or executed in this preflight.**

The feature kernel's three 10,000-operation measurements on Apple M4 / Go 1.27.1
are **618.6, 402.3 and 372.3 ns/op**, each with **0 B/op and 0 heap allocations**.
The output array is 128 bytes; a temporary array and stack frames are additional.
This measures feature construction, with no full-codegen or host CPU/RAM claim.
`benchmark.txt` retains every measurement and the executed feature checks.

## Why a small language-specific step is useful

The old representation gives its model exactly equal inputs for these orders.
Changing training examples alone cannot recover information that input encoding
has removed. The auxiliary sketch preserves an additional clue: which clause
followed which. It could support a small bounded judgment when Gooo already
provides legal ordered alternatives and tests of the assembled body.
V3 also separates these pairs. The next question is whether retaining V4's full
fragment channel alongside a small order channel improves completed bodies and
wrapper tolerance at an acceptable cost; this preflight has no answer yet.

The preflight supports trying this feature in a small training comparison.
It does not provide a new correctness score. The original source facts and full
instruction still need to travel with any learned decision. Runtime observations
remain the way to measure whether the selected assembly performs the requested
finite behavior.

## A remaining counterexample

`A B A C A D` and `A C A B A D` produce the same directed edge multiset, including
the beginning and ending edges. Both still have identical sketches. This
explicit negative control is in `report.json` and a regression test. Hash buckets
create further possible collisions; punctuation and wording affect segmentation.
The feature has no learned semantic vocabulary and no general ordering guarantee.

One possible next comparison keeps the old 256 features and appends normalized
clause counters. That changes the model input contract and requires new weights.
A small fixed contrast set should measure completed body cases and extra search,
including repeated-operation counterexamples. Broad language generalization is
a separate question. No such training result is claimed here.

## Source and reproduction

Collector and feature source: `990b0d8c8e9d4b4b3f5638e1f4f1acb937ad496b`.
The [pre-collection definition](../../tools/audit-intent-order/README.md) fixes
the examples, comparisons and limits. `records.json` preserves both intentions,
reference bodies, arithmetic outputs, feature counters, input hashes and actual
model predictions. `report.json` binds the source and model identities.

```sh
go run ./tools/audit-intent-order \
  --source-revision 990b0d8c8e9d4b4b3f5638e1f4f1acb937ad496b \
  --output publication/a-new-order-preflight
go test ./internal/decision -run '^TestOrderSketch' \
  -bench '^BenchmarkSemanticIntentOrderSketch$' -benchmem -benchtime=10000x -count=3
```

The old model is dogfooded here to expose an actual encoding limitation. The new
128-byte sketch is an experimental next input, with zero optimizer updates and
zero new-feature model predictions. Checksums cover the complete publication.
