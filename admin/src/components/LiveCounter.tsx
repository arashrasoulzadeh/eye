"use client";
import { useEffect, useRef, useState } from "react";
import { Card } from "@/components/Card";

type ConnState = "connecting" | "live" | "error";

/** A live-updating per-tenant event count, fed by the backend's LiveCounters
 * gRPC stream via the /api/proxy-stream/live-counters SSE bridge (browsers
 * can't speak gRPC directly). */
export function LiveCounter({ tenantId }: { tenantId: string }) {
  const [count, setCount] = useState<number | null>(null);
  const [state, setState] = useState<ConnState>("connecting");
  const [pulse, setPulse] = useState(false);

  useEffect(() => {
    const es = new EventSource(`/api/proxy-stream/live-counters?tenant_id=${encodeURIComponent(tenantId)}`);
    es.onmessage = (ev) => {
      setState("live");
      try {
        const data = JSON.parse(ev.data) as { count: number };
        setCount((prev) => {
          if (prev !== null && data.count > prev) {
            setPulse(true);
            setTimeout(() => setPulse(false), 400);
          }
          return data.count;
        });
      } catch {
        // malformed SSE payload — ignore, next message may be fine
      }
    };
    es.onerror = () => setState("error");
    return () => es.close();
  }, [tenantId]);

  return (
    <Card className="px-5 py-4">
      <div className="flex items-center justify-between">
        <p className="text-xs font-medium uppercase tracking-wide text-fg-muted">Live event count</p>
        <span
          className={`flex items-center gap-1.5 text-xs ${
            state === "live" ? "text-green-600" : state === "error" ? "text-red-600" : "text-fg-muted"
          }`}
        >
          <span
            className={`h-1.5 w-1.5 rounded-full ${
              state === "live" ? "bg-green-500" : state === "error" ? "bg-red-500" : "bg-fg-muted"
            }`}
          />
          {state === "live" ? "live" : state === "error" ? "disconnected" : "connecting…"}
        </span>
      </div>
      <p className={`mt-2 text-3xl font-semibold text-fg transition-transform ${pulse ? "scale-110" : ""}`}>
        {count === null ? "—" : count.toLocaleString()}
      </p>
      <p className="mt-1 text-xs text-fg-muted">Updates in real time as events are ingested for this tenant.</p>
    </Card>
  );
}
