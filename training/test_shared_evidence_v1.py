"""Offline optimizer journal checks; no model forward or optimizer step."""
import tempfile
from pathlib import Path
import unittest

from shared_evidence_v1 import Evidence, base


class EvidenceTests(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.root = Path(self.directory.name).resolve()
        (self.root / "runs").mkdir()
        self.output = self.root / "runs/own-three-shared-test"
        self.preflight = {"schema": "gooo/shared-three-storage-preflight/v1", "status": "PASS",
                          "legacy_bytes": 990581179, "optimizer_updates": 0,
                          "phases": [{"bytes": 990581179}],
                          "new_phase_cap_bytes": 64 << 20, "amended_whole_cap_bytes": 2 << 30}
        self.evidence = Evidence(self.root, self.output, self.preflight)

    def test_journal_and_restore_after_exception(self):
        original = base.save, base.save_jsonl, base.record_update
        with self.assertRaisesRegex(RuntimeError, "forced"):
            with self.evidence.writers():
                base.save(self.output / "pre.json", {"a": 1}, self.root)
                for count in (1, 2):
                    base.record_update(self.output / "updates.jsonl", {"actual_stage_updates": count}, self.root)
                base.save_jsonl(self.output / "parity.jsonl", [{"a": 1}, {"a": 2}], self.root)
                raise RuntimeError("forced")
        self.assertEqual(original, (base.save, base.save_jsonl, base.record_update))
        before = self.evidence.used
        self.evidence.reconcile()
        self.assertEqual(before, self.evidence.used)
        self.assertEqual(self.evidence.actual_updates, 2)
        with self.assertRaises(ValueError):
            self.evidence.save(self.output / "pre.json", {}, self.root)

    def test_caps_and_failure_reserve(self):
        self.evidence.used = self.evidence.cap - (1 << 20)
        with self.assertRaises(ValueError):
            self.evidence.save(self.output / "normal.json", {}, self.root)
        self.evidence.save(self.output / "failure.json", {"failed": True}, self.root)
        self.assertLessEqual(self.evidence.used, self.evidence.cap)
        self.evidence.used = self.evidence.cap
        with self.assertRaises(ValueError):
            self.evidence.reserve(1, failure=True)

    def test_scope_symlink_and_existing_prefix(self):
        with self.assertRaises(ValueError):
            self.evidence.save(self.root / "outside.json", {}, self.root)
        self.output.mkdir()
        (self.output / "linked.json").symlink_to(self.root / "target.json")
        with self.assertRaises(ValueError):
            self.evidence.save(self.output / "linked.json", {}, self.root)
        with self.assertRaises(ValueError):
            Evidence(self.root, self.output, self.preflight)


if __name__ == "__main__":
    unittest.main()
