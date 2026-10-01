# Next study: source-bounded continued judgment

Status: proposed study design. No measurements or newly trained weights are
claimed by this document. The native feedback integration results are recorded
separately in `native-feedback-results.md`.

## Objective

The useful objective is continued construction of declared Gooo behavior from
bounded Korean/English intent, with residual failures and cost visible. A
first-choice mistake is an observation that can guide another unattempted path.
Perfect single-response expression is not the acceptance criterion.

The compiler supplies the typed structure, stable IDs, immutable source and
finite acceptance contract. The tiny model ranks alternatives that already
exist in that structure. Successful deterministic type, replay and behavioral
checks determine whether a selected body can be emitted. An output with six of
seven cases satisfied can remain a measured partial result.

## Eight separately measured areas

| Area | Constructed Gooo detail | Separate observation |
| --- | --- | --- |
| Intent judgment | Korean/English reference intention | abstention, bounded input and allowed-label selection |
| Predicate operands | references inside comparisons | finite predicate outcomes and type rejections |
| Boolean locals | guard references and update targets | branch reachability on declared cases |
| Integer locals | assignment destinations | assignment-target construction and output outcomes |
| Branch bodies | existing then/else alternatives | declared branch coverage and unresolved cases |
| Statement order | declared scheduling choices | order-induced counterexamples and repeated-path count |
| Partial continuation | original intent plus observed failure | best-score retention and new-candidate progress per batch |
| Provenance consumption | verified facts plus explicit caller claims | source/case/model hash binding and non-authorizing hint scope |

These are areas to sample with multiple independently specified intentions,
not eight independent results from the existing single-intention study. Current
native typed-path support is the pure Integer-to-Integer bounded profile; this
plan does not claim unrestricted loops, calls, effects or arbitrary text bodies.

## Source-backed feedback curriculum

1. Declare varied Gooo templates and intended finite behaviors before model
   training. Derive labels from allowed path options and deterministic executable
   witnesses. Keep ambiguous options as sets or abstentions rather than inventing
   a unique oracle from an insufficient test suite.
2. Record actual partial sessions: attempted masks, best finite outcomes, first
   mismatch and unchanged original intention. Do not turn a CI PASS claim into
   proof of semantic correctness. A source-pinned CI receipt and an unverified
   caller hint have distinct provenance.
3. Split by source template and intention family before generating paraphrases
   or feedback variants. Keep Korean and English counterparts in the same split.
   Repeated contexts from one session must not leak into independent evaluation.
4. Compare frozen rank-once, deterministic enumeration, frozen feedback and a
   separately trained feedback model. Give each arm the same total candidate
   and time budget. Report inference overhead separately from binding, compiler
   emission, output transport and actual generated-program execution.
5. Train offline on the available GPU in bounded runs, then export fixed weights
   to the Go runtime. Runtime orchestration remains Go; no optimizer runs inside
   the compiler. Publish only reviewed synthetic inputs, bounded receipts,
   weights, metadata and source hashes through an explicit file allowlist.

## Completeness and resource measures

- Satisfied finite cases / declared finite cases, with the denominator retained.
- Lowered selected constructs / declared selected constructs, separately from
  unstated natural-language intent coverage, which remains UNKNOWN.
- Unique attempted masks / declared masks; type-rejected and unattempted counts.
- Best finite score before and after each batch; unresolved counterexamples.
- Actual prediction calls, prediction time and candidate progress per call.
- End-to-end wall time, process CPU time, process maximum RSS and output bytes.
- Actual generated-function evaluations, distinct inputs and reused policy
  observations as different counts.

For the current study the inference sum is small relative to the complete native
process. Any optimization should first measure binding, native validation,
startup and output costs. Faster model selection alone does not imply faster
complete compilation.

PTQ and QAT ternary exports should retain separate disk and resident-memory
measurements. The existing packed storage uses 1.6 bits per ternary weight;
log2(3), approximately 1.58 bits, is the theoretical information bound. Existing
Go inference expands weights to int8 rather than executing a packed ternary
kernel. A packed-compute experiment needs its own parity and cost evidence.

Publishing a failed expression, a partial body or a negative performance result
is useful when its inputs, remaining behavior and resource cost remain
inspectable. Public iteration should continue without making hidden human
approval a model or compiler input.
