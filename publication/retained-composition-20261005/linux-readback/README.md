# Independent Linux result readback

The `retained-composition-replay` job in
[research CI37214051407](https://github.com/kimjooyoon/gooo-neural-decision-experiments/actions/runs/37214051407)
passed at research source `83ed57a0d4f2a9cbd58d8e76f859075605b4d10d`.
It builds clean compiler `38962397144d4418b75f571471225ff02d712617` with Go1.27.1
and runs deterministic, frozen FP32, PTQ and QAT constructions plus saved replays.

We downloaded the job's `retained-composition-replay` artifact and independently
checked its eight response files with the Go observer. `verification.jsonl`
contains those recomputed reports. Each response preserves six current suite
frames, one build and five reuses, two native executions per frame, all current
record inputs and scores4/4,2/2,1/2 repeated twice. The final label expectation
is deliberately different. Original response files remain in the CI artifact.

Across these eight repeated workloads:112/128 named expectations and192/192
record fields. Three fresh model requests make one prediction each; all four
saved replays make zero new predictions. This readback establishes this source
shape's current finite results on Linux. The macOS paired wall/CPU/RSS comparison
remains separate.
