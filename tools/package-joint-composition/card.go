package main

import "fmt"

// Filled after the same main proof verified for native execution.
const deployedCompiler = "363a3d8aa365c35dd634c241248b444de0050973"

func modelCard(predictions int) string {
	return fmt.Sprintf(`---
license: mit
language: [en, ko]
tags: [gooo, metaprogramming, ternary, from-scratch, go]
---
# Own Gooo joint path tiny v1

Six own random-initialized FP32/PTQ/QAT models compare independent decisions
(256/48/8, 12,728 parameters) with a joint four-mask head (512/24/4, 12,412).
No Laya, pretrained or earlier own-model weights. The two architectures share
the first-matrix weight count, not their hidden width or head. This is a matched
480-update experiment, not an isolated loss-only comparison or a text generator.

Gooo binds original source and a typed plan, preserves complete Korean/English
intent, and asks the model to rank four legal compiler-owned paths. The joint
head represents correlated choices with one actual initial call. Actual failed
tests may rerank remaining paths. A sole remaining path needs no model call.
Absent model or unsupported representation uses deterministic continuation.

Calibration selected independent FP32. On 384 development function views,
joint FP32 used 777 predictions versus 1,447 for independent FP32, but needed
455 extra assembly attempts versus 448. Offline needed 520. All seven policies
completed their finite 16-case contracts by four candidates. That finite result
does not establish broader language understanding or universal correctness.

Warm joint FP32 inference measured about 10.7 microseconds with zero per-call
heap allocations. Joint ternary files occupy 2,590 bytes; decoded int8 tensors
12,496 bytes plus 8 scale bytes and a 2,160-byte workspace. Five trits per byte
give 1.6 storage bits per matrix weight; resident arithmetic is decoded int8.

Actual adopted Gooo main: 192 native calls, %d predictions, 192 independently
compiled/executed Go outputs and 3,072 ordered function invocations. Selected
independent FP32, independent FP32 reference, joint FP32 and offline policies
each replay 48 bilingual function views. Selected/reference duplicate one model.

Use SDK v0.2.12 and the adopted compiler with an explicit model:

    gooo body-codegen --json --path-plan plan.json --path-model joint/models/fp32/model.json --path-step-attempts 1 --path-feedback-rounds 3 --path-feedback-unfixed --activity ChoosePath source.gooo

Joint v1 accepts exactly two binary decisions. Both source contexts and complete
intent parts are preserved. It never truncates input to gain model acceptance.
Omit --path-model for deterministic continuation. No default model was promoted.

protocol.md was frozen before fixtures, targets or training. Its scale formula
was found constant before any new fixtures; prefixture-amendment.md preserves
that correction separately. Raw curriculum, SDK/native captures, actual Go
values and exact source inventories are in raw-evidence.zip. All six exports,
negative comparisons, calibration choice and PROV-O provenance remain public.
Publication uses a fixed synthetic allowlist and scans credentials/host paths.
See results.md for full denominators, timing/CPU/RSS scope and ambiguity limits.

Source: https://github.com/kimjooyoon/gooo-neural-decision-experiments
SDK: https://github.com/kimjooyoon/gooo-decision-runtime/releases/tag/v0.2.12-experimental
Compiler: https://github.com/kimjooyoon/meta-ontology-go/commit/%s
`, predictions, deployedCompiler)
}
