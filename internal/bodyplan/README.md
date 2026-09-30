# Typed body plans

`bodyplan` compiles a bounded, closed typed plan into deterministic Go and
Gooo source and provides an interpreter for the same plan. Plans contain
integer and Boolean literals, one `input` integer, scoped locals, the eight
closed binary operations, typed operation holes, assignments, conditional
branches, and returns. They do not contain source snippets or operator text
that is copied into generated code.

Call `Plan.Holes()` to validate a plan and list its unique typed holes. Each
hole exposes a prompt string, operand/result types, an allowed operation list,
and an allowed deterministic fallback. `Compile(plan, choices)` requires an
exact choice for every hole, typechecks every allowed candidate against the
hole's operand types, and checks that all candidates preserve the same result
type. The resulting `Program` is an immutable snapshot and supports concurrent
`Evaluate` calls.

Expression children refer only to earlier expression indices. Statement
indices form a tree: every statement is reachable exactly once from `Root`,
and every expression is used. Lexical scopes are introduced by `if` branches;
locals cannot shadow a visible name, assignments must target an existing local,
and `input` cannot be assigned. Every control-flow path must return the
declared result type. The arenas are limited to 128 expressions, 128
statements, 16 holes, and combined expression/control-flow nesting depth 16;
hole text is nonempty valid UTF-8 up to 512 bytes.

`GoSource()` returns a formatted `package generated` function with an `int64`
input and an `int64` or `bool` result. `GoooSource()` returns `package
bodyplan`, the `bodyplan` namespace, Integer and Boolean entity declarations,
and an activity whose Gooo body is JSON-quoted. Gooo local declarations use
`let`; Go declarations use inferred `var`. `Evaluate` follows Go's wrapping
`int64` arithmetic and short-circuit Boolean operators.
