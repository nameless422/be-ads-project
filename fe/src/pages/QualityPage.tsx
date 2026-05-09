import type { BIDashboardData } from "../api/types";
import { DataTable } from "../components/DataTable";
import { KpiCard } from "../components/KpiCard";
import { buildFieldStats, countDistinct, formatCompact, formatDate, formatMetric, parseMetric } from "../utils/metrics";

type QualityPageProps = {
  data: BIDashboardData;
};

export function QualityPage({ data }: QualityPageProps) {
  const countryCount = countDistinct([...data.uaReports.map((item) => item.country), ...data.gameKPIs.map((item) => item.country)]);
  const osCount = countDistinct([...data.uaReports.map((item) => item.os), ...data.gameKPIs.map((item) => item.os)]);
  const rows = data.uaReports.length;
  const avgActivation = rows > 0 ? data.uaReports.reduce((sum, item) => sum + parseMetric(item.activation_rate), 0) / rows : 0;
  const avgRegistration = rows > 0 ? data.uaReports.reduce((sum, item) => sum + parseMetric(item.registration_rate), 0) / rows : 0;
  const avgPayer = rows > 0 ? data.uaReports.reduce((sum, item) => sum + parseMetric(item.payer_rate), 0) / rows : 0;
  const avgHighValue = rows > 0 ? data.uaReports.reduce((sum, item) => sum + parseMetric(item.high_value_payer_ratio), 0) / rows : 0;
  const avgD3 = rows > 0 ? data.uaReports.reduce((sum, item) => sum + parseMetric(item.retention_d3), 0) / rows : 0;
  const avgD30 = rows > 0 ? data.uaReports.reduce((sum, item) => sum + parseMetric(item.retention_d30), 0) / rows : 0;
  const avgArpu = rows > 0 ? data.uaReports.reduce((sum, item) => sum + parseMetric(item.arpu), 0) / rows : 0;
  const avgArppu = rows > 0 ? data.uaReports.reduce((sum, item) => sum + parseMetric(item.arppu), 0) / rows : 0;
  const avgOnline = rows > 0 ? data.uaReports.reduce((sum, item) => sum + item.avg_online_duration_seconds, 0) / rows : 0;
  const avgTask = rows > 0 ? data.uaReports.reduce((sum, item) => sum + parseMetric(item.task_completion_rate), 0) / rows : 0;
  const avgLtvCpi = rows > 0 ? data.uaReports.reduce((sum, item) => sum + parseMetric(item.ltv_to_cpi_ratio), 0) / rows : 0;
  const fieldStats = buildFieldStats(data.uaFields);

  return (
    <>
      <section className="kpi-grid compact">
        <KpiCard label="Country / OS" value={`${countryCount} / ${osCount}`} />
        <KpiCard label="Activation / Registration" value={`${formatMetric(avgActivation)} / ${formatMetric(avgRegistration)}`} />
        <KpiCard label="Payer / High Value" value={`${formatMetric(avgPayer)} / ${formatMetric(avgHighValue)}`} />
        <KpiCard label="Retention D3 / D30" value={`${formatMetric(avgD3)} / ${formatMetric(avgD30)}`} />
        <KpiCard label="ARPU / ARPPU" value={`${formatMetric(avgArpu)} / ${formatMetric(avgArppu)}`} />
        <KpiCard label="Avg Online Seconds" value={formatMetric(avgOnline, 0)} />
        <KpiCard label="Task Completion" value={formatMetric(avgTask)} />
        <KpiCard label="LTV/CPI" value={formatMetric(avgLtvCpi)} tone="info" />
        <KpiCard label="Available / Ready / Planned" value={`${fieldStats.available} / ${fieldStats.integrationReady} / ${fieldStats.planned}`} />
        <KpiCard label="Fraud/Bounce Signals" value="planned" tone="warn" />
      </section>

      <DataTable
        title="UA Quality Signals"
        rows={data.uaReports}
        columns={[
          { key: "date", header: "Date", render: (row) => formatDate(row.stat_date) },
          { key: "platform", header: "Platform", render: (row) => row.platform },
          { key: "country", header: "Country", render: (row) => row.country || "-" },
          { key: "os", header: "OS", render: (row) => row.os || "-" },
          { key: "network", header: "Network", render: (row) => row.network || "-" },
          { key: "placement", header: "Placement", render: (row) => row.placement || "-" },
          { key: "frequency", header: "Frequency", render: (row) => row.frequency },
          { key: "ctr", header: "CTR", render: (row) => row.ctr },
          { key: "cpi", header: "CPI", render: (row) => row.cpi },
          { key: "activation", header: "Activation Rate", render: (row) => row.activation_rate },
          { key: "registration", header: "Registration Rate", render: (row) => row.registration_rate },
          { key: "payer", header: "Payer Rate", render: (row) => row.payer_rate },
          { key: "d1", header: "D1", render: (row) => row.retention_d1 },
          { key: "d3", header: "D3", render: (row) => row.retention_d3 },
          { key: "d7", header: "D7", render: (row) => row.retention_d7 },
          { key: "d30", header: "D30", render: (row) => row.retention_d30 },
          { key: "arpu", header: "ARPU", render: (row) => row.arpu },
          { key: "arppu", header: "ARPPU", render: (row) => row.arppu },
          { key: "ltv7", header: "LTV D7", render: (row) => row.ltv_d7 },
          { key: "ltv30", header: "LTV D30", render: (row) => row.ltv_d30 },
          { key: "ltvcpi", header: "LTV/CPI", render: (row) => row.ltv_to_cpi_ratio },
        ]}
        getRowKey={(row, index) => `${row.platform}-${row.entity_id}-${row.stat_date}-${index}`}
        emptyText="No UA quality data"
      />

      <section className="two-column">
        <DataTable
          title="Game-side Quality Signals"
          rows={data.gameKPIs}
          columns={[
            { key: "date", header: "Date", render: (row) => formatDate(row.stat_date) },
            { key: "platform", header: "Platform", render: (row) => row.platform },
            { key: "country", header: "Country", render: (row) => row.country || "-" },
            { key: "os", header: "OS", render: (row) => row.os || "-" },
            { key: "placement", header: "Placement", render: (row) => row.placement || "-" },
            { key: "creative", header: "Creative ID", render: (row) => row.creative_id || "-" },
            { key: "installs", header: "Installs", render: (row) => formatCompact(row.installs) },
            { key: "activations", header: "Activations", render: (row) => formatCompact(row.activations) },
            { key: "registrations", header: "Registrations", render: (row) => formatCompact(row.registrations) },
            { key: "purchasers", header: "Purchasers", render: (row) => formatCompact(row.purchasers) },
            { key: "d1", header: "D1", render: (row) => row.retention_d1 },
            { key: "d7", header: "D7", render: (row) => row.retention_d7 },
            { key: "d30", header: "D30", render: (row) => row.retention_d30 },
            { key: "revenue", header: "Revenue D7", render: (row) => row.revenue_d7 },
            { key: "online", header: "Online Seconds", render: (row) => row.avg_online_duration_seconds },
          ]}
          getRowKey={(row, index) => `${row.platform}-${row.platform_account_id}-${row.stat_date}-${index}`}
          emptyText="No game-side quality data. Bounce/fraud 仍待外部信号接入。"
        />
        <DataTable
          title="Field Readiness"
          rows={data.uaFields}
          columns={[
            { key: "key", header: "Field", render: (row) => row.key },
            { key: "label", header: "Label", render: (row) => row.label },
            { key: "category", header: "Category", render: (row) => row.category },
            { key: "status", header: "Status", render: (row) => <span className={`badge ${row.status === "planned" ? "warn" : ""}`}>{row.status}</span> },
            { key: "source", header: "Source", render: (row) => row.source },
            { key: "notes", header: "Notes", render: (row) => row.notes || row.example_api || "-" },
          ]}
          getRowKey={(row) => row.key}
          emptyText="No field catalog data"
        />
      </section>
    </>
  );
}
