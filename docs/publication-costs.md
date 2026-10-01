# Cheaper frozen-evidence publication

The bounded publication scanner preserves the original rejection expression.
Each alternative needs one of seven literal byte prefixes. If no prefix exists,
the full regex cannot match; suspected token text still uses the complete original
expression. The two directory alternatives reject directly. No source paths,
tokens or additional files enter the public allowlist.

On one Apple M4 with Go 1.27.1, the synthetic 1,160,000-byte safe-text benchmark
observed 36.6–37.7 ms for the original regex and 1.001–1.003 ms for necessary-prefix
checking. All three prefix repetitions allocate zero bytes. The test compares
2,752 bounded positive/negative combinations with the original expression,
including whitespace, short tokens, case variants, Korean text and later matched
alternatives. Suspected-prefix text still incurs regex work; it is not claimed faster.

The full frozen-inventory package race test took 84.051 seconds before and 2.007
seconds after in local runs. These runs have different cache/order conditions;
their difference does not establish a causal full-CI speedup. Repacking the native
feature appendix preserved the exact manifest and compressed payload hashes, and
all 104 raw entries passed verification. Model weights, ranking, native codegen
and previous publication bytes are unaffected.

[Raw descriptive numbers](../publication/unfixed-publication-prefix-microbenchmark-20261001.json)
and the [reproducible benchmark](../tools/package-family-evidence/privacy_test.go)
have narrower scope than the native model/process measurements. They add zero
predictions, native calls, generated-program executions, optimizer steps or GPU work.
