# Complete split and wording audit

This tool currently implements the **model-free preparation and preparation
verification stage** of the [registered protocol](../../docs/full-input-all-forms-protocol-20261003.md).
It reconstructs all 3,072 frozen views, produces eleven explicitly named input
forms, retains complete text and source/target identities, and indexes full-feature
digests per representation and form. Each duplicate group's member rows retain
its valid-mask intersection and every source/input identity. The separate frozen
operation-order alias remains in `order-control.json`.

Run from a clean published checkout with Go 1.27.1 and the original frozen
dataset at its registered `runs/` location:

```sh
go run ./tools/audit-full-input-forms --mode prepare \
  --source-revision "$(git rev-parse HEAD)" \
  --output runs/own-three-full-input-forms-20261003

go run ./tools/audit-full-input-forms --mode verify-prepare \
  --source-revision "$(git rev-parse HEAD)" \
  --output runs/own-three-full-input-forms-20261003
```

Preparation makes zero model calls and zero optimizer updates. It reports actual
accepted/declined rows, collisions, wall time, process CPU seconds and process
peak RSS. The verifier regenerates inputs and collision groups using this same
source and checks the complete five-file preparation inventory. It does not
claim independent prediction replay.

The fixed storage reservation includes 224 MiB raw evidence (16 MiB reserved for
failure) and 64 MiB public evidence, within the existing 768 MiB full-input study
and 3 GiB own-three limits. Rows are capped at 64 KiB. Metadata and journals use
exclusive creation; a failed prefix is preserved. A collection never resumes or
retries automatically.

Controlled tests cover complete-input forms, source coordinates, stable ranking,
partial and full finite coverage, bilingual outcomes, passing-set NLL, confidence
bins, Brier/ECE summaries, collision membership, file limits and storage reserves.
Prediction collection and the separate independent reader are the next stage;
their source must be published before the protocol's model calls begin. Planned
calls are never added to measured preparation counts.
