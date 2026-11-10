"use client";
import { CopyButton } from "@/components/CopyButton";

/**
 * A short, navigable reference to a tenant or session: the truncated id
 * links to that entity's detail page, with a separate copy affordance for
 * the full id (so clicking the link always navigates, never just copies).
 */
export function EntityLink({
  kind,
  id,
  truncate = 10,
  label,
}: {
  kind: "tenant" | "session" | "event";
  id: string;
  truncate?: number;
  label?: string;
}) {
  const href = kind === "tenant" ? `/tenants/${id}` : kind === "session" ? `/sessions/${id}` : `/events/${id}`;
  const short = id.length > truncate ? id.slice(0, truncate) + "…" : id;
  return (
    <span className="inline-flex items-center gap-1.5 font-mono text-xs">
      <a href={href} className="text-fg-muted hover:text-accent hover:underline">
        <span title={id}>{label || short}</span>
      </a>
      <CopyButton value={id} label="⧉" />
    </span>
  );
}
