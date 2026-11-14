# Global intelligence training

Download the training CSV from Intelligence in Eye. The sidebar tenant selects the export scope. Share production batches privately. The export contains normalized sequences and coarse timing, which can still be identifying when combined with other data.

By default, the workflow reads `train/data/*.csv` from [arashrasoulzadeh/eye](https://github.com/arashrasoulzadeh/eye). Submit synthetic or privacy-reviewed public batches there. The default source needs no training-data token.

To use another repository, set the Actions variable `TRAINING_DATA_REPOSITORY` to its GitHub URL or `owner/repo`. Place the reviewed CSVs in that repository's `train/data/` folder. Public repositories can be cloned without credentials. For a private repository, set the Actions secret `TRAINING_DATA_TOKEN` to a token with read access.

Run **Train global intelligence model** in GitHub Actions. At least two distinct batches with event transitions are needed. Keep distinct contributors in separate batches and avoid overlapping exports. One batch is held out for next-event evaluation. The optional temporal CNN is distilled into a context lookup for CPU-light local inference.

The workflow publishes a versioned `eye-global-model.json` release asset. Eye polls GitHub Releases, verifies the asset digest, and caches the bundle locally. Successful inference is recorded per tenant in Intelligence. Models and their outputs stay available locally during download outages.
