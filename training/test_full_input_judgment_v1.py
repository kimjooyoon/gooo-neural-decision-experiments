"""Offline bank weighting/storage tests; no optimizer or MPS operation."""
from pathlib import Path
import tempfile
import unittest

import numpy as np
from full_input_evidence_v1 import Evidence, inventory
from train_full_input_judgment_v1 import arm_data


class FullInputTests(unittest.TestCase):
    def test_original_weights_and_unchanged_calibration(self):
        rows = []
        for split, groups in (("train", 1024), ("calibration", 256), ("development", 256)):
            for group in range(groups):
                for language in ("en", "ko"):
                    phases = ("initial", "feedback") if split == "train" else ("initial",)
                    forms = ("original", "request-prefix") if split == "train" else ("original",)
                    for phase in phases:
                        for form in forms:
                            weight = .5 / len(phases)
                            rows.append({"feature_row_index": len(rows), "split": split,
                                         "program_contract_group": f"{split}/{group}", "language": language,
                                         "phase": phase, "form": form, "finite_soft_targets": [1., 0., 0., 0., 0., 0., 0., 0.],
                                         "original_feedback_arm_row_weight": weight,
                                         "feedback_arm_row_weight": weight/len(forms),
                                         "initial_arm_row_weight": 0.})
        x = np.arange(len(rows)*2, dtype=np.float32).reshape(len(rows), 2)
        original = arm_data(rows, x, "positioned-original")
        varied = arm_data(rows, x, "bag-varied")
        self.assertEqual(len(original["train"][0]), 4096)
        self.assertEqual(len(varied["train"][0]), 8192)
        np.testing.assert_array_equal(original["train"][0], varied["train"][0][::2])
        np.testing.assert_array_equal(original["train"][2], varied["train"][2][::2]*2)
        for index in range(3):
            np.testing.assert_array_equal(original["calibration"][index], varied["calibration"][index])
        self.assertTrue(all(row["initial_arm_row_weight"] == 0. for row in rows))

    def test_storage_journal_scope_and_freshness(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary).resolve()
            (root / "runs/own-three-old").mkdir(parents=True)
            (root / "runs/own-three-old/bytes").write_bytes(b"old")
            (root / "runs/own-three-full-input-bank").mkdir()
            (root / "runs/own-three-full-input-bank/bytes").write_bytes(b"new")
            output = root / "runs/own-three-full-input-training"
            revision = "a"*40
            preflight = {"schema": "gooo/full-input-storage-preflight/v1", "status": "PASS",
                         "source_revision": revision, "future_phase": output.name,
                         "optimizer_updates": 0, "model_predictions": 0,
                         "new_study_cap_bytes": 768 << 20, "whole_own_three_cap_bytes": 3 << 30,
                         "failure_reserve_bytes": 16 << 20,
                         "prior_own_three_bytes": 3, "existing_full_input_bytes": 3}
            self.assertEqual(inventory(root), (3, 3))
            evidence = Evidence(root, output, preflight, revision)
            evidence.save(output / "preexecution.json", {"updates": 0}, root)
            with self.assertRaises(ValueError):
                Evidence(root, output, preflight, revision)
            with self.assertRaises(ValueError):
                evidence.save(root / "outside.json", {}, root)
            evidence.reconcile()
            evidence.used = evidence.cap - (16 << 20)
            with self.assertRaises(ValueError):
                evidence.reserve(1)
            evidence.save(output / "failure.json", {"status": "retained"}, root)
            self.assertLessEqual(evidence.used, evidence.cap)
            preflight["future_phase"] = "own-three-full-input-other"
            with self.assertRaises(ValueError):
                Evidence(root, root / "runs/own-three-full-input-other", preflight, revision)


if __name__ == "__main__":
    unittest.main()
