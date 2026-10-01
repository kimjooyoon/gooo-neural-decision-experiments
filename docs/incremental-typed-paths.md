# Incremental typed Gooo path search

One prepared Gooo plan and one copied finite test suite define a session.
Starting it ranks each declared decision once with the optional local model;
`Advance(ctx, 1..64)` tests only new masks. Without a model its ordering is
fully deterministic. It emits no network requests and retains no model pointer.
Changing the plan, cases, model or seed requires a new session.

This targets repeated work when a small initial budget leaves a partial result.
It does not optimize first-proposal correctness or turn finite tests into proof
of general natural-language understanding. Legacy `Search` remains unchanged.

## Working API

- `prepared.NewSession` owns copied cases and the prepared immutable arena.
- `Observe` records initialization, including actual calls made before a
  cancellation or failed numerical prediction. It evaluates no candidates.
- `Advance` returns the selected typed arena and caller-owned progress with
  only this call's new attempts. Earlier completed masks cannot repeat.
- A cancellation in the middle of a candidate puts that unfinished candidate
  back on the frontier; partial case results do not become committed success.
- Concurrent advance/observe returns `ErrSessionBusy` immediately. There is no
  mutex wait or worker/channel dependency inside this CPU-only session.
- `SearchBatches` adapts sessions to the existing 1..64 total-attempt native
  document contract. Direct `Advance` calls can traverse the full declared
  finite space, up to 65,536 masks from 16 decisions.

Each batch separates cumulative attempted/evaluated/type-rejected counts,
passing/total declared cases, unattempted combinations, new attempts and the
initial model observations. A predecessor/digest chain binds emitted progress
and the fixed case/plan/model hashes. These are observations, not signed
permission to edit source or accepted ontology facts. No deserialized progress
is accepted as executable state.

## Memory layout and scope

The scheduled set is `ceil(combinations/64)` uint64 words: 8 bytes for the
six-decision 64-path example, at most 8 KiB for all 16-bit masks. The session
keeps a finite heap, the best program and up to 128 selected case results. It
does not retain prior attempt/result logs. Each returned batch is limited to 64
new attempts, each with at most 128 cases; the caller may stream or save them.

`scheduled_bitset_bytes` measures only that array. `frontier_capacity_bytes`
measures the Go heap node backing array; both exclude the prepared body, maps,
model tensors, Go/runtime overhead and caller-retained reports. They are not
RSS measurements. A large declared frontier can still consume more memory.
The native compatibility helper deliberately retains at most 64 total attempts
and their progress, matching the existing strict document envelope.

## CLI

`gooo-path-compose --plan plan.json --tests tests.json --max-attempts 64
--step-attempts 8` emits JSON lines: one initialization record, then each bounded
step and its current best Gooo/Go body. Partial functional results remain marked
`PARTIAL`; exhaustion and passing all stated cases are separate fields. The
same model-free inputs reproduce identical records and code.

With `--step-attempts`, `--max-attempts` accepts 1..65536 total candidates;
each advance is still limited to 1..64. The prepared plan declares at most
65536 masks and exhaustion can stop before the requested budget. Without the
step option, legacy search retains its 1..64 cap. Tests require explicit,
non-null integer `input` and `expected` values. The CLI deadline remains at
most 30 seconds and does not cover a blocking generic output writer.

The CLI race regression now traverses all 128 masks of a seven-choice synthetic
plan over two 64-attempt records. This tests the interface boundary, not trained
model quality, inference performance, another independent intent or execution
of all 65536 masks. The primary 288-call native measurements remain unchanged.

The core deadline/cancellation/nonblocking-lock contract covers its local
ranking and finite evaluation. Output writers are a caller-owned I/O boundary;
process harnesses must drain or bound them rather than leave stdout unread.

## Verified coverage

Local race tests compare complete and deliberately inconsistent finite cases
against legacy search with deterministic and seeded synthetic-model ranking;
all masks, typed rejection, choices and code remain equal. They exercise copied
inputs/outputs, cancellation rollback, failed-ranking call accounting,
nonblocking concurrent access, 128 paths over two 64-attempt calls without
retained old logs, and the highest 16-bit mask. These fixtures are API/bounds
regressions, not independent language tasks or a measurement of trained models.

The first fixture authoring mistakes were a bias-tensor shape and an arena with
unused expressions. Existing validation refused both; the corrected fixture
uses the required 1-by-N bias shape and a used expression chain. Compiler/model
validation was not weakened. Fresh trained-model/native measurements will be
reported separately from these unit regressions.
