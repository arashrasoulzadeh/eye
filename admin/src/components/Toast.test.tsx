import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, act } from "@testing-library/react";
import { ToastProvider, useToast } from "./Toast";

function Trigger() {
  const { push } = useToast();
  return (
    <button onClick={() => push("success", "Tenant created")}>fire</button>
  );
}

describe("Toast", () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => vi.useRealTimers());

  it("useToast throws outside a ToastProvider", () => {
    const Bare = () => {
      useToast();
      return null;
    };
    expect(() => render(<Bare />)).toThrow(/ToastProvider/);
  });

  it("shows a pushed toast and auto-dismisses it after a timeout", () => {
    render(
      <ToastProvider>
        <Trigger />
      </ToastProvider>
    );
    fireEvent.click(screen.getByText("fire"));
    expect(screen.getByText("Tenant created")).toBeInTheDocument();

    act(() => {
      vi.advanceTimersByTime(4000);
    });
    expect(screen.queryByText("Tenant created")).not.toBeInTheDocument();
  });
});
