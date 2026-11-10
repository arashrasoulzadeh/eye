"use client";
import { useState } from "react";

export function JsonView({ value }: { value: string }) {
  const [open, setOpen] = useState(false);
  let pretty = value;
  try {
    pretty = JSON.stringify(JSON.parse(value), null, 2);
  } catch {
    // not JSON, show raw
  }
  const isEmpty = !value || value === "null" || value === "{}";
  if (isEmpty) return <span className="font-mono text-xs text-fg-muted">{value || "{}"}</span>;

  return (
    <div>
      <button
        onClick={() => setOpen((o) => !o)}
        className="font-mono text-xs text-accent hover:underline"
      >
        {open ? "hide" : "view"} json
      </button>
      {open && (
        <pre className="mt-1 max-w-sm overflow-auto rounded-md bg-surface-2 p-2 text-xs text-fg">
          {pretty}
        </pre>
      )}
    </div>
  );
}
