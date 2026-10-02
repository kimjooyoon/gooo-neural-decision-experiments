# Three-choice authored fixture verification

The [frozen protocol](own-three-choice-completeness-preregistration-20261002.md)
predates this implementation and any model observation. The new
`internal/threecompositionstudy` package implements its eight authored families:

| Family | Ordered editable targets |
| --- | --- |
| Chained arithmetic | Three successive subtraction operand orders |
| Successive assignments | Two assignment targets, then a local return reference |
| Conditional assignment | Branch layout, one branch's assignment target, final reference |
| Assignment and read | Assignment target, later local reference, subtraction order |
| Dependent execution | Root write order, dependent assignment target, subtraction order |
| Nested conditions | Outer branch layout, inner branch layout, final reference |
| Comparison and write | Comparison operand order, branch layout, branch assignment target |
| Boolean control | Boolean local reference, branch layout, final integer operand order |

Configurations 0–23, goal masks 0–7 and Korean/English wording produce 3,072
function views in 1,536 bilingual groups. The split remains 2,048 training,
512 calibration and 512 development views. These are eight authored ways,
not 3,072 independent experiment designs. Source fallback masks depend only on
configuration. Instructions describe source edits or typed local positions;
option labels and goal masks are excluded from model input text.

## Completed checks

Every view enumerates eight type-valid alternatives and 16 ordered signed-int64
cases. Typed evaluation agrees with an independently written ordinary Go oracle
in all 393,216 comparisons. The cases retain int64 extremes, both sides and
equality at conditional thresholds; repeated input values remain ordered cases.
Sixteen separately hand-calculated examples cover every family.

878 views / 439 bilingual groups have more than one complete passing mask.
All passing masks are retained with normalized joint target mass. Coordinate
marginals are diagnostic; they do not replace the complete passing set.
All three canonical source-v3 inputs and the complete framed input fit the
declared bounds without truncation. Literal/name/language/goal variations do
not alter source-feature facts for fixtures with the same normalized structure.

A separate ordinary Go compilation executes 128 emitted functions (all families,
all masks, configurations 0 and 1) against 2,048 oracle values. It covers both
less-or-equal and equality comparison families. This is SDK fixture execution,
separate from the declared 640 actual native Gooo generations in the later study.
Local targeted race tests and vet pass with Go 1.27.1.

The first fixture check rejected an eagerly allocated unused threshold expression
in arithmetic-only families. Threshold expressions are now created only when
referenced; the closed body validator and all declared families/cases remain.
The negative check precedes collection and is not a training example.

## Phase boundary

This is authored source/oracle preparation. It makes zero model predictions and
zero optimizer updates. Actual native exports, concrete source/document/input/
dataset hashes, corpus freeze and independent capture audit must still complete
before teacher observation or training. No trained three-choice quality,
natural-language generalization or total compiler speedup is established.
