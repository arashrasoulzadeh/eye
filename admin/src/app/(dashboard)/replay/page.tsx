"use client";
import { useCallback } from "react";
import { OperationsResourcePage } from "@/components/OperationsResourcePage";
export default function ReplayPage() { const loader = useCallback(() => Promise.resolve([]), []); return <OperationsResourcePage title="Event Replay" description="Select a tenant and use the backend replay endpoint for a historical range." tenantScoped loader={loader}/>; }
