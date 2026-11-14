# Global intelligence training

Download the training CSV from Intelligence in Eye. The sidebar tenant selects the export scope. Share production batches privately. The export contains normalized sequences and coarse timing, which can still be identifying when combined with other data.

Place reviewed CSVs in `train/data/` in a private GitHub repository. Set `TRAINING_DATA_REPOSITORY` to its owner and name in the Eye repository Actions variables, and give `TRAINING_DATA_TOKEN` read access to that repository. The public Eye repository's `train/data/` directory is suitable for synthetic fixtures.

Run **Train global intelligence model** in GitHub Actions. At least two distinct batches with event transitions are needed. Keep distinct contributors in separate batches and avoid overlapping exports. One batch is held out for next-event evaluation. The optional temporal CNN is distilled into a context lookup for CPU-light local inference.

The workflow publishes a versioned `eye-global-model.json` release asset. Eye polls GitHub Releases, verifies the asset digest, and caches the bundle locally. Successful inference is recorded per tenant in Intelligence. Models and their outputs stay available locally during download outages.
