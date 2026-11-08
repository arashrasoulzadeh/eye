"use client";
import { useState } from "react";
import { useRouter } from "next/navigation";

export default function Login() {
  const [key, setKey] = useState("");
  const router = useRouter();
  async function submit(e: React.FormEvent) {
    e.preventDefault();
    const res = await fetch("/api/login", { method: "POST", body: JSON.stringify({ key }), headers: { "content-type": "application/json" } });
    if (res.ok) router.push("/dashboard");
  }
  return (
    <form onSubmit={submit} className="p-8 max-w-sm mx-auto">
      <h1 className="text-xl mb-4">Admin Login</h1>
      <input className="border p-2 w-full mb-2" value={key} onChange={e=>setKey(e.target.value)} placeholder="API key" />
      <button className="bg-blue-600 text-white px-4 py-2">Login</button>
    </form>
  );
}