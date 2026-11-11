"use client";
import { useCallback } from "react";
import { api } from "@/lib/api";
import { OperationsResourcePage } from "@/components/OperationsResourcePage";
export default function AnomaliesPage() { const loader = useCallback(() => api.operations.anomalies(), []); return <OperationsResourcePage title="Anomalies" description="Review currently detected event anomalies." loader={loader}/>; }
