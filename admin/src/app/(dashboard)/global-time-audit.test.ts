import { describe, expect, it } from "vitest";
import { readFileSync } from "node:fs";
import { join } from "node:path";

const root = join(process.cwd(), "src", "app", "(dashboard)");
const pages: Record<string, string[]> = {
  "dashboard/page.tsx": ["useTimeRange", "from_ts", "to_ts"],
  "analytics/page.tsx": ["useTimeRange", "from_ts", "to_ts"],
  "analytics/funnel/page.tsx": ["useTimeRange", "globalFrom", "globalTo"],
  "analytics/retention/page.tsx": ["useTimeRange", "from_ts", "to_ts"],
  "graphs/page.tsx": ["useTimeRange", "from_ts", "to_ts"],
  "events/page.tsx": ["useTimeRange", "from_ts", "to_ts"],
  "sessions/page.tsx": ["useTimeRange", "from_ts", "to_ts"],
  "users/page.tsx": ["useTimeRange", "from_ts", "to_ts"],
  "replay/page.tsx": ["useTimeRange", "globalFrom", "globalTo"],
  "stream/page.tsx": ["useTimeRange", "from_ts", "to_ts"],
};

describe("global time contract", () => {
  it("keeps every time-sensitive page connected to the shared range", () => {
    for (const [relativePath, required] of Object.entries(pages)) {
      const source = readFileSync(join(root, relativePath), "utf8");
      for (const token of required) expect(source, `${relativePath} must contain ${token}`).toContain(token);
      expect(source, `${relativePath} must not add local date controls`).not.toMatch(/datetime-local|Last days|Range mode/);
    }
  });
});
