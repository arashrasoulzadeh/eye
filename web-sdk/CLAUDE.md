TS client SDK (`src/index.ts`): `EyeClient` — session persistence (localStorage), bounded ring buffer, batched flush. Tests in `../tests/web` (not colocated), run via `npm test` (vitest). See root [../CLAUDE.md](../CLAUDE.md) and [../phases.md](../phases.md) for status (Phase 5).

`src/cdn.ts`: CDN `<script>` build on top of `EyeClient` — exposes `window.Eye`, auto-tracks ~19 events by default (page view, clicks, forms, scroll depth, errors, etc. — see `AutoTrackOptions`), toggleable per-event in `init()`. Build with `npm run build:cdn` → `dist/eye.min.js` (gitignored, rebuild before shipping). Embed:

```html
<script src="https://cdn.example.com/eye.min.js"></script>
<script>
  Eye.init({ apiKey: "<tenant api key>", tenant: "<tenant id>", endpoint: "https://api.example.com" });
</script>
```

`Eye.init` creates a session via `POST /sessions` using the tenant's own API key (not the admin key — see `handler.TenantOrAdminAuth`, since the admin key must never ship to a browser). `Eye.track(name, props)` / `Eye.identify(userId, traits)` / `Eye.flush()` are also exposed.
