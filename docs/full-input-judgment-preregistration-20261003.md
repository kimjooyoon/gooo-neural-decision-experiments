# Full-input judgments: representation × phrasing

Status: preregistration before optimizer updates. This follows the fixed-weight
[wrapper diagnosis](bilingual-wrapper-audit-results-20261003.md). It addresses
the observed collapse under introductory wording while retaining complete input.

## Question and fixed comparison

Four freshly initialized shared 256→8→2 judges (2,072 parameters each) compare:

| Arm | Intent representation | Training phrasing |
| --- | --- | --- |
| positioned-original | frozen source-v3, four relative-position buckets | original source/teacher states |
| positioned-varied | frozen source-v3 | original plus four neutral wrapper forms |
| bag-original | source-v4, full-text byte-fragment counts | original source/teacher states |
| bag-varied | source-v4 | original plus four neutral wrapper forms |

V4 keeps the 64 source coordinates, 192 intent coordinates, normalization,
three-part framing, 768 total features and 3,200-byte caller workspace. It hashes
all UTF-8 2/3-byte intent fragments into 192 bins without relative-position
buckets. Its explicit feature version is
`triple_semantic_context_bag_v4_joint_v1`. Source-v3 remains byte-for-byte fixed.
This factor changes position encoding and hash collisions together. An effect
therefore belongs to the representation as a whole.

The complete caller text is always retained. There is no production prefix
removal, hidden shortening, automatic model replacement or altered finite oracle.
An unsupported feature version or oversized complete input follows the existing
zero-prediction deterministic continuation contract.

## Source data and split boundaries

Reuse the frozen 3,072-view Gooo corpus, dataset SHA256
`9a887dc09caf2f2b2b947641509328a2ee6f25dcefb6b52efe178fe8aff4fb3a`.
The disjoint program groups remain 1,024 training, 256 calibration, 256
development, with two language views per group. The previously audited teacher
bank has 10,739 initial/feedback states: 9,715 training states, 512 calibration
initial states and 512 development initial states. Reconstruct these in Go from
the complete source/teacher evidence and preserve every state identity and target.
Development has already been observed; results describe this known cohort.

Varied training has five equally weighted forms for each original training state:

| Form | English addition | Korean addition |
| --- | --- | --- |
| original | unchanged | unchanged |
| request-prefix | `Request: ` | `요청: ` |
| please-prefix | `Please follow this instruction: ` | `다음 지시를 따라 주세요: ` |
| request-suffix | ` This is the request.` | ` 이것이 요청입니다.` |
| please-suffix | ` Please follow this instruction.` | ` 이 지시를 따라 주세요.` |

Insert additions after the semantic source header, around the full intent and
any existing feedback. Preserve the header and original input alongside the new
text. Each original function/language/state weight is divided equally among its
representable forms. Record every rejected full form and its input bytes/hash;
include the original form in every state. Both representations use the exact
same accepted forms and weights. Per-state renormalization and its counts are
reported, since overflow may make exposure to forms unequal.

Calibration remains the original 512 complete views for all arms. No development
data or evaluation-only wording selects epochs, temperatures, weights or budgets.
New evaluation-only forms are `For this task, ` / `이 작업에서는, ` as prefixes
and ` That is the complete instruction.` / ` 이것이 전체 지시입니다.` as suffixes.
The five previously observed wrapper-study forms are also evaluated. New wording
over old source tasks is reported as a phrasing intervention.

## Initialization, optimization and quantization

Use the same fresh Go initializer recipe and bytes as the prior matched study,
with seed 20261031 and shuffle seed 20261032. The shared judge takes the same
initial local slices for every arm; these are random initialization, not learned
teacher or previous student weights. Preserve the exact initializer digest.

Each arm uses AdamW, learning rate 0.001 and weight decay 0.01, for 100 epochs,
8 batches of 128 function groups per epoch: 800 FP32 updates. QAT starts its own
calibration-selected FP32 checkpoint and makes 800 more updates with the same
budget. All four arms total **6,400 optimizer updates**. The calibration choice
is minimum (passing-set NLL, epoch, temperature), with temperatures 0.5, 1, 2, 4.
Function groups retain equal total weight; valid passing masks define the loss.

Keep the prior shared expanded-matrix QAT recipe, including structural zeroes,
constant across arms. Export FP32, PTQ and QAT for every arm. Compact conversion
must preserve all tied values and zeroes exactly; it adds zero optimizer updates.
Calibration alone chooses PTQ temperature. Retain all twelve exports and losses.

