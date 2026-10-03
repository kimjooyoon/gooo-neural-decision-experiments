# Complete historical arithmetic replay — 2026-10-03

## 한국어 요약

같은 모델과 입력에서도 계산 환경에 따라 아주 작은 실수 차이가 생겼고,
첫 후보 뒤에 시도하는 순서가 달라졌습니다. 원래 Linux 비교에서 101개 요약
항목과 272개 전체 후보 순위가 달랐습니다. 앞선 명시적 반올림 규칙은 이 문제를
해결했고, 현재 SDK와 컴파일러에서 사용하고 있습니다.

이번에는 과거 기록의 재현 방법을 정리했습니다. Go 검증기가 자신이 실행되는
환경의 고정 원본과 새 결과를 모두 대조합니다. 원래 두 환경의 차이도 항목과
양쪽 값을 함께 기록하므로, 과거 불일치와 이번 재현의 결과를 같이 읽을 수 있습니다.
이미 쓴 자의 눈금을 기록해 두고, 새 눈금으로 측정한 결과도 따로 확인하는 방식입니다.

## Checks and denominators

| Fresh collection | darwin/arm64 | linux/amd64 |
| --- | ---: | ---: |
| Complete development rows checked | 18,432 | 18,432 |
| Compact metadata/weight files, exact bytes | 24 | 24 |
| Expanded export parity predictions | 624 | 624 |
| Compact export parity predictions | 624 | 624 |
| Compact calibration predictions | 6,144 | 6,144 |
| Compact development predictions | 18,432 | 18,432 |
| Total actual audit predictions | 25,824 | 25,824 |
| Same-platform replay result | PASS | PASS |
| Original arm64-reference summary differences | 0 | 101 |
| Original arm64-reference complete ranking differences | 0 | 272 |
| Original comparison status | PASS | MISMATCH |
| Reader model predictions / optimizer updates / native executions | 0 / 0 / 0 | 0 / 0 / 0 |

These are the same previously observed 512 development views in three input
forms, four training arms and three weight variants. Repetition leaves the
intention denominator unchanged. All original models and expectations are fixed.
The current 4,096-parameter whole-candidate order judge is a separate model;
this historical audit uses the earlier shared-judge artifacts.

The reader checks complete text/input/source/finite-target identities, every
float32 logit/probability bit, selected mask and all eight ranks. It rejects
missing/extra files or rows, changed models, duplicate views and malformed
arrays. The report retains the previous 0.001 accumulated mass/NLL tolerance
and 1e-5 export bound. The runtime chooses the baseline: Go 1.27.1 on
darwin/arm64 or linux/amd64. Other platforms require their own evidence.

The new Linux record retains every one of the original 101 path/value pairs.
An independent sorted comparison also matches the last failed workflow's
[complete difference list](https://github.com/kimjooyoon/gooo-neural-decision-experiments/blob/main/publication/toolchain-version-reuse-20261003/research-legacy-arithmetic-differences.json).
The unmodified original comparator was executed again and exited with failure;
its complete output is [preserved](original-cross-platform-comparator.raw).

The independent `float32_separate_v1` job continues to compare all 18,432
cross-platform rows, hidden/logit digests, complete rankings and finite outcomes.
Its [original paired result](https://github.com/kimjooyoon/gooo-neural-decision-experiments/blob/main/docs/full-input-separate-arithmetic-results-20261003.md)
remains unchanged. Go's [floating-point operator specification](https://go.dev/ref/spec#Floating_point_operators)
describes operation fusion and explicit conversion rounding; the earlier repair
put those rounding points in the model's declared execution contract.

## Source and original records

- [Research PR #1](https://github.com/kimjooyoon/gooo-neural-decision-experiments/pull/1)
  merged normally as `87c6f219289578f5ca861d4349712e74055e64d2` after
  [all 19 Linux jobs](https://github.com/kimjooyoon/gooo-neural-decision-experiments/actions/runs/37126899513)
  passed. The merged computational tree equals the tested tree.
- Local audit/reader source: `9200a1888f3e3fccb4648fbaf707528a467c8ae7`.
- Linux checkout: `f3f2374d5922698f9f3cd7087215a3427757a628`, the PR merge
  tree with parents `78b51f0` and `9200a18`. Its tree equals the local source tree.
- [Protocol](https://github.com/kimjooyoon/gooo-neural-decision-experiments/blob/main/docs/legacy-platform-replay-protocol-20261003.md) and
  [Go reader](https://github.com/kimjooyoon/gooo-neural-decision-experiments/blob/main/tools/package-full-input-study/platform.go).
- [macOS reader](arm64-replay.json), [Linux reader](linux-replay.json), and the
  complete fresh [macOS](arm64-audit.zip) / [Linux](linux-audit.zip) audit archives.
- [Independent downloaded-file/source check](linux-independent-files.json)
  verified all 61 downloaded files and all 77 computational-source pins.
  All 60 non-report Linux files also match the frozen Linux compressed/model
  bytes exactly. The report preserves its new auditor revision.
- [Downloaded-artifact verification source](verify-downloaded.go), with zero
  model calls, takes the reader JSON, decoded audit directory and computational
  source checkout as its three arguments.

The original manifest hashes remain
`45f78a838c0a7cc7255b26d596db3871a283cf6f1a8504a74f9cb211329ad32a`
and `59575533d14a660886a7c6b78d08475b50153ec39919d2ecf91945663e6aac37`.
The adjacent SHA256SUMS binds every new published file. Decoded text and journals
were checked for personal filesystem paths and token/private-key patterns.

## Run the current contract

Use a clean research checkout and Go 1.27.1. The full
[CI recipe](https://github.com/kimjooyoon/gooo-neural-decision-experiments/blob/main/.github/workflows/go-model-validation.yml) unpacks the unchanged
training/source archives and makes a fresh audit. Its final reader invocation is:

```sh
go run ./tools/package-full-input-study --mode verify \
  --compare-platform-audit runs/own-three-full-input-ci-replay \
  --replay-revision "$(git rev-parse HEAD)" \
  --comparison-output /tmp/legacy-platform-replay-new.json
```

Both audit and comparison destinations must be fresh. To read a saved result,
look at `status`, then `original_comparison.status`, its complete
`summary_differences` and `changed_rank_rows`. The first reports whether this
historical platform replay reproduced its own observations. The second reports
how the same results compare with the original arm64 reference. Model-call
counts are in the audit report, while the reader's call count is zero.

## Local failures and negative checks

The initial test-first build failed on the missing reader functions;
[first-tests.raw](first-tests.raw) retains it. The completed archive tests pass,
and focused race tests pass in 79.680 seconds. Source packages pass vet with
`go vet ./cmd/... ./internal/... ./tools/...`.

The initial local `go vet ./...` attempt encountered an ignored retained
experiment's `runs/compact-shared-native-20261003/independent-consumer.go`, whose
imports require the earlier compiler/SDK module context. Its original failure is
[vet.raw](vet.raw). Clean-checkout CI passed its whole-module vet and race checks separately.

Negative CLI checks on temporary copies reject changed weight bytes and an
extra inventory file; both exit with failure and produce no success report.
Their outputs are retained alongside unit tests for changed row identities,
targets, signed zero, numbers, ranks, missing/extra rows, null/short arrays,
report paths/values and unsupported platforms.

Inference latency, CPU utilization and RSS were not measured in this repair.
The earlier [native execution cost study](https://github.com/kimjooyoon/gooo-neural-decision-experiments/tree/main/publication/toolchain-version-reuse-20261003)
has its own measured scopes and unchanged model weights.
