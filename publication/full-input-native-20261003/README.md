# Full-input models in the Gooo compiler — 2026-10-03

The new V3/V4 model contracts now have a completed native integration observation.
Gooo supplied a typed construction plan, a small local judge ranked its eight
permitted paths, and finite failures guided later choices. Immediately after each
generation, the runner built the emitted Go and executed it twice.

Think of the declaration as a workshop plan. The model chooses the next assembly,
the compiler fits the typed pieces together, and the test bench records what the
assembled program does. This study follows that entire route and keeps the
original records for each step.

## What actually completed

| Observation | Actual count |
| --- | ---: |
| Known source views | 16: eight composition families, Korean/English |
| Model configurations per view | 24: four training arms × three precisions × two layouts |
| Disconnected configurations per view | 1 |
| Generations | 400 |
| Local model predictions | 816: 384 initial + 432 feedback |
| Compiled program runs | 800 |
| Finite generation/case expectations | 9,600/9,600 matched |
| Ordered outputs across both executions | 19,200 |
| Expanded/compact pairs | 192/192 matched |
| Verified progress / feedback records | 1,760 / 432 |
| Runtime model calls / external provider calls | 0 / 0 |
| Optimizer updates / automatic collection retries | 0 / 0 |
| Collector wall time | 146.71 seconds |

Each request used configuration 20, goal 4, eight allowed candidates, one attempt
per advance, and at most seven feedback rounds. The original sixteen selection
expectations stayed fixed. The native suite added eight inputs absent from that
selection suite, for 24 per generated program. These are known authored tasks;
training generalization and new intention discovery remain separate questions.
No CI hint was supplied in this observation. Layout order alternated after each
view. The default model checkpoint stayed unchanged.

All 400 runtime receipts retain `permission_boundary` as their first unresolved
dimension. Finite behavior, source links, process execution and unobserved
permissions are recorded individually. Model probabilities and finite coverage
keep their own meanings and denominators.

## First choice and additional assembly attempts

The table counts **16 requests per layout**. Expanded and compact results match
within every arm and precision. “First complete” means the first candidate met all
16 supplied selection expectations. Extra attempts count later candidates tried
before reaching the selected body. Every row ultimately matched all 384 native
expectations per layout, including the disconnected row.

| Arm | Precision | First complete /16 | Extra attempts | Predictions |
| --- | --- | ---: | ---: | ---: |
| positioned-original | FP32 | 1 | 31 | 47 |
| positioned-original | PTQ | 3 | 29 | 45 |
| positioned-original | QAT | 0 | 41 | 57 |
| positioned-varied | FP32 | 9 | 10 | 26 |
| positioned-varied | PTQ | 3 | 21 | 37 |
| positioned-varied | QAT | 3 | 27 | 43 |
| bag-original | FP32 | 16 | 0 | 16 |
| bag-original | PTQ | 1 | 19 | 35 |
| bag-original | QAT | 10 | 10 | 26 |
| bag-varied | FP32 | 15 | 2 | 18 |
| bag-varied | PTQ | 3 | 15 | 31 |
| bag-varied | QAT | 10 | 11 | 27 |
| disconnected | deterministic | 0 | 96 | 0 |

Bag-original FP32 needed fewer attempts on this small integration slice. Its
broader, previously observed 512-input development result remains 368/512 first
completions. The operation-order feature alias and quantization regressions from
the earlier study remain open. QAT improves some rows while positioned-original
QAT needs more attempts than its FP32 counterpart. All variants remain published.

## Time, CPU and memory

Local environment: darwin/arm64, Go 1.27.1, one fresh compiler process per request,
ordinary warmed Go build cache. No concurrent local source builds ran during the
timed collection. Process wall time includes startup and operating-system
scheduling; kernel prediction time excludes loading, source generation and build.

The following are medians for the compact FP32 configurations; whole-codegen and
RSS have sixteen request samples each. Prediction samples include actual feedback
calls, so their counts are 47, 26, 16 and 18 respectively.

