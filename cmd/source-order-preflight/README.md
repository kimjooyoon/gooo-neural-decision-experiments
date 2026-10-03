# Source order preflight

Protocol: four previously published arithmetic families × two source orders ×
two instruction languages × three presentations (base, renamed local, commuting
operand swap). The intent always requests the first published order. Expected
selection cases are copied unchanged; context export never executes them.

Every request goes through `gooo body-context --include-plan` with V3 context.
The collector checks source binding, raw source identity, expanded-plan identity,
and zero model calls/tests/writes, then independently validates the typed plan.
The existing V3 local root input is compared with the new 48-byte descriptor per
alternative. A pair of alternatives occupies 96 bytes, excluding compiler ASTs,
JSON, validator and collector memory. All raw sources, recipes and exports remain
available. No native execution or model learning happens in this experiment.

The descriptor covers `let v=input; v=op(...); v=op(...); return v`. Operators are
integer add/subtract/multiply; operands are the current local, input or int64
literal. Constants retain all 64 bits. Add/multiply operands are canonically
ordered, and local names are omitted. Nested expressions, branches, extra locals
and holes decline. Source order is described structurally; this is not a complete
program equivalence check. For example, adding 1 then 2 versus 2 then 1 has
different descriptors despite equal outputs. Other choices are held at the base
structure; a later joint-mask model must account for interacting alternatives.

The representation is experimental and does not overwrite frozen V3/V4 inputs.
Run with Go1.27.1 from a clean pinned checkout:

```sh
go run ./cmd/source-order-preflight --compiler /path/to/gooo \
  --compiler-sha <revision> --collector-sha <revision> \
  --go /path/to/go1.27.1 --out /path/to/new-output
```

The encoder microbenchmark excludes validation, parsing, process startup and
export. The export receipts measure a different boundary. Results describe these
authored cases; instruction understanding and held-out performance require a
separately versioned model experiment.
