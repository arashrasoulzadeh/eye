import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import SettingsPage from "./page";
import { ToastProvider } from "@/components/Toast";

vi.mock("@/lib/api", async () => {
  const actual = await vi.importActual<typeof import("@/lib/api")>("@/lib/api");
  return {
    ...actual,
    api: {
      settings: { get: vi.fn(), update: vi.fn() },
    },
  };
});

import { api } from "@/lib/api";

function renderPage() {
  return render(
    <ToastProvider>
      <SettingsPage />
    </ToastProvider>
  );
}

describe("SettingsPage", () => {
  beforeEach(() => {
    vi.mocked(api.settings.get).mockResolvedValue({ ID: 1, RetentionHours: 72, RateLimitPerSecond: 50, UpdatedAt: 0 });
  });

  it("renders the current settings", async () => {
    renderPage();
    await waitFor(() => expect(screen.getByDisplayValue("72")).toBeInTheDocument());
    expect(screen.getByDisplayValue("50")).toBeInTheDocument();
  });

  it("submits an update and shows a success toast", async () => {
    vi.mocked(api.settings.update).mockResolvedValue({ ID: 1, RetentionHours: 10, RateLimitPerSecond: 50, UpdatedAt: 1234 });
    renderPage();
    await waitFor(() => expect(screen.getByDisplayValue("72")).toBeInTheDocument());

    fireEvent.change(screen.getByDisplayValue("72"), { target: { value: "10" } });
    fireEvent.click(screen.getByText("Save"));

    await waitFor(() => expect(api.settings.update).toHaveBeenCalledWith({ retention_hours: 10, rate_limit_per_second: 50 }));
    await waitFor(() => expect(screen.getByText(/Settings saved/)).toBeInTheDocument());
  });

  it("shows an error toast when loading fails", async () => {
    vi.mocked(api.settings.get).mockRejectedValue(new Error("boom"));
    renderPage();
    await waitFor(() => expect(screen.getByText("Failed to load settings")).toBeInTheDocument());
  });
});
