# Own small models connected to Gooo compiler facts

The feature implementation binds each original typed fallback to authoritative
Gooo source before making compiler-built context for own split-v2 models.
Korean/English natural intention is preserved; facts include local references,
assignment targets, operand order, branch bodies and execution order. Finite TDD
then assembles valid candidates, and the selected Go is type checked, replayed
and independently executed. Python is absent from runtime and orchestration.

## Feature observations

Source `052c250208e38f33a65242e38b042aab80cf85b7`, clean Go 1.27.1 CLI with
SDK 0.2.9. The published preregistered runner
`a60b5b802e0f2251b262c7a31cc7ebe6751a4464` made 32 actual native child calls
and 57 initial predictions. It executed all 32 selected Go packages with 120
function invocations. Separate audit reconstructs typed candidate actuals and
pass flags, checks subtraction arithmetic and compares independently compiled
Go outputs. It makes zero further model calls.

The selected repeated finite observations are 116/120. Every non-overflow cell
satisfies its authored cases. Four partial-contract cells retain six of seven
cases (85.714%); compiler lowering remains complete. These are three reused
intentions across languages, contracts and variants plus a synthetic bounds
probe. They are not 32 independent tasks or a language-accuracy percentage.

Three model cells exceed the 512-byte context bound and record zero-prediction,
zero-feedback deterministic continuation. They preserve partial bodies and full
attempted byte counts. Three sparse subtraction choices disagree with the
intended `input-2` body while passing input 2. Full contracts distinguish these
paths. This preserves the reason to measure finite coverage, added assembly and
intention alignment separately.

There are 455 candidate attempts, 423 after the first, and 2,999 actual attempted
case evaluations. The broad conditional and deliberately inconsistent partial
contract dominate these costs. Extra candidate construction makes zero new
model predictions; authored finite tests supply repair authority. It is not a
learned autonomous failure-repair policy.

Measured initial prediction time: 57 calls, minimum 10.833 microseconds, middle
observed value 12.542 microseconds, maximum 36.417 microseconds. Context preparation
for 21 connected non-overflow cells ranges 0.0425–0.703917 ms (middle observed
0.088917 ms). Whole-child peak RSS ranges 16,580,608–22,102,016 bytes. The first
native child wall time is 451.4045 ms despite native internal timing 1.330042 ms;
retain this startup outlier. Child CPU is `(user+system)/wall`, a one-core-equivalent
ratio. No sampled host CPU utilization, independent causal speedup or warm-service
latency claim. Raw sidecars retain all child timings and resource observations.

Own models have 12,728 parameters, a fixed 256-feature/48-hidden/8-label layout and
1,248-byte inference workspace. FP32 tensor payload is 50,912 bytes; ternary
exports use 2,759 packed disk bytes and decode to an int8 resident tensor layout.
Five trits per byte is 1.6 disk bits/trit; this is not packed compute or complete
process RAM. These experimental models are independently initialized own models,
not Laya weight derivatives. Prior split-feature regressions remain published.

## Development direction

Train the next iteration on compiler-emitted facts and bounded Korean/English
intentions from disjoint Gooo source groups. The current context differs from its
training text distribution, so this integration is not evidence of improvement.
Keep finite reachable coverage, unresolved cases, attempts, actual calls and
resources alongside optional first-choice agreement. Preserve ambiguous and
contradictory contracts. Do not teach a model that repeated retries imply truth.

Extend assembly through typed Gooo templates and legal alternatives for operators,
conditions, variable declarations and statements. More model freedom needs
corresponding compiler-owned type, locality, source identity and finite execution
checks. A model suggests paths; executable Gooo contracts define what was achieved.
Source, intent, weights, context, candidate, test and emission digests can link
provenance observations without making inferred ontology facts authoritative.
