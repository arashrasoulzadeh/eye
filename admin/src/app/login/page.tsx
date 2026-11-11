"use client";
import { useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/Button";
import { Input, Label } from "@/components/Input";
import { Card } from "@/components/Card";

export default function Login() {
  const [key, setKey] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const router = useRouter();

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      const res = await fetch("/api/login", {
        method: "POST",
        body: JSON.stringify({ key }),
        headers: { "content-type": "application/json" },
      });
      if (res.ok) {
        router.push("/dashboard");
      } else {
        setError("Invalid API key.");
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-bg px-4">
      <Card className="w-full max-w-sm p-7">
        <div className="mb-6 flex items-center gap-2">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-accent text-sm font-bold text-accent-fg">
            e
          </div>
          <span className="text-base font-semibold text-fg">eye admin</span>
        </div>
        <form onSubmit={submit} className="space-y-4">
          <div>
            <Label>Admin API key</Label>
            <Input
              type="password"
              value={key}
              onChange={(e) => setKey(e.target.value)}
              placeholder="••••••••••••"
              autoFocus
            />
          </div>
          {error && <p className="text-sm text-danger">{error}</p>}
          <Button type="submit" loading={loading} className="w-full">
            Sign in
          </Button>
        </form>
        {process.env.NEXT_PUBLIC_OIDC_ENABLED === "true" && (
          <a href="/api/oidc/start" className="mt-3 block rounded border border-border px-3 py-2 text-center text-sm text-fg">Sign in with SSO</a>
        )}
      </Card>
    </div>
  );
}
