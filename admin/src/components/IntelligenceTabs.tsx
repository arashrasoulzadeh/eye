"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

const tabs = [["Overview", "/intelligence"], ["Models", "/intelligence/models"], ["Training", "/intelligence/training"], ["Labels", "/intelligence/labels"]] as const;

export function IntelligenceTabs() {
  const pathname = usePathname();
  if (!pathname.startsWith("/intelligence")) return null;
  return <nav aria-label="Intelligence sections" className="mb-5 flex flex-wrap gap-1 border-b border-border">{tabs.map(([label, href]) => { const active = pathname === href; return <Link key={href} href={href} aria-current={active ? "page" : undefined} className={`border-b-2 px-3 py-2 text-sm ${active ? "border-accent font-medium text-accent" : "border-transparent text-fg-muted hover:border-border hover:text-fg"}`}>{label}</Link>; })}</nav>;
}
