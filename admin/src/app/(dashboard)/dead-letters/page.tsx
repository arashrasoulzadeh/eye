"use client";
import { useCallback } from "react";
import { api } from "@/lib/api";
import { OperationsResourcePage } from "@/components/OperationsResourcePage";
export default function DeadLettersPage() { const loader = useCallback(() => api.operations.deadLetters(), []); return <OperationsResourcePage title="Webhook Dead Letters" description="Inspect webhook deliveries that exhausted their retries." loader={loader}/>; }
