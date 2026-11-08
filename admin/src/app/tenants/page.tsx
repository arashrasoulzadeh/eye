"use client";
import { useEffect, useState } from "react";

export default function Tenants() {
  const [tenants, setTenants] = useState<any[]>([]);
  const [name, setName] = useState("");
  useEffect(() => {
    fetch("http://localhost:8080/tenants").then(r=>r.json()).then(setTenants);
  }, []);
  async function create() {
    await fetch("http://localhost:8080/tenants", { method: "POST", headers: { "content-type": "application/json" }, body: JSON.stringify({ Name: name, APIKey: "key-" + name }) });
    const r = await fetch("http://localhost:8080/tenants");
    setTenants(await r.json());
  }
  return (
    <div className="p-8">
      <h1 className="text-2xl mb-4">Tenants</h1>
      <input className="border p-2 mr-2" value={name} onChange={e=>setName(e.target.value)} placeholder="name" />
      <button className="bg-blue-600 text-white px-3 py-2" onClick={create}>Create</button>
      <ul className="mt-4">{tenants.map((t:any)=><li key={t.ID}>{t.Name} ({t.ID})</li>)}</ul>
    </div>
  );
}