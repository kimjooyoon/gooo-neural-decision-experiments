# Independent generated-Go replay — revision 2

This replay independently compiled and executed the six validated tiny-go outputs from the frozen capture run. It used compiler source commit `e7dc8198600f27fa3d6fc07d2a8fd8b6a4af3b27`, native binary SHA-256 `5e7f75badf8079d15d971bc05f65c08516a9b79c4bb0539036ccb318c093f79c`, run-plan SHA-256 `1c5a86b321eacf9549d03f4b7df2c94f8ea8171b7dbb04f384094895869c3a58`, and `go version go1.27.0 darwin/arm64`.

The replay plan retains all eight cells and the fixed 48-case denominator. Six tiny-go cells compiled and executed successfully: 6/8 cells compiled, 6/8 completed observed validation, 36/48 cases observed and all 36 passed. The two offline baseline captures were omitted from this replay input because their original wrappers recorded known completeness-validator errors; their twelve cases remain unknown here. Their raw captures remain in the capture directory for separate audit. The replay did not modify those captures.

All six tiny-go cases were the predeclared finite candidate-scoring suite. This is reused evaluation material, not a holdout or a full-domain correctness claim. Generated activity bodies were compiled and executed with a small independent Go `main` harness; the body bytes were preserved.

Across the six generated programs, Go build wall time ranged from 125.033 ms to 194.856 ms (913.310 ms total), and generated execution wall time ranged from 378.505 ms to 452.218 ms (2458.846 ms total). These are six sequential observations without repetition; they are not latency benchmarks.

The original `independent-replay.json` retains the eight-cell view: `status=PARTIAL`, 36 observed, 0 failed, and 12 unknown.
