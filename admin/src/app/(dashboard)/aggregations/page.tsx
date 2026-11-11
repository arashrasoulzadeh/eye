"use client";
import { useCallback } from "react";
import { api } from "@/lib/api";
import { OperationsResourcePage } from "@/components/OperationsResourcePage";
export default function AggregationsPage() { const loader = useCallback((tenant: string) => api.operations.aggregations(tenant), []); return <OperationsResourcePage title="Aggregations" description="Inspect tenant materialized aggregation rules and counts." tenantScoped loader={loader}/>; }
