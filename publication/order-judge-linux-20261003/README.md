# Whole-candidate judge: Linux replay

[CI run 37097293126](https://github.com/kimjooyoon/gooo-neural-decision-experiments/actions/runs/37097293126)
ran the new `whole-candidate-judge` job at clean source
`8a41bb0f825dfd3f950101b33491581492c740f6` with Go1.27.1 Linux/amd64.
That job succeeded and published immutable artifact **11264582420**, retained
here as `evidence.zip`. Its GitHub digest and downloaded SHA256 both equal
`99eb85c376cad4d875d78a9b183e88580bcccabaa77af5ceec4adddd536fb350`.

The job independently reconstructed 160 original requests, executed 960 bounded
searches with 480 actual model predictions, and compared 320 baseline SDK searches.
The summary matches the local arm64 follow-up exactly. Downloaded detailed records
also match exactly after removing only `SearchNS` and `Ranking.predict_ns`:

```sh
jq -S 'map(del(.SearchNS) | .Ranking |= del(.predict_ns))' records.json
```

Both canonical record files have SHA256
`f0f6864734aed872cd84766979abde26733f12b505072db0dda6078f9398377f`.
This compares predictions, descriptor identities, every choice, interpreter case
result and equivalence skip. Zero training updates and zero native compiler runs
were made in this replay. It is a portability check of the observed cohort.

See [the unchanged initial fit and local follow-up](../order-judge-initial-20261003)
for results, costs, protocol and reproduction. The broader workflow retains a
failure in the historical `full-input-initial-replay` arithmetic comparison;
this new job's success does not imply the entire research workflow passed.

The new model is public at
[asketeddy/gooo-order-judge-tiny-v1](https://huggingface.co/asketeddy/gooo-order-judge-tiny-v1/tree/f94b77da30677f4f6d4a348d38c08124e6c2d9e3).
Its public weights, metadata and card were downloaded at that revision and matched
the local uploaded bytes. The weights are identical to the original fit.
