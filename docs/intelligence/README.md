# Intelligence and model lifecycle

The Intelligence workspace exports training batches, displays the installed and latest global model, downloads releases and shows the last successful inference and inference log. The legacy labels, models and training routes redirect to this workspace; they are not independent editors.

## Training workflow

1. Select the tenant scope and time range and download the training CSV.
2. Review normalized sequences and coarse timing for privacy. Share production batches privately.
3. Put reviewed batches in train/data/ in the training repository. The default is this repository. Set TRAINING_DATA_REPOSITORY for another repository and TRAINING_DATA_TOKEN for private access.
4. Run **Train global intelligence model** in GitHub Actions. Epochs range from 1 to 5000 (default 5000).
5. The trainer holds out one batch when multiple batches exist; with one CSV it holds out session sequences, keeping identical sequences together. A single file requires at least two distinct sequences with transitions.

The optional temporal CNN is distilled into a context lookup for CPU-light inference. The workflow publishes a versioned eye-global-model.json release asset. Eye verifies the digest and caches the bundle locally. Inference runs locally and remains available during download outages if a valid model is cached.

## Offline registry

Validated buyer artifacts use models/global/buyer/manifest.json and model.json. Tenant overrides use models/tenants/<tenant-id>/buyer/. The server verifies manifest checksums. Tenant artifacts override global artifacts.

Publish with deploy/publish-model.sh and update the model volume with deploy/update-models.sh. The update validates artifacts and backs up the existing volume before activation.

## Monitoring

Check installed version, latest release, last successful model and inference log. A model score is an estimate, not a verified user outcome. Monitor quality on held-out data and avoid overlapping training exports.

Sources: [training](https://github.com/arashrasoulzadeh/eye/blob/main/train/README.md), [registry](https://github.com/arashrasoulzadeh/eye/blob/main/models/README.md).