| Configuration | Prediction µs | Model load ms | Codegen ms | Codegen CPU, one core | Codegen peak RSS MiB |
| --- | ---: | ---: | ---: | ---: | ---: |
| positioned-original FP32 | 7.71 | 0.140 | 10.18 | 79.37% | 18.18 |
| positioned-varied FP32 | 7.65 | 0.147 | 9.96 | 78.74% | 17.77 |
| bag-original FP32 | 8.33 | 0.144 | 10.00 | 78.84% | 17.78 |
| bag-varied FP32 | 7.81 | 0.148 | 9.88 | 78.63% | 17.82 |
| disconnected | — | 0 | 9.80 | 78.61% | 17.71 |

For bag-original compact FP32, the native build median was 97.51 ms, the two-run
pipeline median 301.12 ms, and each compiled process's median peak RSS 4.30 MiB.
All precision/layout modes, sample counts, nearest-rank p95 and maxima are in
`native/independent-consumption.json`. A mode has sixteen generation samples,
which makes its nearest-rank p95 equal to its maximum. Original slow samples
remain available.

CPU percentages divide a process's user+system CPU time by wall time, using one
core as 100%. Host-wide utilization change was not measured. Generated-program
RSS, compiler-process RSS, fixed tensor arrays and training RSS are different
resource scopes. Previous native studies used different process/cache conditions;
the dated measurements should be read within their own runs.

At this task size, whole-codegen medians are close to the disconnected baseline.
The reduction in extra candidates gives a concrete reason to investigate larger
assembly spaces and more expensive checks. The next timing study should measure
that relationship with repeated cohorts and explicit cache conditions.

## Source, artifacts and independent reading

- Compiler: `e461c1defddd4bb35500c7f5dfd67d2c23836156`, with released SDK
  `v0.2.15-experimental`; [integration PR](https://github.com/kimjooyoon/meta-ontology-go/pull/1158).
- Collector and preregistered consumer source:
  `07afc044810dca126cf8890dd086efe4adb56dd8` in the research repository.
- [Protocol](https://github.com/kimjooyoon/gooo-neural-decision-experiments/blob/daecfea3583614e006c960de263448a1645a9190/docs/full-input-sdk-native-protocol-20261003.md)
  SHA-256: `95cecea073effa783183803812ce5ae61f62f904beb1ca40ab01b741413431d6`.
- Model manifest SHA-256:
  `ce4ad854c0d7fcb3e7fa049658f40ea4b0393a9bff39a2ef21f64d83f221bc3a`;
  all 48 metadata/weight files come from the
  [frozen arithmetic bundle](https://huggingface.co/asketeddy/gooo-shared-judgment-tiny-v1/tree/7c4501789b34d885c14ab54aee1d40995eb8b1e6/research/full-input-separate-20261003).
- Independent consumer SHA-256:
  `e0df1e4fe16b1ab903eb1051b32b89e660c75204048953b17ffd70d2c7ab8b42`.
  It makes zero predictions and executes zero generated programs.

The consumer decodes the original compiler receipts, checks source and generated
bytes, verifies parent bindings and progress/feedback chains, and recomputes
prediction and finite-case counts. Within each representation pair the collector
compares full search/feedback semantics and emitted Go, normalizing only artifact
identity, artifact-bound chain hashes and measured timing.

The original collector, generation/runtime bytes, process observations and
independent reader are in the closed publication archive. Every member has a
byte count and digest. The packaging check inspects text, decoded JSON and embedded
parent/diagnostic bytes for private host paths and credential patterns. Model
weights stay in their separately pinned bundle, avoiding another weight copy.

The final pre-report inventory counted about 46.22 MB for this raw phase,
485.20 MB for the retained full-input/arithmetic study and 1.55 GB for all retained
`own-three` phases; 32.36 GB remained available. The registered raw-phase limit
was 128 MiB with an 8 MiB failure reserve. These inventory values precede the
final summary and publication files; exact per-file sizes are in each manifest.

## Adoption and language work

This observation used the integration branch. The deployed compiler remains on
SDK v0.2.14 until PR 1158 and the ordinary dev/main promotion complete their six
canonical checks and source-bound proof. Local focused/race checks passed.
The full local suite retains four Darwin namespace/symlink failures in unchanged
packages; CI supplies its own platform result. The earlier legacy-arithmetic
cross-platform failure also remains recorded.

The language work now has an executable example of complete text → bounded
judgment → typed body → native observation. The next questions are richer types,
more useful declared constructions, operation-order representation, new intentions,
and reusable abstractions. Each should leave measured completeness changes and
source-linked counterexamples that can guide the next development step.
