# Native path diagnosis integration smoke: preregistration

Before new native execution, commit this document and the Go runner. Use a clean
Go 1.27.1 compiler at the supplied exact source revision, with SDK 0.2.8. Verify
its build information and record the binary digest. Pin the already published
own FP32 metadata and weights; do not train or call upstream Laya.

Use the native path-diagnosis source/plan/input fixture: one binary subtraction
order choice, one declared case `(2, 0)`, construction budget one, diagnostic
budget two, probes `[2, 3]`. Replace only its decision instruction with fixed
Korean `입력값에서 2를 뺀다.` or English `Subtract two from the input.`.

Run four pairs: Korean/English × disconnected/own FP32. Each pair constructs
with diagnosis disabled and enabled, alternating their order across pairs.
Thus eight native calls, four predictions, four diagnoses and eight diagnostic
candidate observations are planned. This is one reused synthetic intention,
not eight independent tasks or a language accuracy benchmark.

Both legal candidates pass the finite case. For every enabled receipt, require
two case-indistinguishable candidates, one separating witness at input three,
one bounded unresolved reference and zero diagnostic predictions. Preserve
the actually selected reference and its outputs, even if it disagrees with the
natural-language instruction. The arithmetic oracle is `x-2` for mask zero and
`2-x` for mask one. Witness outputs are not expected answers.

Require paired emitted Go and selected choice to stay identical. Independently
compile/run each distinct actually emitted Go on `[2, 3]`, retain output values,
and compare against the integer oracle. The audit reuses captures without new
predictions, native calls or generated-Go processes. Prediction origin depends
on the pinned clean runner/model and source CI; an offline audit does not replay
ranking. Alternative candidates remain typed-arena observations, not native
executions. Save the preexecution receipt before the first native invocation.

Record raw native receipts, whole-child wall/CPU/peak RSS sidecars, separate
diagnostic time and emitted-Go execution captures. Eight calls provide
integration evidence, not stable performance estimates, causal speedup or host
CPU utilization change. A failed attempt must remain recorded and separate.
