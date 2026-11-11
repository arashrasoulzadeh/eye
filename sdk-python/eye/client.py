from dataclasses import dataclass, asdict
import json
import os
import sqlite3
import time
import uuid
from urllib import request


@dataclass(frozen=True)
class Diagnostics:
    tenant_id: str
    session_id: str | None
    queued_events: int
    dropped_events: int
    last_flush_at: int | None
    last_error: str | None


class EyeClient:
    """Small Python client with SQLite-backed, retry-safe event delivery."""

    def __init__(self, endpoint: str, api_key: str, tenant_id: str, queue_path: str = ".eye-queue.sqlite3", max_queue_size: int = 10000):
        self.endpoint = endpoint.rstrip("/")
        self.api_key = api_key
        self.tenant_id = tenant_id
        self.max_queue_size = max_queue_size
        self.db = sqlite3.connect(queue_path)
        self.db.execute("CREATE TABLE IF NOT EXISTS eye_meta (key TEXT PRIMARY KEY, value TEXT NOT NULL)")
        self.db.execute("CREATE TABLE IF NOT EXISTS eye_events (id INTEGER PRIMARY KEY AUTOINCREMENT, event TEXT NOT NULL)")
        self.db.commit()

    def _meta(self, key: str) -> str | None:
        row = self.db.execute("SELECT value FROM eye_meta WHERE key = ?", (key,)).fetchone()
        return row[0] if row else None

    def _set_meta(self, key: str, value: str) -> None:
        self.db.execute("INSERT INTO eye_meta(key, value) VALUES(?, ?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", (key, value))

    def ensure_session(self) -> str:
        current = self._meta("session_id")
        if current:
            return current
        body = json.dumps({"tenant_id": self.tenant_id, "device": {"runtime": "python"}}).encode()
        response = request.urlopen(request.Request(self.endpoint + "/sessions", data=body, headers={"Content-Type": "application/json", "x-api-key": self.api_key}, method="POST"))
        data = json.loads(response.read())
        session_id = data.get("session_id") or data.get("ID") or data.get("id")
        if not session_id:
            raise RuntimeError("session creation returned no id")
        self._set_meta("session_id", session_id)
        self.db.commit()
        return session_id

    def track(self, event_name: str, payload: dict | None = None, dims: dict | None = None) -> None:
        if not event_name.strip():
            raise ValueError("event name is required")
        session_id = self._meta("session_id")
        if not session_id:
            raise RuntimeError("call ensure_session first")
        event = {"event_id": uuid.uuid4().hex, "session_id": session_id, "event_name": event_name, "client_ts": int(time.time() * 1000), "payload": payload or {}, "dims": dims or {}}
        self.db.execute("INSERT INTO eye_events(event) VALUES(?)", (json.dumps(event, separators=(",", ":")),))
        count = self.db.execute("SELECT COUNT(*) FROM eye_events").fetchone()[0]
        dropped = max(0, count - self.max_queue_size)
        if dropped:
            self.db.execute("DELETE FROM eye_events WHERE id IN (SELECT id FROM eye_events ORDER BY id LIMIT ?)", (dropped,))
            self._set_meta("dropped_events", str(int(self._meta("dropped_events") or "0") + dropped))
        self.db.commit()

    def flush(self) -> int:
        rows = self.db.execute("SELECT id, event FROM eye_events ORDER BY id").fetchall()
        if not rows:
            return 0
        events = [json.loads(row[1]) for row in rows]
        try:
            body = json.dumps({"events": events}).encode()
            response = request.urlopen(request.Request(self.endpoint + "/ingest", data=body, headers={"Content-Type": "application/json", "x-api-key": self.api_key}, method="POST"))
            if response.status >= 300:
                raise RuntimeError(f"ingest failed: HTTP {response.status}")
        except Exception as error:
            self._set_meta("last_error", str(error)); self.db.commit(); raise
        self.db.execute("DELETE FROM eye_events WHERE id IN (SELECT id FROM eye_events ORDER BY id LIMIT ?)", (len(rows),))
        self._set_meta("last_flush_at", str(int(time.time() * 1000)))
        self._set_meta("last_error", "")
        self.db.commit()
        return len(rows)

    def diagnostics(self) -> Diagnostics:
        return Diagnostics(self.tenant_id, self._meta("session_id"), self.db.execute("SELECT COUNT(*) FROM eye_events").fetchone()[0], int(self._meta("dropped_events") or "0"), int(self._meta("last_flush_at")) if self._meta("last_flush_at") else None, self._meta("last_error") or None)

    def close(self) -> None:
        self.db.close()
