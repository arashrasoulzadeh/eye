"use client";
import { useEffect, useState } from "react";

export default function Sessions() {
  const [sessions, setSessions] = useState<any[]>([]);
  useEffect(() => {
    fetch("http://localhost:8080/sessions").then(r=>r.json()).then(setSessions);
  }, []);
  return (
    <div className="p-8">
      <h1 className="text-2xl mb-4">Sessions</h1>
      <ul>{sessions.map((s:any)=><li key={s.ID}>{s.ID} - {s.TenantID} - {s.EventCount} events</li>)}</ul>
    </div>
  );
}