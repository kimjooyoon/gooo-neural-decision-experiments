# Five-family native continuation study

This extends actual compiler dogfood from one compound source to five existing
structural families: references, assignment targets, operand order, branch
layout and statement order. Ten parameterized fallback bodies represent twenty
intended path behaviors and forty bilingual instructions. Each has complete,
sparse or deliberately contradictory selection tests: 120 views, not 120 new
independent experiments. Reserved configurations 64 and 79 were excluded from
training, but prior development probes have used these families/templates.

The source-pinned pilot made ten native calls, zero model predictions and ten
actual Go processes with 190 function evaluations. The full fixed-order matrix
made 1,080 native calls, 1,204 model predictions (244 feedback), and 1,638 candidate
attempts. It actually executed twenty distinct emitted Go programs on 22 inputs
each: 440 function evaluations. The 9,720 policy observations on separate inputs
reuse those actual executions and are not 9,720 independent Go executions.
No training, GPU, upstream Laya or external model call occurred in this phase.

| Arm, each 120 views | Predictions, rank once / feedback | Finite cases passed / 640 | Separate input observations passed / 1,080 |
| --- | ---: | ---: | ---: |
| Parent FP32 | 120 / 177 | 600 | 1,017 |
| New feedback FP32 | 120 / 182 | 600 | 990 |
| New feedback PTQ | 120 / 183 | 600 | 999 |
| New feedback QAT | 120 / 182 | 600 | 990 |
| Deterministic offline | 0 | 600 | 990 |

Every arm selects a finite best option. The deliberately contradictory duplicate
retains its denominator. Sparse witnesses can make both options equally valid
on selected inputs while they disagree on other inputs: finite completion is
not original-intention or unseen-behavior completion. The new models do not
improve on the parent in these separate-input observations.

All 480 rank-once/feedback pairs preserve the same Go, finite passes, separate
passes and attempts. **One decision has only two declared paths.** After the
first failed candidate, one path remains; another ranking cannot change which
path is next. This is useful breadth evidence and an observed redundant-call
case, not evidence that feedback is ineffective for larger composed plans.

Complete native-process wall medians were 5.276 ms offline; 5.453/5.466 ms parent
rank-once/feedback; and 5.394–5.482 ms across new-model arms. CPU-time medians
were 4.554–4.771 ms, and median child peak RSS 16,949,248–17,268,736 bytes. These
are per-process observations in one baseline-first ordering with no repeats;
host utilization and causal speedup are not measured. The raw matrix retains
every observation, including startup outliers.

Go audit reinterprets every candidate and checks source/cohort/model/progress,
counterexample, actual-Go and paired-outcome bindings with zero fresh model/native
calls. Frozen source: `30fbf81b951e12d504aca19d06e8d1a8dc08a865`; native main:
`4dced73b26dde567cb7129f3e4ba5733850196d0`; SDK `v0.2.4-experimental`.

## Observed repairs

The next SDK change records a hashed `ranking_unnecessary` observation with zero
predictions when exactly one unattempted declared path remains. It retains the
counterexample, original model/case/plan bindings and frontier, consumes a round,
and rejects a repeated same-batch reconsideration. Missing or changed models,
invalid CI hints and cancellation still fail. This is implemented and unit-tested
in research; downstream SDK/native deployment and fresh measured comparison are
separate work, not an already measured speedup.

The old SDK receipt's fixed wording says “non-feedback-trained” even when our
new models have actually been tuned on finite feedback. The raw old wording is
preserved. Future receipts describe the original frozen model without asserting
an unobserved training history. This is a provenance-description repair.

The first cohort build referred to an SDK-only document-schema constant absent
from the research package. That compile failure executed zero native calls; the
correct native document schema was supplied and focused race/vet rerun passed.
An argument-order defect was corrected by inspection before actual matrix calls.

[Cohort](../studies/feedback-family-v1/manifest.json),
[pilot audit](../runs/feedback-family-pilot-20261001/audit.json),
[matrix and full captures](../runs/feedback-family-matrix-20261001/report.json).

## Public evidence appendix

