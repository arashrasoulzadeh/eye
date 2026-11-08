import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Combobox } from "./Combobox";

const OPTIONS = [
  { value: "click", label: "click" },
  { value: "page_view", label: "page_view" },
  { value: "purchase", label: "purchase" },
];

function setup(value = "") {
  const onChange = vi.fn();
  render(<Combobox value={value} onChange={onChange} options={OPTIONS} placeholder="event name" />);
  return { onChange, input: screen.getByPlaceholderText("event name") as HTMLInputElement };
}

describe("Combobox", () => {
  it("shows all preloaded top options on focus when empty", async () => {
    const { input } = setup();
    await userEvent.click(input);
    for (const o of OPTIONS) {
      expect(screen.getByText(o.value)).toBeInTheDocument();
    }
  });

  it("filters the option list as the user types", async () => {
    const { input } = setup("pur");
    await userEvent.click(input);
    expect(screen.getByText("purchase")).toBeInTheDocument();
    expect(screen.queryByText("click")).not.toBeInTheDocument();
    expect(screen.queryByText("page_view")).not.toBeInTheDocument();
  });

  it("calls onChange with the typed value on every keystroke", async () => {
    const { input, onChange } = setup();
    await userEvent.type(input, "a");
    expect(onChange).toHaveBeenCalledWith("a");
  });

  it("selects an option on click and closes the dropdown", async () => {
    const onChange = vi.fn();
    const { rerender } = render(
      <Combobox value="" onChange={onChange} options={OPTIONS} placeholder="event name" />
    );
    const input = screen.getByPlaceholderText("event name");
    await userEvent.click(input);
    await userEvent.click(screen.getByText("purchase"));
    expect(onChange).toHaveBeenCalledWith("purchase");

    rerender(<Combobox value="purchase" onChange={onChange} options={OPTIONS} placeholder="event name" />);
    expect(screen.queryByText("click")).not.toBeInTheDocument();
  });

  it("navigates and selects with arrow keys + Enter", async () => {
    const onChange = vi.fn();
    render(<Combobox value="" onChange={onChange} options={OPTIONS} placeholder="event name" />);
    const input = screen.getByPlaceholderText("event name");
    fireEvent.focus(input);
    fireEvent.keyDown(input, { key: "ArrowDown" }); // highlight moves from "click" to "page_view"
    fireEvent.keyDown(input, { key: "Enter" });
    expect(onChange).toHaveBeenCalledWith("page_view");
  });

  it("closes the dropdown on Escape", async () => {
    const { input } = setup();
    await userEvent.click(input);
    expect(screen.getByText("click")).toBeInTheDocument();
    fireEvent.keyDown(input, { key: "Escape" });
    expect(screen.queryByText("click")).not.toBeInTheDocument();
  });

  it("allows freeform typed values that don't match any option, without blocking input", async () => {
    const { input, onChange } = setup();
    await userEvent.type(input, "zzz");
    expect(onChange).toHaveBeenCalledTimes(3);
    expect(input).not.toBeDisabled();
  });
});
