# Prepared typed paths

The native adapter currently prepares the same plan through document validation,
fallback validation, fallback compilation and search validation. Each preparation
type-checks the fallback and every individual alternative. Candidate search already
checks each complete combination once; those checks must remain.

`pathplan.Prepare` creates a bounded, owned snapshot and retains its immutable
compiled fallback. `PreparedPlan.Search` allocates its own workspace, choices and
heap and still runs the existing arena compiler on every complete candidate.
There is no global cache and no retained test result, confidence decision or model.
Caller-owned input must be synchronized while preparing; subsequent mutations of
input expressions, roots, options and order arrays cannot change the snapshot.
Returned default maps and search receipts are independent copies.

The plan hash is the same canonical JSON hash used by the previous search API.
The existing `Search` entry point prepares one snapshot before delegating, so its
signature remains compatible. The native adapter can prepare once, bind the cached
fallback to authoritative Gooo, load the explicit model and search that snapshot.
Original source equivalence, combined scope checks, final native emission,
replay, finite outcomes and partial completion remain separate checks.

## Library observation

The five-run Go 1.27.1 microbenchmark compares the current public API calls needed
by the previous native adapter (four preparations) with one prepared snapshot and
the same bounded search. It uses one two-decision interacting plan, no model and
an intentionally unattainable finite expectation. All four combinations are
attempted, retaining the combined scope rejection. The source was uncommitted at
capture; exact source-file hashes are retained with the benchmark.

| Measure, median | Repeated API calls | Prepared snapshot |
| --- | ---: | ---: |
| Time per operation | 405,112 ns | 141,354 ns |
| Allocated bytes per operation | 572,740 | 199,357 |
| Allocations per operation | 9,595 | 3,250 |

This is a library API comparison, not an old/new native binary measurement or a
model speedup. Total allocated bytes are not retained RAM or lifetime RSS. The
separate source-bound native comparison must use actual clean binaries and the
same source, cases, weights and attempt budgets before attributing an end-to-end
effect. Raw evidence is in
[the preparation capture](../runs/typed-path-preparation-20261001/).

## Next native workload

Use six interacting paths: a comparison's operand order, a Boolean local
reference, a Boolean assignment target, an Integer assignment target, `if` body
layout and Boolean-update order. The existing structural label contract already
describes these edits; no comparison labels or arbitrary text are reinterpreted
from the operation model. Test both Korean and English, short/full attempt
budgets, deterministic/model arms and independent arithmetic inputs. Preserve
failed and partially completed paths; this iteration freezes the nine existing
model bundles and performs no optimizer steps.
