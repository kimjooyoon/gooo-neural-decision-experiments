# Observed feedback for partial Gooo judgment

The experimental research API now supports explicit `Session.Reconsider`.
The default session and the released native compiler still rank once; this
opt-in API reuses the **same frozen model** after a new partial batch. It is
not training, a learned repair policy or production online learning.

## What the model receives

Each decision receives a bounded prefix containing committed attempts,
selected passing/total finite cases, type rejections, remaining paths, the
selected label and the first failing selected case as input/actual/expected.
An optional caller CI status is included before the unchanged original intent.
Full UTF-8 text is retained in the returned judgment receipt. Every input is
validated against the existing 512-byte ABI **before any prediction**; long
intents are rejected for this operation rather than silently truncated.

The existing positioned-intent feature contract still finds the final
`intent: ` marker. Context is weakly weighted, and the original model was not
trained to interpret this failure prefix. Measuring no improvement or extra
cost is a valid result. First-choice accuracy is not the acceptance objective.

## What can change

Only remaining path priority and an unscheduled model proposal can change.
The original plan, source intent, tests, weights and best observed body stay
fixed. Committed masks cannot repeat. A temporary finite heap is built and
committed only after complete predictions and cancellation checks. The heap
uses fixed-size numeric nodes; its temporary backing array adds memory during
reconsideration and is not included in the scheduled-bitset byte count.

No model pointer or old feedback logs are retained in the session. The caller
owns each returned receipt. It binds plan/cases/model hashes, the prior progress,
the previous feedback, actual prediction count, contexts and observed failures.
The next progress links the latest feedback digest and cumulative feedback calls.
Original eligible probabilities and initial proposals remain the initialization
observations; current feedback probabilities live in its separate receipt.

At most one reconsideration is allowed per newly committed batch, and there
are at most 16 rounds per session. A cancellation after predictions preserves
their count and a failed receipt, leaves the frontier unchanged and consumes
that batch's reconsideration. Search can continue normally. Invalid input or
pre-cancellation makes zero calls. There are no retries or waiting mutex calls.
This bounds the local CPU operation; arbitrary output-writer deadlock freedom
is not claimed.

## CLI

```sh
go run ./cmd/gooo-path-compose --plan plan.json --tests tests.json \
  --model model.json --max-attempts 64 --step-attempts 8 --feedback-rounds 2
```

The stream interleaves compose progress and feedback judgment JSON lines.
Reconsideration occurs only after partial, non-exhausted batches with remaining
attempt budget. It adds actual model calls; it never pretends to be free.
Omit `--feedback-rounds` for the unchanged rank-once behavior. Omit the model
as well for deterministic disconnected behavior.

`--feedback-ci ci.json` optionally accepts a regular file at most 512 bytes:

```json
{"source_sha":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","status":"UNKNOWN"}
```

Allowed statuses are PASS, FAIL and UNKNOWN. The source SHA identifies what
the caller says was checked. **This API does not verify GitHub or confer CI
authority.** The hint cannot authorize source edits, override compiler checks,
replace finite denominators or introduce Guardian/human-review decisions.
Public studies must separately bind any real hint to its actual CI receipt.

Finite success still means stated cases only. Unknown intent, contradictory
cases, type failures, partial bodies, unattempted paths and resource cost remain
separate observations. A complete finite result does not prove arbitrary
natural-language correctness or justify ontology state changes.
