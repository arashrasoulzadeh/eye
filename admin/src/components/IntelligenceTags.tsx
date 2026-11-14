"use client";

import { Badge } from "@/components/Badge";
import { translate, useLanguage } from "@/components/LanguageContext";

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
