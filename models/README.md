# Offline model registry

Place validated artifacts here before publishing:

```text
models/global/buyer/manifest.json
models/global/buyer/model.json
models/tenants/<tenant-id>/buyer/manifest.json
models/tenants/<tenant-id>/buyer/model.json
```

The model workflow publishes this directory as a standalone GHCR image. The
server verifies each manifest checksum before using an artifact.
