"use client";
import { useEffect, useState } from "react";

export default function Events() {
  const [events, setEvents] = useState<any[]>([]);
  useEffect(() => {
    fetch("http://localhost:8080/events").then(r=>r.json()).then(setEvents);
  }, []);
  return (
    <div className="p-8">
      <h1 className="text-2xl mb-4">Events</h1>
      <ul>{events.map((e:any)=><li key={e.EventID}>{e.EventName} - {e.SessionID}</li>)}</ul>
    </div>
  );
}