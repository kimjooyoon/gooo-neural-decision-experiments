# Exact missing-only SDK continuation implementation

This implements the separately published storage continuation without changing
the original protocol or reclassifying its failed 768 MiB attempt. The resource
amendment remains pinned at
`b127f62dc5c45694afc9703be713bd9db834bee0699f1c31614e62729a721e10`.
This implementation note is written before operational tail collection.

## Collection

`tools/own-three-sdk-study --prefix` requires a clean exact Go 1.27.1 source
revision and a fresh phase directly under `runs/`. It first checks the exact
published original terminal receipt, all original phase bytes, the full source
dataset and unchanged model metadata/weights. It independently audits the
original prefix and retains that audit before deriving remaining identities.

The preexecution journal contains every missing policy/view identity in original
order, its complete input/source SHA, all three ordered part SHAs, the actual
model pins, the whole prior raw inventory and unchanged calibration selection.
The exact boundary is 75 original development observations for
`set-initial/ptq_ternary`, then four untouched 512-view cells. No completed
original observation or calibration view is sent to an operational model again.

The original storage object still defaults to 768 MiB. This separate mode uses
2 GiB, counting every original phase and all tail evidence. Before each call it
reserves 1 MiB for the capture plus 2 MiB for terminal evidence and checks that
at least 4 GiB of disk remains available. A terminal record retains actual tail
wall/CPU/lifetime RSS and the closed file inventory on success or failure.
The tail preserves the original eight candidates, one candidate per advance,
seven feedback rounds, empty seed and absent CI hint.

```sh
GOTOOLCHAIN=go1.27.1 go run ./tools/own-three-sdk-study \
  --dataset runs/own-three-composition-curriculum-fixed-20261002/dataset.jsonl \
  --models models/own-three-feedback-v1 \
  --prefix runs/own-three-sdk-study-20261002 \
  --output runs/own-three-sdk-storage-tail-20261002 \
  --source-revision <exact-clean-published-source>
```

## Independent reconciliation

`--combined-audit-report` runs no model inference. It independently replays the
original audit, validates the full preexecution identity list, closed original
and tail byte inventories, and every retained prior raw file. It streams each
original cell, then only its missing tail rows. Exact input/source/policy order
rejects overlap, omissions, duplicates and post-hoc replacements. Every actual
arithmetic result, full failure context, probability, frontier and progress
receipt is independently checked. Cell curves and times are recomputed from
actual original and tail observations, including the split 75/437 cell.

Only 11,264 unique sessions and all 22 complete 512-view cells can produce
`PASS_WITH_SEPARATE_STORAGE_AMENDMENT`. The original attempt remains
`FAILED_PREFIX_RETAINED`, with its missing collector CPU/RSS marked unavailable.
The tail's measured resource costs are reported separately. No optimizer update,
native generation, compiler default promotion or general-language completeness
is implied by this finite comparison.

Local race tests and vet cover actual native fixture cells split across two
streams, exact reconciliation with a single full stream, unchanged original
storage bounds, terminal reserve, and rejection of repeated/changed identities,
parts, models, selection, seed, CI hint and resource amendment. These local
validation calls are separate from the 2,485 operational tail sessions.
