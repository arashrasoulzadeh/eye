import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Button } from "./Button";

describe("Button", () => {
  it("fires onClick when enabled", async () => {
    const onClick = vi.fn();
    render(<Button onClick={onClick}>Send</Button>);
    await userEvent.click(screen.getByRole("button", { name: "Send" }));
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it("is disabled and shows a spinner while loading, and does not fire onClick", async () => {
    const onClick = vi.fn();
    render(
      <Button onClick={onClick} loading>
        Send
      </Button>
    );
    const btn = screen.getByRole("button", { name: /Send/ });
    expect(btn).toBeDisabled();
    await userEvent.click(btn);
    expect(onClick).not.toHaveBeenCalled();
  });

  it("respects an explicit disabled prop", () => {
    render(<Button disabled>Send</Button>);
    expect(screen.getByRole("button", { name: "Send" })).toBeDisabled();
  });
});
