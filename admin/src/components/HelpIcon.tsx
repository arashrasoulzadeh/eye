"use client";

import { ReactNode, useState } from "react";

export function HelpIcon({ title, children }: { title: string; children: ReactNode }) {
  const [open, setOpen] = useState(false);
  return <div className="relative"><button type="button" aria-label={`${title} help`} title={`${title} help`} onClick={() => setOpen((value) => !value)} className="inline-flex h-8 w-8 items-center justify-center rounded-full border border-border bg-surface text-sm font-semibold text-fg-muted transition hover:border-accent hover:text-accent">?</button>{open && <div role="dialog" aria-label={`${title} help`} className="absolute right-0 top-10 z-50 w-80 rounded-xl border border-border bg-surface p-4 text-sm shadow-xl"><div className="mb-2 flex items-center justify-between gap-3 font-medium text-fg"><span>{title} help</span><button type="button" aria-label="Close help" onClick={() => setOpen(false)} className="text-lg leading-none text-fg-muted hover:text-fg">×</button></div><div className="leading-6 text-fg-muted">{children}</div></div>}</div>;
}
