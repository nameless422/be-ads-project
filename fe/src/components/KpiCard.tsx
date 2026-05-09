import type { ReactNode } from "react";

type KpiCardProps = {
  label: string;
  value: ReactNode;
  detail?: ReactNode;
  tone?: "default" | "good" | "warn" | "info";
};

export function KpiCard({ label, value, detail, tone = "default" }: KpiCardProps) {
  return (
    <article className={`kpi-card ${tone}`}>
      <div className="kpi-label">{label}</div>
      <div className="kpi-value">{value}</div>
      {detail ? <div className="kpi-detail">{detail}</div> : null}
    </article>
  );
}
