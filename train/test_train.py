import tempfile
import unittest
from pathlib import Path

from train.train import load_batches


class LoadBatchesTest(unittest.TestCase):
    def test_skips_unrelated_csv_exports(self):
        with tempfile.TemporaryDirectory() as directory:
            folder = Path(directory)
            (folder / "metrics.csv.csv").write_text('"Metric","Value"\n"Events","10"\n')
            (folder / "training.csv").write_text(
                "schema_version,event_sequence,time_gap_buckets,purchased,returned_within_7d,prior_visits\n"
                "1,entry_point|purchase_confirmed,1,1,,\n"
            )

            batches = load_batches(folder)

        self.assertEqual(len(batches), 1)
        self.assertEqual(batches[0][0][0], ("entry_point", "purchase_confirmed"))

    def test_fails_when_no_supported_training_csv_exists(self):
        with tempfile.TemporaryDirectory() as directory:
            folder = Path(directory)
            (folder / "metrics.csv.csv").write_text('"Metric","Value"\n"Events","10"\n')

            with self.assertRaisesRegex(ValueError, "No supported training CSV"):
                load_batches(folder)


if __name__ == "__main__":
    unittest.main()
