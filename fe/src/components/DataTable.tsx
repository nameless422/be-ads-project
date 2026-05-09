import type { ReactNode } from "react";

export type Column<T> = {
  key: string;
  header: string;
  width?: string;
  render: (row: T) => ReactNode;
};

type DataTableProps<T> = {
  title: string;
  caption?: string;
  rows: T[];
  columns: Array<Column<T>>;
  emptyText?: string;
  getRowKey: (row: T, index: number) => string;
};

export function DataTable<T>({ title, caption, rows, columns, emptyText = "No data", getRowKey }: DataTableProps<T>) {
  return (
    <section className="panel table-panel">
      <div className="panel-head">
        <h2>{title}</h2>
        <span>{caption ?? `${rows.length} row(s)`}</span>
      </div>
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              {columns.map((column) => (
                <th key={column.key} style={{ width: column.width }}>
                  {column.header}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.length > 0 ? (
              rows.map((row, index) => (
                <tr key={getRowKey(row, index)}>
                  {columns.map((column) => (
                    <td key={column.key}>{column.render(row)}</td>
                  ))}
                </tr>
              ))
            ) : (
              <tr>
                <td colSpan={columns.length} className="muted empty-cell">
                  {emptyText}
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </section>
  );
}
