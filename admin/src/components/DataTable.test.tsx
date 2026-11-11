import { describe, it, expect, vi } from "vitest";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { DataTable, Column } from "./DataTable";

type Row = { id: string; name: string; count: number };

const columns: Column<Row>[] = [
  { key: "name", header: "Name", render: (r) => r.name, sortValue: (r) => r.name },
  { key: "count", header: "Count", render: (r) => String(r.count), sortValue: (r) => r.count },
];

function rows(n: number): Row[] {
  return Array.from({ length: n }, (_, i) => ({ id: `r${i}`, name: `name-${i}`, count: n - i }));
}

describe("DataTable", () => {
  it("shows the empty message when there are no rows", () => {
    render(<DataTable columns={columns} rows={[]} rowKey={(r) => r.id} emptyMessage="Nothing here." />);
    expect(screen.getByText("Nothing here.")).toBeInTheDocument();
  });

  it("renders one row per item with the configured columns", () => {
    render(<DataTable columns={columns} rows={rows(3)} rowKey={(r) => r.id} />);
    expect(screen.getAllByRole("row")).toHaveLength(4); // header + 3 rows
    expect(screen.getByText("name-0")).toBeInTheDocument();
  });

  it("sorts ascending then descending when a sortable header is clicked", async () => {
    render(<DataTable columns={columns} rows={rows(3)} rowKey={(r) => r.id} pageSize={10} />);
    const header = screen.getByText("Name");

    await userEvent.click(header);
    let dataRows = screen.getAllByRole("row").slice(1);
    expect(within(dataRows[0]).getByText("name-0")).toBeInTheDocument();

    await userEvent.click(header);
    dataRows = screen.getAllByRole("row").slice(1);
    expect(within(dataRows[0]).getByText("name-2")).toBeInTheDocument();
  });

  it("paginates and navigates with Prev/Next", async () => {
    render(<DataTable columns={columns} rows={rows(25)} rowKey={(r) => r.id} pageSize={10} />);
    expect(screen.getByText(/Page 1 of 3/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Prev" })).toBeDisabled();

    await userEvent.click(screen.getByRole("button", { name: "Next" }));
    expect(screen.getByText(/Page 2 of 3/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Prev" })).not.toBeDisabled();

    await userEvent.click(screen.getByRole("button", { name: "Next" }));
    expect(screen.getByText(/Page 3 of 3/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Next" })).toBeDisabled();
  });

  it("notifies the API when server-side pagination changes page", async () => {
    const onQueryChange = vi.fn();
    render(<DataTable columns={columns} rows={rows(10)} rowKey={(r) => r.id} pageSize={10} serverSide totalRows={25} onQueryChange={onQueryChange} />);

    await userEvent.click(screen.getByRole("button", { name: "Next" }));

    expect(onQueryChange).toHaveBeenCalledWith(expect.objectContaining({ page: 1 }));
  });

  it("uses the server-reported total for server-side page counts", () => {
    render(<DataTable columns={columns} rows={rows(10)} rowKey={(r) => r.id} pageSize={10} serverSide totalRows={25} />);
    expect(screen.getByText(/Page 1 of 3/)).toBeInTheDocument();
  });

  it("filters across all row fields", async () => {
    render(<DataTable columns={columns} rows={rows(3)} rowKey={(r) => r.id} />);
    await userEvent.type(screen.getByRole("textbox", { name: "Filter table" }), "r1");
    expect(screen.getByText("name-1")).toBeInTheDocument();
    expect(screen.queryByText("name-0")).not.toBeInTheDocument();
  });

  it("starts with the configured latest-first sort", () => {
    render(<DataTable columns={columns} rows={rows(3)} rowKey={(r) => r.id} defaultSortKey="count" />);
    const dataRows = screen.getAllByRole("row").slice(1);
    expect(within(dataRows[0]).getByText("name-0")).toBeInTheDocument();
  });

  it("opens row details from the actions column", async () => {
    render(<DataTable columns={columns} rows={rows(1)} rowKey={(r) => r.id} />);
    await userEvent.click(screen.getByRole("button", { name: "Inspect row r0" }));
    expect(screen.getByRole("dialog", { name: "Row details" })).toBeInTheDocument();
    expect(screen.getByText(/"name": "name-0"/)).toBeInTheDocument();
  });
});
