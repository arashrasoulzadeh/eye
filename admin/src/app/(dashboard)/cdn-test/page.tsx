"use client";
import { useEffect, useState } from "react";
import { api, Tenant, ApiError } from "@/lib/api";
import { Card, CardHeader } from "@/components/Card";
import { Button } from "@/components/Button";
import { Input, Label, Select } from "@/components/Input";
import { CopyButton } from "@/components/CopyButton";
import { useToast } from "@/components/Toast";

export default function CdnTestPage() {
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [tenantId, setTenantId] = useState("");
  const [apiKey, setApiKey] = useState("");
  const [rotating, setRotating] = useState(false);
  const [endpoint, setEndpoint] = useState("http://localhost:8080");
  const [scriptUrl, setScriptUrl] = useState("/cdn-eye.min.js");
  const { push } = useToast();

  useEffect(() => {
    api.tenants.list().then((ts) => {
      setTenants(ts);
      if (ts[0]) setTenantId(ts[0].ID);
    });
  }, []);

  async function rotateAndFillKey() {
    if (!tenantId) return;
    setRotating(true);
    try {
      const res = await api.tenants.rotateKey(tenantId);
      setApiKey(res.api_key);
      push("success", "New key generated — any existing integration using the old key for this tenant just broke");
    } catch (e) {
      push("error", e instanceof ApiError ? e.message : "Failed to rotate key");
    } finally {
      setRotating(false);
    }
  }

  const landingUrl =
    tenantId && apiKey
      ? `/cdn-test-landing.html?${new URLSearchParams({
          tenant: tenantId,
          apiKey,
          endpoint,
          script: scriptUrl,
        }).toString()}`
      : "";

  function openTestTab() {
    if (!landingUrl) {
      push("error", "Pick a tenant and provide an API key first");
      return;
    }
    window.open(landingUrl, "_blank", "noopener,noreferrer");
  }

  return (
    <div>
      <h1 className="text-xl font-semibold text-fg">CDN Test</h1>
      <p className="mt-1 text-sm text-fg-muted">
        Open a plain, standalone HTML page in a new tab that loads the CDN build of the web SDK
        (<code className="text-xs">web-sdk/src/cdn.ts</code> → <code className="text-xs">window.Eye</code>) exactly the
        way a customer&apos;s site would, and fires its auto-tracked events for real against this backend.
      </p>

      <Card className="mt-6">
        <CardHeader title="Test configuration" />
        <div className="space-y-3 p-5">
          <div>
            <Label>Tenant</Label>
            <Select value={tenantId} onChange={(e) => { setTenantId(e.target.value); setApiKey(""); }}>
              {tenants.length === 0 && <option value="">Create a tenant first</option>}
              {tenants.map((t) => (
                <option key={t.ID} value={t.ID}>
                  {t.Name}
                </option>
              ))}
            </Select>
          </div>

          <div>
            <Label>Tenant API key (plaintext — only ever shown once, on creation/rotation)</Label>
            <div className="flex gap-2">
              <Input
                value={apiKey}
                onChange={(e) => setApiKey(e.target.value)}
                placeholder="paste a tenant API key, or rotate one below"
              />
              <Button variant="secondary" onClick={rotateAndFillKey} loading={rotating} disabled={!tenantId}>
                Rotate &amp; use
              </Button>
            </div>
            <p className="mt-1 text-xs text-fg-muted">
              Rotating invalidates this tenant&apos;s current key for any real integration — only do this for a
              throwaway/test tenant.
            </p>
          </div>

          <div>
            <Label>Backend endpoint</Label>
            <Input value={endpoint} onChange={(e) => setEndpoint(e.target.value)} placeholder="http://localhost:8080" />
          </div>

          <div>
            <Label>Script URL</Label>
            <Input value={scriptUrl} onChange={(e) => setScriptUrl(e.target.value)} placeholder="/cdn-eye.min.js" />
            <p className="mt-1 text-xs text-fg-muted">
              Defaults to the build copied into this admin app&apos;s <code>public/</code> (run{" "}
              <code>npm run sync:cdn</code> in <code>admin/</code> after rebuilding the SDK). Point this at a jsDelivr
              URL (e.g. <code>https://cdn.jsdelivr.net/gh/&lt;owner&gt;/&lt;repo&gt;@cdn/eye.min.js</code>) to test the
              real published CDN build instead.
            </p>
          </div>

          <div className="flex items-center gap-2 pt-2">
            <Button onClick={openTestTab} disabled={!tenantId || !apiKey}>
              Open test page in new tab
            </Button>
            {landingUrl && <CopyButton value={window.location.origin + landingUrl} label="copy link" />}
          </div>
        </div>
      </Card>
    </div>
  );
}
