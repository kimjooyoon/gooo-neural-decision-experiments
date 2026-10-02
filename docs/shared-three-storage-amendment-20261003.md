# Shared local judgment: storage correction before the first optimizer update

The first invocation at source `6a5dd2ea393275c27feca20f6f3b1e7b6fed3c17`
failed in `base.retained_bytes(root)` before writing its preexecution receipt,
creating an output directory or calling the optimizer. Its actual update count
is zero. The private original traceback is retained; its public digest and
sanitized failure facts are in `preexecution/shared-three-storage-failure-20261003.json`.
This failed attempt is not relabeled as a successful training run.

The original shared-judge protocol incorrectly put its new 64-MiB allowance
inside the earlier 768-MiB study cap. The original three-choice SDK study had
already stopped at that cap and subsequently used the separately published
`own-three-choice-sdk-storage-continuation-preregistration-20261002.md`, which
allows 2 GiB for preserved original evidence, the tail and native observations.
The new Go preflight finds **990,581,179 bytes** in eight retained original
phases, hashes every regular file, and records a per-phase tree digest. This
explains the failed old-cap check; it was not physical disk exhaustion.

This amendment is published before any shared-judge optimizer update. It
replaces only that experiment's storage clause: the new experiment has **64 MiB
of additional uncompressed evidence within 2 GiB total**, counting the original
990,581,179 bytes and all new training, audit and native observations. Require
4 GiB free disk before starting. Neither the original 768-MiB outcome nor its
later continuation is changed. Original evidence and model files are retained.

The optimizer uses one fresh output directory and records its storage preflight
and both protocol hashes. It counts each journal write instead of scanning all
old evidence after every update, reserves one MiB for a terminal failure record,
and reserves model export space before export. It reconciles actual new file
sizes after exports and at termination. A separate Go scan must confirm all
eight original phase digests after training. The existing trainer's writers are
temporarily supplied by this sequential offline caller and restored on exit;
the original trainer and its 768-MiB default are unchanged.

The two architectures, frozen source/intent features, initialization, splits,
3,200 updates, calibration selector, six exports, 288 parity predictions,
3,072 development predictions and 96 immediate generation/runtime pairs all
remain as originally declared. There is no repeated completed update, refreshed
target, extra seed, new data split, accuracy claim or default promotion. Preserve
any new failure prefix and its actual post-step journal counter; do not retry it
to manufacture success. Python remains offline optimization/export only; the
storage inventory, model audits and actual Gooo execution are Go programs.
