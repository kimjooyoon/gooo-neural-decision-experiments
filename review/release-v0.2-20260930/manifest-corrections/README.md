# Pre-publication manifest size correction

The independent final review found one incorrect derived file size in the
initial path-neutral attempt manifest. The file
`ptq-threshold-check-revision2/source-and-input-sha256.txt` is 808 bytes; the
initial manifest recorded 837. Its recorded SHA-256 was already correct.

`derived-size-initial.json` retains the complete initial manifest unchanged,
with SHA-256
`839c31d859ceaee233b594c1e0de7bd17c2958c676cc0453f52221f802fd28ed`.
The corrected [manifest](../attempts/derivation-manifest.json) changes only that
size and has SHA-256
`b97b5a713fcb24929d2d12be3a28b14899d5cbf8459c2391ba30a810c7669130`.

No raw attempt, derived capsule, release artifact, verifier source or execution
receipt was changed by this metadata correction. The initial manifest had
not been committed or published when the discrepancy was found.
