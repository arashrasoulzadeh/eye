"use client";

import { useEffect, useState } from "react";
import { Badge } from "@/components/Badge";
import { translate, useLanguage } from "@/components/LanguageContext";

type Kind = "user" | "session" | "event";
type Ref = { tenant_id: string; id: string };

export function tagRef(tenant: string, id: string): Ref {
  return { tenant_id: tenant, id };
}

export function useIntelligenceTags(kind: Kind, refs: Ref[]) {
  const body = JSON.stringify({ kind, refs });
  const enabled = refs.length > 0;
  const [result, setResult] = useState<{ body: string; tags: (string[] | null)[] }>();
  useEffect(() => {
    if (!enabled) return;
    const controller = new AbortController();
    fetch("/api/proxy/intelligence/tags", {
      method: "POST",
      body,
      signal: controller.signal,
    })
      .then(async (response) => {
        if (!response.ok) return;
        const tags: unknown = await response.json();
        if (!controller.signal.aborted && Array.isArray(tags)) setResult({ body, tags });
      })
      .catch(() => undefined);
    return () => controller.abort();
  }, [body, enabled]);
  return result?.body === body ? result.tags : undefined;
}

export function IntelligenceTags({ tags, compact = false }: { tags?: string[] | null; compact?: boolean }) {
  const { language } = useLanguage();
  if (!tags?.length) return null;
  const visible = compact ? tags.slice(0, 2) : tags;
  return (
    <div className="mt-1 flex flex-wrap items-center gap-1" aria-label={translate("Intelligence tags", language)}>
      {!compact && <span className="me-1 text-xs text-fg-muted">{translate("Intelligence tags", language)}</span>}
      {visible.map((tag) => <Badge key={tag} tone="accent">{translate(tag.replaceAll("_", " "), language)}</Badge>)}
      {visible.length < tags.length && <Badge>+{tags.length - visible.length}</Badge>}
    </div>
  );
}

export function EntityIntelligenceTags({ kind, refs }: { kind: Kind; refs: Ref[] }) {
  const tags = useIntelligenceTags(kind, refs);
  return <IntelligenceTags tags={[...new Set(tags?.flatMap((row) => row ?? []) ?? [])].sort()} />;
}
