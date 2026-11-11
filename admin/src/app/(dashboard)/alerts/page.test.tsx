import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import AlertsPage from "./page";
import { ToastProvider } from "@/components/Toast";

vi.mock("@/lib/api", async () => {
  const actual = await vi.importActual<typeof import("@/lib/api")>("@/lib/api");
  return {
    ...actual,
    api: {
      tenants: { list: vi.fn() },
      events: { list: vi.fn() },
      alerts: { list: vi.fn(), listPage: vi.fn(), create: vi.fn(), update: vi.fn(), remove: vi.fn(), setEnabled: vi.fn() },
    },
  };
});

import { api } from "@/lib/api";

function renderPage() {
  return render(
    <ToastProvider>
      <AlertsPage />
    </ToastProvider>
  );
}

const baseRule = {
  ID: "r1",
  TenantID: "t1",
  EventName: "error",
  DimsKey: "",
  Metric: "count",
  Threshold: 10,
  WebhookURL: "https://example.com/hook",
  Enabled: true,
  LastFiredAt: 0,
  LastFiredCount: 0,
  CreatedAt: 0,
  UpdatedAt: 0,
};

describe("AlertsPage", () => {
  beforeEach(() => {
    vi.mocked(api.tenants.list).mockResolvedValue([{ ID: "t1", Name: "acme", APIKey: "", CreatedAt: "" }]);
    vi.mocked(api.events.list).mockResolvedValue([]);
    vi.mocked(api.alerts.list).mockResolvedValue([baseRule]);
    vi.mocked(api.alerts.listPage).mockResolvedValue({ items: [baseRule], total: 1, limit: 10, offset: 0 });
  });

  it("lists existing alert rules with enabled status and never-fired state", async () => {
    renderPage();
    await waitFor(() => expect(screen.getByText("error")).toBeInTheDocument());
    expect(screen.getByText("enabled")).toBeInTheDocument();
    expect(screen.getByText("never")).toBeInTheDocument();
  });

  it("toggles a rule's enabled state", async () => {
    vi.mocked(api.alerts.setEnabled).mockResolvedValue({ ...baseRule, Enabled: false });
    renderPage();
    await waitFor(() => expect(screen.getByText("error")).toBeInTheDocument());
    fireEvent.click(screen.getByText("Disable"));
    await waitFor(() => expect(api.alerts.setEnabled).toHaveBeenCalledWith("r1", false));
  });

  it("deletes a rule after confirmation", async () => {
    vi.mocked(api.alerts.remove).mockResolvedValue(undefined);
    renderPage();
    await waitFor(() => expect(screen.getByText("error")).toBeInTheDocument());
    fireEvent.click(screen.getByText("Delete"));
    fireEvent.click(screen.getByText("Delete rule"));
    await waitFor(() => expect(api.alerts.remove).toHaveBeenCalledWith("r1"));
  });

  it("creates a new rule from the form", async () => {
    vi.mocked(api.alerts.create).mockResolvedValue({ ...baseRule, ID: "r2" });
    renderPage();
    await waitFor(() => expect(screen.getByText("error")).toBeInTheDocument());

    fireEvent.change(screen.getByPlaceholderText("error"), { target: { value: "purchase" } });
    fireEvent.change(screen.getByPlaceholderText("https://example.com/hook"), { target: { value: "https://example.com/wh" } });
    fireEvent.click(screen.getByText("Create rule"));

    await waitFor(() =>
      expect(api.alerts.create).toHaveBeenCalledWith({ tenant_id: "t1", event_name: "purchase", threshold: 10, webhook_url: "https://example.com/wh" })
    );
  });

  it("edits an existing rule", async () => {
    vi.mocked(api.alerts.update).mockResolvedValue({ ...baseRule, Threshold: 25 });
    renderPage();
    await waitFor(() => expect(screen.getByText("error")).toBeInTheDocument());
    fireEvent.click(screen.getByText("Edit"));
    fireEvent.change(screen.getByDisplayValue("10"), { target: { value: "25" } });
    fireEvent.click(screen.getByText("Save rule"));
    await waitFor(() => expect(api.alerts.update).toHaveBeenCalledWith("r1", { event_name: "error", dims_key: "", metric: "count", threshold: 25, webhook_url: "https://example.com/hook" }));
  });
});
