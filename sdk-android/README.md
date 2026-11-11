# Eye Android SDK

`EyeClient` is the Android/Kotlin port of the web and Go SDK. It creates a tenant session with `POST /sessions`, batches events to `POST /ingest`, keeps a bounded buffer, persists the session ID in `SharedPreferences`, and supports `identify` traits.

```kotlin
val eye = EyeClient(this, EyeOptions(
    endpoint = "https://eye-api.example.com",
    apiKey = "TENANT_CLIENT_KEY",
    tenantId = "TENANT_ID",
))
eye.ensureSession { result -> result.getOrThrow() }
eye.track("screen_view", JSONObject().put("screen", "home"))
```

The API key must be the plaintext tenant client key returned by tenant creation/rotation; never use `ADMIN_KEY`, `API_KEY_SALT`, or the hashed database value.
