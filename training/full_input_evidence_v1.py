"""Storage accounting within the offline full-input optimizer process."""
from pathlib import Path
import shutil

from shared_evidence_v1 import Evidence as BaseEvidence, core


def inventory(root):
    prior, current = 0, 0
    for phase in (root / "runs").glob("own-three-*"):
        core.require(phase.is_dir() and not phase.is_symlink(), "regular raw phase")
        size = 0
        for path in phase.rglob("*"):
            core.require(not path.is_symlink(), "no raw evidence symlinks")
            if path.is_file():
                size += path.stat().st_size
            else:
                core.require(path.is_dir(), "regular evidence entries")
        if phase.name.startswith("own-three-full-input-"):
            current += size
        else:
            prior += size
    return prior, current


class Evidence(BaseEvidence):
    def __init__(self, root, output, preflight, revision):
        self.root, self.output = root.resolve(), output.resolve()
        core.require(not output.exists() and self.output.parent == self.root / "runs" and
                     self.output.name.startswith("own-three-full-input-"), "fresh optimizer phase")
        core.require(preflight["schema"] == "gooo/full-input-storage-preflight/v1" and
                     preflight["status"] == "PASS" and preflight["source_revision"] == revision and
                     preflight["future_phase"] == self.output.name and
                     preflight["optimizer_updates"] == preflight["model_predictions"] == 0 and
                     preflight["new_study_cap_bytes"] == 768 << 20 and
                     preflight["whole_own_three_cap_bytes"] == 3 << 30 and
                     preflight["failure_reserve_bytes"] == 16 << 20,
                     "exact Go storage preflight")
        prior, current = inventory(self.root)
        core.require((prior, current) == (preflight["prior_own_three_bytes"],
                                         preflight["existing_full_input_bytes"]),
                     "retained evidence still matches Go preflight")
        self.legacy = prior + current
        self.cap, self.whole_cap = (768 << 20) - current, 3 << 30
        self.used, self.actual_updates = 0, 0
        core.require(shutil.disk_usage(self.root).free >= 4 << 30, "four GiB free before MPS")
        self.reserve(0)

    def reserve(self, size, failure=False):
        margin = 0 if failure else 16 << 20
        core.require(size >= 0 and self.used+size <= self.cap-margin and
                     self.legacy+self.used+size <= self.whole_cap-margin,
                     "768-MiB study / 3-GiB total bounds; preserve prefix")

