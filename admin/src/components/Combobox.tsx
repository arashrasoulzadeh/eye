"use client";
import { useEffect, useRef, useState } from "react";

export type ComboOption = { value: string; label?: string };

/**
 * Searchable combobox with optional API-backed searching.
 */
export function Combobox({
  value,
  onChange,
  options,
  placeholder,
  maxVisible = 5,
  onSearch,
}: {
  value: string;
  onChange: (v: string) => void;
  options: ComboOption[];
  placeholder?: string;
  maxVisible?: number;
  onSearch?: (query: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const [highlight, setHighlight] = useState(0);
  const [search, setSearch] = useState("");
  const rootRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function onClick(e: MouseEvent) {
      if (rootRef.current && !rootRef.current.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener("mousedown", onClick);
    return () => document.removeEventListener("mousedown", onClick);
  }, []);

  const filtered = (
    search.trim()
      ? options.filter(
          (o) =>
            o.value.toLowerCase().includes(search.toLowerCase()) ||
            (o.label || "").toLowerCase().includes(search.toLowerCase())
        )
      : options
  ).slice(0, maxVisible);

  function select(v: string) {
    onChange(v);
    setSearch("");
    setOpen(false);
  }

  const selectedLabel = options.find((o) => o.value === value)?.label || value;

  return (
    <div ref={rootRef} className="relative">
      <input
        value={open ? search : selectedLabel}
        onChange={(e) => {
          setSearch(e.target.value);
          onChange(e.target.value);
          onSearch?.(e.target.value);
          setOpen(true);
          setHighlight(0);
        }}
        onFocus={() => { setSearch(options.some((o) => o.value === value) ? "" : value); setOpen(true); onSearch?.(value); }}
        onClick={() => { setSearch(options.some((o) => o.value === value) ? "" : search); setOpen(true); onSearch?.(""); }}
        onKeyDown={(e) => {
          if (!open) return;
          if (e.key === "ArrowDown") {
            e.preventDefault();
            setHighlight((h) => Math.min(h + 1, filtered.length - 1));
          } else if (e.key === "ArrowUp") {
            e.preventDefault();
            setHighlight((h) => Math.max(h - 1, 0));
          } else if (e.key === "Enter") {
            if (filtered[highlight]) {
              e.preventDefault();
              select(filtered[highlight].value);
            }
          } else if (e.key === "Escape") {
            setOpen(false);
          }
        }}
        placeholder={placeholder}
        className="w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-fg placeholder:text-fg-muted outline-none focus:border-accent focus:ring-2 focus:ring-accent/20"
      />
      {open && filtered.length > 0 && (
        <ul className="absolute z-20 mt-1 w-full max-h-56 overflow-auto rounded-lg border border-border bg-surface shadow-lg">
          {filtered.map((o, i) => (
            <li key={o.value}>
              <button
                type="button"
                onMouseDown={(e) => e.preventDefault()}
                onClick={() => select(o.value)}
                className={`flex w-full items-center justify-between px-3 py-1.5 text-left text-sm ${
                  i === highlight ? "bg-accent/10 text-accent" : "text-fg hover:bg-surface-2"
                }`}
              >
                <span className="truncate">{o.value}</span>
                {o.label && o.label !== o.value && (
                  <span className="ml-2 shrink-0 text-xs text-fg-muted">{o.label}</span>
                )}
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
