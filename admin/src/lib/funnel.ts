export function formatEventLabel(name: unknown, titles: Record<string, string> = {}) {
  const value = typeof name === "string" && name.trim() ? name : "Unknown event";
  return titles[value] || String(value).replace(/[_.-]+/g, " ").replace(/\b\w/g, (letter) => letter.toUpperCase());
}
