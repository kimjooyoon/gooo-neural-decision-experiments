# Integrated body stream: installed main verification

2026-10-03. [Main PR #1181](https://github.com/kimjooyoon/meta-ontology-go/pull/1181)
was normally squash merged as `cb2892cb583e693190bd68f75eb4df223819f053`.
The clean Go1.27.1 build uses SDK v0.2.20-experimental and is installed locally.
`gooo body-path-stream` now provides the retained worker through the main CLI.

Two identical public Korean requests were submitted with the original 16 KiB
model and again without a model. **Four generations, two model predictions,
eight compiled executions and 32/32 finite expectations** passed. Full generated
source matched the earlier local CLI observation. The first model request has
`reused:false`; the second has `reused:true`. Both perform fresh model predictions.
The deterministic requests perform zero predictions. No weights changed.

[Run the two-request example](../../examples/whole-candidate-order).
The stream emits JSON results when ready; callers use correlation IDs and
inspect each request's status. An ordinary end of input can exit successfully
even when an individual request was rejected. Setup messages are on stderr.
Before-EOF delivery, cancellation and bounded worker behavior were covered by
the shared runner tests and [earlier worker observations](../prepared-worker-native-20261003).
This four-generation installation check measures correctness on known examples;
it does not establish a throughput or new-request generalization result.

## Promotion evidence

- Dev source: `e9d1fd0f87fc1aa95756fd55293421051f6f3650` (#1180 merged).
- Main candidate: `2057cce103dcb00416335e71735b29b6d91673ae`.
- Previous main and candidate's sole parent: `8117fbaefac490a28e78c956bf1d40aed1372608`.
- Dev, candidate and installed main tree: `57570d8a9cb43a838b645d17af399068e2423bf7`.
- [CI 37105566652](https://github.com/kimjooyoon/meta-ontology-go/actions/runs/37105566652),
  attempt 1, all six canonical jobs and whole workflow PASS.
- Immutable artifact `11268217020`, `ci-proof-37105566652-1`.
- ZIP SHA256 `5d81fe1008290fbb390b37671d24f0b3b65858d1c24763316f9795e94101b658`
  matched GitHub metadata before extraction.
- Independent proof verification PASS; bundle digest
  `041b505f3168917dcc08d8f85ff0b5bbad57e8dd6822a164cccf15ea0f9716ad`.
- Live source refs, tree, sole parent, open non-draft clean mergeable PR and
  the source/run/attempt tuple were checked before the normal merge. The
  promotion authorization is bound to the same proof digest.
- Installed binary SHA256
  `a884dc2f7a52969c7c8fe0077d15d20cefb2b42accce89b954dcddda1f8d77b7`.

`native.zip` preserves every generation, native report, source comparison,
setup message and summary. Public input files are included separately. Build
information uses a basename instead of the local binary path. Verify all
publication files with `shasum -a 256 -c SHA256SUMS`.
