# Joint-path v1 pre-implementation parameter correction

The original protocol is retained at commit
`244f57eb6505f8ae049e5d2af63c1b23253f0171`, SHA256
`03655ff443402b3e7d1b59fa780d4c8e68b750e1093b504a61de3fbf691d6a28`.

Before any new fixture, oracle, collection or training implementation, arithmetic
inspection found that `2+((5*c+1) mod 5)` is always three. The effective scale
formula is **s=2+((2*c+1) mod 5)**, giving all five values 2–6. Delta, threshold,
fallback masks, families, source ABI, splits, input cases, model architectures,
optimizer budget and selector follow the original protocol.

No model predictions, native collection calls or optimizer updates have occurred
for this new joint cohort. This is a definition correction before observation.
Collection/training/publication provenance must bind both this amendment and
the original frozen protocol. Old source-v3 data and results remain historical
evidence outside this new cohort.
