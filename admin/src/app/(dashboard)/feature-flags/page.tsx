"use client";
import { useCallback } from "react";
import { api } from "@/lib/api";
import { OperationsResourcePage } from "@/components/OperationsResourcePage";
export default function FeatureFlagsPage() { const loader = useCallback((tenant: string) => api.operations.flags(tenant), []); return <OperationsResourcePage title="Feature Flags" description="Manage tenant-scoped feature switches." tenantScoped loader={loader}/>; }