The same frozen evidence is public on
[Hugging Face at immutable revision 9b26f9d](https://huggingface.co/asketeddy/gooo-feedback-path-tiny-v1/tree/9b26f9dbe2afba9abb74886c1abfa6cac5ac7ee8/research/native-family-20261001).
The deterministic gzip archive is 986,241 bytes and contains 1,128 allowlisted
raw JSON/JSONL files with per-entry hashes and sizes. Nine publication files
were verified by ten anonymous HTTP requests; the existing 25 core model files
were separately verified at that same revision by 26 anonymous requests. The
appendix adds evidence without replacing the frozen model card or weights.
Packaging and verification execute zero new model/native/Go calls. Local CI
reproduces the archive manifest without credentials or network publication.

The SDK `v0.2.5-experimental` source release passes Go 1.27.1 format/vet/unit/race
checks (run 36814919123). Native integration is tracked separately in
[PR 1122](https://github.com/kimjooyoon/meta-ontology-go/pull/1122). The first
focused native test edit had a missing loop brace; compilation failed before
native execution, the brace was repaired, and bilingual focused race/vet passed.
The first publication tool build had an unused import; no publication occurred
until the corrected tool verified the frozen allowlist and archive.

## Actual optimized feature comparison

Runner `1727f64e5636ae1fec985aaa1fa1a2da27035a57` executed the same 1,080 views
using clean feature binary `bf19d5b84fe27c15f3571a6ec14e6f95fed565c5` and SDK
`v0.2.5-experimental`. It made 960 initialization predictions, **zero feedback
predictions**, and retained 244 hashed `ranking_unnecessary` receipts. All 1,080
paired observations preserve the selected label, generated Go, finite passes,
separate-input passes, intention agreement and candidate count. There are still
1,638 attempted candidates, 5,400/5,760 repeated finite passes and 8,982/9,720
repeated separate-input passes. Twenty actual Go processes perform 440 actual
function evaluations. No fresh training or upstream Laya call occurred.

The removed 244 predictions are 20.27% of the baseline's 1,204 total predictions.
This is an exact invocation-count reduction, not a demonstrated wall speedup or
host utilization decrease. It preserves partial construction and failure lineage
instead of treating another model judgment as a requirement for progress.

The feature run's predeclared caller CI hint was `PASS`; it is an unauthenticated
context input, not a check result. Actual CI 36815255280 later failed its module
fixed-point step because two obsolete v0.2.4 checksum lines remained. Updating
the branch canceled that old run; the failure and feature captures are retained.
`go mod tidy` removed only those unused lines, yielding source
`6636f13953ad0652d3a46d578d0299a6bad9a77d`. Promotion requires fresh exact-head
CI and proof. These feature results do not establish main deployment.

[Optimized feature matrix](../runs/feedback-family-optimized-feature-20261001/report.json)
and [zero-inference paired audit](../runs/feedback-family-optimized-feature-20261001/audit.json).

The optimized feature archive is also public at
[immutable Hugging Face revision 651e956](https://huggingface.co/asketeddy/gooo-feedback-path-tiny-v1/tree/651e956c8287e59244f5ee68a359254028c8fec6/research/native-family-optimized-feature-20261001).
Its 878,001 compressed bytes retain 1,105 raw allowlisted files. Seven files
were anonymously verified by eight requests. The older nine-file appendix and
25-file core were separately reverified at that same revision. No unpublished
host path or credential was included by the allowlist packager.

Go-derived [descriptive resource metrics](../publication/native-family-descriptive-metrics-20261001.json)
retain all 120 process observations per arm and both editions. Optimized native
wall medians are 5.280–5.485 ms, CPU-time medians 4.562–4.760 ms, and median peak
RSS 16,891,904–17,203,200 bytes. Median process CPU/wall ratios are 86.12–86.99%
of one core. These figures are whole compiler subprocess measurements, not
host CPU utilization or its increase, and do not establish a wall-speedup claim.

## 개발에 사용하는 방식

자체 Gooo 판단 모델은 한글·영어 의도를 선언된 경로 후보와 연결합니다.
현재 모델은 지역 변수 참조, 변수 할당 대상, 피연산자 순서, if 분기의 배치,
문장 실행 순서를 판단합니다. 컴파일러는 선택된 경로로 실제 본문을 구성하고,
타입·범위·원본 소스 연결을 검사한 뒤 Go 코드를 생성합니다. 개발에서는
모델의 첫 응답과 별개로 관찰한 실패를 기록하면서 다음 후보를 구성합니다.

이번 실행에서 사용한 명령 구조는 다음과 같습니다. `plan.json`에는 Gooo
원본에 연결된 본문, 허용된 대안, 테스트 입력·기대값, 최대 시도 수가 들어갑니다.

```sh
gooo body-codegen --json --path-plan plan.json \
  --path-model model-bundle/models/qat_ternary/model.json \
  --path-step-attempts 1 --path-feedback-rounds 1 \
  --path-feedback-ci ci.json --activity ChoosePath source.gooo
```

개발 순서는 의도와 타입이 있는 Gooo 본문을 선언하고, 모델이 후보 순서를
힌트로 주고, 테스트가 관찰한 결과를 다음 단계에 연결하는 흐름입니다.
테스트가 부족하면 별도 입력 통과율과 의도 경로 일치율도 함께 기록합니다.
표현이 모델의 입력 크기를 넘으면 그 상태를 기록하고 기존 후보 구성을
이어갑니다. 후보가 하나 남으면 모델을 추가 호출하지 않고 그 후보를 검사합니다.

모델을 연결하지 않을 때는 모델·피드백 옵션을 생략합니다. 같은 원본·계획·테스트·
예산에서 선언된 순서로 후보를 평가하므로 최종 코드와 선택 경로를 재현할 수
있습니다. 실행시간 측정은 별도로 기록합니다. 새 모델은 실험 체크포인트로
보존하며, 이전 모델과의 회귀 결과도 공개합니다. SDK v0.2.5 연결은 개발
브랜치에 병합됐고 [main 승격 PR 1123](https://github.com/kimjooyoon/meta-ontology-go/pull/1123)은
별도의 필수 CI와 소스 연결 증거를 검사합니다.
