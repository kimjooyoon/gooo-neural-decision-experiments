# Superseded pre-inference draft

These exact bytes are preserved for audit history and are not part of the corrected cohort or any scored denominator.

- `cohort.jsonl` SHA-256: `650130fbca5b1731370b2b0aeb2d00267e0545db11452a2ae67a4b6e368d1730`
- `manifest.json` SHA-256: `c454f96bece12e069fee5df0d749c55949a85492820092bdbb1fefae2e1b7ae1`
- `generate.go.txt` SHA-256: `618eb76063be986c47c9d766d3c9dc158666c2919807f9c0893fd03f22799847`

The manifest binds the archived generator and cohort hashes. The draft omitted hole `text` and `fallback` because canonical-intent normalization mutated a shared expression slice. Its heldout input ordering also selected only negative values. It was identified before any model request, compilation-based score, or GPU run; it is a pre-inference draft, not a failed measured intent.

An earlier intermediate version was observed by the root task with cohort SHA-256 `56b43b9845ad45dcbdaa230e797228b54d998810ed9c5a35d2d198b07b141956` and generator SHA-256 `1273ccedbbbd84d6fb3ecfbc2921c73dc1cad4b2508ad59e6851f690c153ac22`. Those bytes were not retained. No claim is made that the current archive preserves that earlier intermediate.
