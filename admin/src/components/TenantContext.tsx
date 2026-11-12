"use client";

import { createContext, useContext, useEffect, useMemo, useState } from "react";
import { api, Tenant } from "@/lib/api";

type TenantContextValue = {
  tenants: Tenant[];
  tenantId: string;
  setTenantId: (value: string) => void;
  loading: boolean;
};

const TenantContext = createContext<TenantContextValue | null>(null);
const STORAGE_KEY = "eye.admin.tenant";

export function TenantProvider({ children }: { children: React.ReactNode }) {
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [tenantId, setTenantIdState] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const stored = window.localStorage.getItem(STORAGE_KEY);
    if (stored) setTenantIdState(stored);
    api.tenants.list().then((items) => {
      const available = Array.isArray(items) ? items : [];
      setTenants(available);
      setTenantIdState((current) => current && available.some((item) => item.ID === current) ? current : "");
    }).finally(() => setLoading(false));
  }, []);

  function setTenantId(value: string) {
    setTenantIdState(value);
    if (value) window.localStorage.setItem(STORAGE_KEY, value);
    else window.localStorage.removeItem(STORAGE_KEY);
    window.dispatchEvent(new CustomEvent("eye-tenant-change", { detail: value }));
    window.location.reload();
  }

  const value = useMemo(() => ({ tenants, tenantId, setTenantId, loading }), [tenants, tenantId, loading]);
  return <TenantContext.Provider value={value}>{children}</TenantContext.Provider>;
}

export function useTenant() {
  const value = useContext(TenantContext);
  return value || { tenants: [], tenantId: "", setTenantId: () => undefined, loading: false };
}
