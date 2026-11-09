"use client";
import { useEffect, useRef, useState } from "react";
import { Card } from "@/components/Card";
import { Session } from "@/lib/api";

type ConnState = "connecting" | "live" | "error";

type LiveSessionData = {
  event_count: number;
  last_seen: string;
};

/** A live-updating per-session event count and last seen, fed by the backend's
 * LiveSessionHub via the /api/proxy-stream/live-session SSE bridge. */
export function LiveSession({ session, enabled = true, className }: { session: Session; enabled?: boolean; className?: string }) {
  const [count, setCount] = useState<number>(session.EventCount);
  const [lastSeen, setLastSeen] = useState<string>(session.LastSeen);
  const [state, setState] = useState<ConnState>("connecting");
  const [pulse, setPulse] = useState(false);
  const countRef = useRef(count);
  const lastSeenRef = useRef(lastSeen);

  useEffect(() => {
    countRef.current = count;
  }, [count]);

  useEffect(() => {
    lastSeenRef.current = lastSeen;
  }, [lastSeen]);

  useEffect(() => {
    if (!enabled) {
      setState("error");
      return;
    }

    const es = new EventSource(`/api/proxy-stream/live-session?session_id=${encodeURIComponent(session.ID)}`);
    es.onmessage = (ev) => {
      setState("live");
      try {
        const data = JSON.parse(ev.data) as LiveSessionData;
        if (data.event_count > countRef.current) {
          setPulse(true);
          setTimeout(() => setPulse(false), 400);
        }
        setCount(data.event_count);
        if (data.last_seen) {
          setLastSeen(data.last_seen);
        }
      } catch {
        // malformed SSE payload — ignore, next message may be fine
      }
    };
    es.onerror = () => setState("error");
    return () => es.close();
  }, [session.ID, enabled]);

  return (
    <Card className={`px-5 py-4 ${className || ""}`}>
      <div className="flex items-center justify-between">
        <p className="text-xs font-medium uppercase tracking-wide text-fg-muted">Live session</p>
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
      <div className="mt-2 grid grid-cols-2 gap-4">
        <div>
          <p className="text-xs text-fg-muted">Events</p>
          <p className={`text-2xl font-semibold text-fg transition-transform ${pulse ? "scale-110" : ""}`}>
            {count.toLocaleString()}
          </p>
        </div>
        <div>
          <p className="text-xs text-fg-muted">Last seen</p>
          <p className="text-lg font-medium text-fg font-mono">
            {lastSeen ? new Date(lastSeen).toLocaleString() : "—"}
          </p>
        </div>
      </div>
      <p className="mt-1 text-xs text-fg-muted">Updates in real time as events are ingested for this session.</p>
    </Card>
  );
}