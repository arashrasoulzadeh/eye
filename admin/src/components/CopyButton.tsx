"use client";
import { useState } from "react";

export function CopyButton({ value, label }: { value: string; label?: string }) {
  const [copied, setCopied] = useState(false);
  return (
    <button
      onClick={async () => {
        try {
          await navigator.clipboard.writeText(value);
        } catch {
          // clipboard unavailable, ignore
        }
        setCopied(true);
        setTimeout(() => setCopied(false), 1200);
      }}
      className="font-mono text-xs text-fg-muted hover:text-accent"
      title="Copy to clipboard"
    >
      {copied ? "copied!" : label || value}
    </button>
  );
}
