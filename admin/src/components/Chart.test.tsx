import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { LineTrend, BarBreakdown } from "./Chart";

describe("LineTrend", () => {
  it("shows an empty state with no data", () => {
    render(<LineTrend data={[]} />);
    expect(screen.getByText("No data yet.")).toBeInTheDocument();
  });

  it("renders the chart container when given data (jsdom has no layout, so recharts skips the inner svg)", () => {
    const { container } = render(<LineTrend data={[{ label: "mon", value: 3 }, { label: "tue", value: 5 }]} />);
    expect(container.querySelector(".recharts-responsive-container")).toBeInTheDocument();
  });
});

describe("BarBreakdown", () => {
  it("shows an empty state with no data", () => {
    render(<BarBreakdown data={[]} />);
    expect(screen.getByText("No data yet.")).toBeInTheDocument();
  });

  it("renders the chart container when given data (jsdom has no layout, so recharts skips the inner svg)", () => {
    const { container } = render(<BarBreakdown data={[{ label: "tenant-a", value: 10 }]} />);
    expect(container.querySelector(".recharts-responsive-container")).toBeInTheDocument();
  });
});
