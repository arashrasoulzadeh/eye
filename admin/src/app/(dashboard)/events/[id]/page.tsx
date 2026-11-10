import { Suspense } from "react";
import EventDetailClient from "./EventDetailClient";
export default function EventPage(){return <Suspense fallback={<p className="text-sm text-fg-muted">Loading event…</p>}><EventDetailClient/></Suspense>}
