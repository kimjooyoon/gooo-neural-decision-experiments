# Own Gooo model: installed-main replay

Compiler feature PR #1133 and exact-dev-tree promotion PR #1134 are merged.
Main revision: `b431ef7547a968e2532fa2dcf0891ee44ec2ee86`.
Dev revision: `0ec9eaed55cefb43d18feb54488a2bc2625d82af`.
Tree: `1e4725058cf42c4a73ecf03cae115004f0db141e`.
Six canonical checks passed on the feature and promotion revisions; downloaded
CI proofs independently verified PASS. Main strict/admin protection keeps six
canonical checks and zero required reviews, with no Guardian status requirement.

Clean main binaries were installed locally with Go 1.27.1 and decision SDK
v0.2.9-experimental. Gooo binary SHA:
`4fe98228a5ce94766a2178ef1e35ec3826cc74ae24a679d7ae0d323618880981`;
retained worker SHA:
`cff4c667e7b891863130f2283b39744ad62a25be5d9b66fb888072ac66618f43`.
Build metadata reports exact main VCS revision and `vcs.modified=false`.
No default global executable or hosted service deployment is claimed.

Following the prepublished main replay protocol, the unchanged 40-view subset
was executed with compiler-context FP32/PTQ/QAT/disconnected policies: 160 actual
native generation calls, 120 model predictions, 20 compiled-Go executions.
Independent case and input-hash audit PASS. All 160 normalized feature/main
observations are identical: source/semantic/context/model bindings, choices,
candidate attempts and case actuals, emitted Go and executed Go values. Source
revision, timing and process resources remain separately recorded.

Total local study observations across this iteration: 4160 compiler export calls
(2080 rejected from training, 2080 validated), 1920 held-out Go predictions,
192 Go numerical-parity predictions, 320 native generation calls, 240 native
model predictions, 40 compiled-Go executions (480 function invocations).
That is 2352 study predictions total, excluding unit-test calls and CI repeats.
There are 560 optimizer updates and zero new independent intentions/Laya calls.
The reused development benchmark and negative matched-context results remain
unchanged; main consistency does not turn those regressions into model quality.

The own model and append-only evidence are public at
[Hugging Face](https://huggingface.co/asketeddy/gooo-compiler-context-tiny-v1).
Remaining model work follows the registered family/language bottlenecks and
bounded study directions, with deterministic finite completion retained.