Offline PyTorch performs local MPS optimization/export. Go owns preparation,
features, inference, independent numerical/functional audits and native execution.
Each actual optimizer update is journaled immediately; failures retain their
prefix and are recorded before any explicitly described repair or continuation.
One Go launcher runs the four arms sequentially in a single MPS process, with a
30-minute process deadline and zero automatic retries. MPS cached allocations
are released between arms. Stage measurements retain their warm-process scope.
Numerical export parity uses 52 fixed vectors per export: the first 16 development
rows per language and first two training feedback rows per language/form. These
vectors are never inputs to checkpoint or temperature selection.

## Measurements and criteria

Report all arms and all three exports, per split, input form, language and family:

- first-path finite completion / views; finite cases matched / expectations;
- extra static ranked attempts and completion by budgets 1, 2, 4, 8;
- EN/KO both-valid pairs, different-but-valid pairs, same-wrong pairs and mask
  disagreements; equal masks alone do not establish meaning preservation;
- probability calibration and passing-set NLL with stable float64 arithmetic;
- rejected complete forms and actual exposure weights, source-feature equality,
  exact feature collisions whose finite valid-mask sets have empty intersection;
- complete preparation/training/audit wall time, process CPU, process peak RSS,
  sampled MPS allocated/driver memory, model tensor/file and workspace bytes;
- prediction-only and end-to-end native generation timings with their own scopes.

Add an order-sensitive representation control before training: swap two distinct
operations between identical clause delimiters, retaining identical 2/3-byte
fragment counts but different intended operation order. Show the corresponding
finite arithmetic outputs. A bag alias remains a known expressivity limit even
if aggregate development metrics improve. Preserve this negative result.

The main descriptive comparison is each varied arm against its original arm,
and V4 against V3 under equal phrasing treatment. Report regressions per family.
At least one full-input result must reduce the current 113/512 original-input
shortfall or its search burden before claiming practical progress. Model
selection from development is labeled exploratory; deployment requires a new
frozen native comparison and exact source binding. Broader capability discovery,
new task generalization and whole-language completeness remain separately scoped.

Before adoption, independently replay Go predictions against exported numerical
vectors, run actual Gooo code generation followed immediately by compiled
execution, and preserve completeness receipts and their unresolved frontier.
V3 regression evidence must still reproduce. Existing default model selection
remains in place until these observations are available.

## Resource and publication bounds

Reuse current checkouts, raw evidence and the MPS environment. No new worktree or
large pretrained download. Require at least 4 GiB free before preparation and
training. This study allows 768 MiB new retained local evidence across preparation,
optimization and audit, including a 16 MiB failure reserve. Bounded generated
feature banks may be compressed for publication; full texts and rejections stay
reconstructible from pinned source and journals. Inventory existing evidence
before work; total existing plus new own-three evidence may reach at most 3 GiB.
Stop at a bound while retaining the actual prefix; publish any amendment before
further data or optimizer work.

Publish source, all model variants, controlled results, negative controls,
reproduction commands and a closed digest inventory to the public research
repository and Hugging Face. Public packaging excludes private paths, credentials,
local environment directories and unrelated data. New documentation will clearly
separate trained-model quality, representation parity and native finite behavior.

## Reproduction sequence

Use Go 1.27.1 and the existing local MPS environment. Publish this protocol and
its source before running the commands. `SOURCE` is the clean full commit hash;
`MPS_PYTHON` is the absolute executable path. Keep output directories fresh.

```sh
go run ./tools/prepare-full-input-study \
  --source-revision "$SOURCE" --output runs/own-three-full-input-bank-20261003
go run ./tools/prepare-full-input-study \
  --source-revision "$SOURCE" --output runs/own-three-full-input-bank-20261003 \
  --verify-report runs/own-three-full-input-bank-20261003/replay.json
go run ./tools/prepare-full-input-study \
  --source-revision "$SOURCE" --output runs/own-three-full-input-training-20261003 \
  --storage-report runs/own-three-full-input-bank-20261003/training-storage.json
go run ./tools/train-full-input-study --python "$MPS_PYTHON" \
  --source-revision "$SOURCE" --prepared runs/own-three-full-input-bank-20261003 \
  --preparation-audit runs/own-three-full-input-bank-20261003/replay.json \
  --storage-preflight runs/own-three-full-input-bank-20261003/training-storage.json \
  --output runs/own-three-full-input-training-20261003
```

The storage report itself is included in the optimizer's retained-byte inventory.
Its writer accounts for that file before recording the future-phase preflight.
