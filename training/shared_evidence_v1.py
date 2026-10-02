"""Bounded journals for one offline shared-judge optimizer invocation."""
from contextlib import contextmanager
import json
from pathlib import Path
import shutil

import train_own_three_feedback_v1 as base

core = base.core


class Evidence:
    def __init__(self, root, output, preflight):
        self.root, self.output = root.resolve(), output.resolve()
        self.legacy = preflight["legacy_bytes"]
        self.cap, self.whole_cap = 64 << 20, 2 << 30
        self.used, self.actual_updates = 0, 0
        core.require(preflight["schema"] == "gooo/shared-three-storage-preflight/v1" and
                     preflight["status"] == "PASS" and preflight["optimizer_updates"] == 0 and
                     preflight["new_phase_cap_bytes"] == self.cap and
                     preflight["amended_whole_cap_bytes"] == self.whole_cap and
                     sum(p["bytes"] for p in preflight["phases"]) == self.legacy and
                     0 < self.legacy <= self.whole_cap-self.cap, "published Go storage preflight")
        core.require(not list((self.root / "runs").glob("own-three-shared-*")),
                     "no existing shared optimizer prefix may be restarted or omitted")
        core.require(self.output.parent == self.root / "runs" and
                     self.output.name.startswith("own-three-shared-"), "single bounded new phase")
        core.require(shutil.disk_usage(self.root).free >= 4 << 30, "at least four GiB available")

    def reserve(self, size, failure=False):
        margin = 0 if failure else 1 << 20
        core.require(size >= 0 and self.used+size <= self.cap-margin and
                     self.legacy+self.used+size <= self.whole_cap-margin,
                     "new 64-MiB and amended 2-GiB caps; preserve prefix")

    def path(self, path):
        path = Path(path)
        core.require(path.resolve().is_relative_to(self.output), "optimizer output scope")
        core.require(not path.is_symlink() and all(not p.is_symlink() for p in path.parents),
                     "no optimizer evidence symlinks")
        return path

    def write(self, path, value, append=False, pretty=False):
        path = self.path(path)
        raw = (json.dumps(value, ensure_ascii=False, sort_keys=True,
                          indent=2 if pretty else None, allow_nan=False)+"\n").encode()
        core.require(len(raw) <= 1 << 20, "one-MiB record maximum")
        self.reserve(len(raw), failure=path.name == "failure.json")
        core.require(path.is_file() if append else not path.exists(), "immutable output or existing journal")
        path.parent.mkdir(parents=True, exist_ok=True)
        with path.open("ab" if append else "xb") as out:
            written = out.write(raw)
            self.used += written
            core.require(written == len(raw), "complete journal write")

    def save(self, path, value, root):
        core.require(root.resolve() == self.root, "storage root binding")
        self.write(path, value, pretty=True)

    def save_jsonl(self, path, rows, root):
        core.require(root.resolve() == self.root, "storage root binding")
        for index, row in enumerate(rows):
            self.write(path, row, append=index > 0)

    def record_update(self, path, value, root):
        core.require(root.resolve() == self.root, "storage root binding")
        # The frozen fit loop calls this immediately after the actual optimizer step.
        self.actual_updates += 1
        self.write(path, value, append=value["actual_stage_updates"] != 1)

    def reconcile(self):
        total = 0
        if self.output.exists():
            for path in self.output.rglob("*"):
                self.path(path)
                if path.is_file():
                    total += path.stat().st_size
                else:
                    core.require(path.is_dir(), "regular evidence entries")
        core.require(total >= self.used, "no previously counted evidence removed")
        self.used = total
        self.reserve(0, failure=True)

    @contextmanager
    def writers(self):
        # Sequential offline process only; original trainer defaults stay unchanged.
        names = ("save", "save_jsonl", "record_update")
        previous = {name: getattr(base, name) for name in names}
        try:
            for name in names:
                setattr(base, name, getattr(self, name))
            yield
        finally:
            for name, value in previous.items():
                setattr(base, name, value)
