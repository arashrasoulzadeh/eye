"use client";
import { useEffect, useState } from "react";

export default function Dashboard() {
  const [m, setM] = useState<any>(null);
  useEffect(() => {
    fetch("http://localhost:8080/metrics").then(r=>r.json()).then(setM);
  }, []);
  return (
    <div className="p-8">
      <h1 className="text-2xl mb-4">Dashboard</h1>
      {m ? (
        <ul>
          <li>Events: {m.events}</li>
          <li>Sessions: {m.sessions}</li>
          <li>Tenants: {m.tenants}</li>
        </ul>
      ) : "Loading..."}
    </div>
  );
}