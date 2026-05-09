import type { BIDashboardData } from "../api/types";
import { TrendChart } from "../components/Charts";
import { DataTable, type Column } from "../components/DataTable";
import { KpiCard } from "../components/KpiCard";
import {
  aggregateInsightTrend,
  aggregateUATrend,
  buildOverviewMetrics,
  buildPlatformSummary,
  formatCompact,
  formatDate,
  formatMetric,
  formatMoney,
} from "../utils/metrics";

type OverviewPageProps = {
  data: BIDashboardData;
};

export function OverviewPage({ data }: OverviewPageProps) {
  const metrics = buildOverviewMetrics(data.insights, data.uaReports, data.gameKPIs);
  const platformRows = buildPlatformSummary(data.snapshots, data.insights);

  const platformColumns: Array<Column<(typeof platformRows)[number]>> = [
    { key: "platform", header: "Platform", render: (row) => <span className="badge">{row.platform}</span> },
    { key: "accounts", header: "Accounts", render: (row) => row.accounts },
    { key: "campaigns", header: "Campaigns", render: (row) => row.campaigns },
    { key: "adGroups", header: "Ad Groups", render: (row) => row.adGroups },
    { key: "ads", header: "Ads", render: (row) => row.ads },
    { key: "insights", header: "Insights", render: (row) => row.insights },
    { key: "impressions", header: "Impressions", render: (row) => formatCompact(row.impressions) },
    { key: "clicks", header: "Clicks", render: (row) => formatCompact(row.clicks) },
    { key: "spend", header: "Spend", render: (row) => formatMoney(row.spend) },
  ];

  const uaColumns: Array<Column<(typeof data.uaReports)[number]>> = [
    { key: "date", header: "Date", render: (row) => formatDate(row.stat_date) },
    { key: "platform", header: "Platform", render: (row) => row.platform },
    { key: "account", header: "Account", render: (row) => row.platform_account_id },
    { key: "campaign", header: "Campaign", render: (row) => row.platform_campaign_id || "-" },
    { key: "country", header: "Country", render: (row) => row.country || <span className="muted">待接</span> },
    { key: "os", header: "OS", render: (row) => row.os || <span className="muted">待接</span> },
    { key: "spend", header: "Spend", render: (row) => row.spend },
    { key: "impressions", header: "Impr", render: (row) => formatCompact(row.impressions) },
    { key: "clicks", header: "Clicks", render: (row) => formatCompact(row.clicks) },
    { key: "installs", header: "Installs", render: (row) => formatCompact(row.installs) },
    { key: "cpi", header: "CPI", render: (row) => row.cpi },
    { key: "revenue", header: "Revenue", render: (row) => row.total_revenue },
    { key: "roas", header: "ROAS", render: (row) => row.roas },
    { key: "roi", header: "ROI", render: (row) => row.roi },
  ];

  return (
    <>
      <section className="kpi-grid">
        <KpiCard label="UA Status" value={metrics.dataStatus} tone={data.gameKPIs.length > 0 ? "good" : "warn"} detail={`${data.uaReports.length} ua rows`} />
        <KpiCard label="Spend / Revenue" value={`${formatMoney(metrics.spend)} / ${formatMoney(metrics.totalRevenue)}`} tone="good" detail={`D7 ${formatMoney(metrics.revenueD7)}`} />
        <KpiCard label="Installs / Purchasers" value={`${formatCompact(metrics.installs)} / ${formatCompact(metrics.purchasers)}`} detail={`CPI ${formatMetric(metrics.cpi)}`} />
        <KpiCard label="Retention D1 / D7" value={`${formatMetric(metrics.retentionD1)} / ${formatMetric(metrics.retentionD7)}`} />
        <KpiCard label="ROAS / ROI" value={`${formatMetric(metrics.roas)} / ${formatMetric(metrics.roi)}`} tone="info" />
        <KpiCard label="CTR / CPC / CPM" value={`${formatMetric(metrics.ctr)} / ${formatMetric(metrics.cpc)} / ${formatMetric(metrics.cpm)}`} />
        <KpiCard label="Impressions / Clicks" value={`${formatCompact(metrics.impressions)} / ${formatCompact(metrics.clicks)}`} />
        <KpiCard label="Reach / Frequency" value={`${formatCompact(metrics.reach)} / ${formatMetric(metrics.frequency)}`} />
        <KpiCard label="Conversions" value={formatMetric(metrics.conversions)} detail={`${metrics.activations} activations / ${metrics.registrations} registrations`} />
        <KpiCard label="Snapshots / Campaigns" value={`${data.snapshots.length} / ${data.campaigns.length}`} />
      </section>

      <section className="chart-grid">
        <TrendChart title="Spend Trend" caption="insight summary" points={aggregateInsightTrend(data.insights, "spend")} tone="green" />
        <TrendChart title="Install Trend" caption="ua report" points={aggregateUATrend(data.uaReports, "installs")} tone="blue" />
        <TrendChart title="Revenue Trend" caption="ua report" points={aggregateUATrend(data.uaReports, "revenue")} tone="amber" />
        <TrendChart title="Impression Trend" caption="insight summary" points={aggregateInsightTrend(data.insights, "impressions")} tone="green" />
      </section>

      <section className="two-column">
        <DataTable title="Platform Summary" rows={platformRows} columns={platformColumns} getRowKey={(row) => row.platform} emptyText="No grouped platform data" />
        <DataTable
          title="Account Snapshots"
          rows={data.snapshots}
          columns={[
            { key: "platform", header: "Platform", render: (row) => <span className="badge">{row.Platform}</span> },
            { key: "account", header: "Account", render: (row) => row.AccountID },
            { key: "name", header: "Name", render: (row) => row.AccountName || "-" },
            { key: "source", header: "Source", render: (row) => row.LastSourceMode },
            { key: "campaigns", header: "Campaigns", render: (row) => row.CampaignCount },
            { key: "adgroups", header: "Ad Groups", render: (row) => row.AdGroupCount },
            { key: "ads", header: "Ads", render: (row) => row.AdCount },
            { key: "insights", header: "Insights", render: (row) => row.InsightCount },
          ]}
          getRowKey={(row, index) => `${row.Platform}-${row.AccountID}-${index}`}
          emptyText="No snapshot data"
        />
      </section>

      <DataTable title="UA Overview" rows={data.uaReports} columns={uaColumns} getRowKey={(row, index) => `${row.platform}-${row.entity_id}-${row.stat_date}-${index}`} emptyText="No UA data" />
    </>
  );
}
