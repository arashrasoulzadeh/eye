import { Suspense } from "react";
import { connection } from "next/server";
import EventDetailClient from "./EventDetailClient";
export default async function EventPage(){await connection(); return <Suspense fallback={<p className="text-sm text-fg-muted">Loading event…</p>}><EventDetailClient/></Suspense>}
