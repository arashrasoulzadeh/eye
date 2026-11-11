"use client";
import { useCallback } from "react";
import { api } from "@/lib/api";
import { OperationsResourcePage } from "@/components/OperationsResourcePage";
export default function AuditLogPage() { const loader = useCallback(() => api.operations.auditLog(), []); return <OperationsResourcePage title="Audit Log" description="Review administrative actions newest first." loader={loader}/>; }
