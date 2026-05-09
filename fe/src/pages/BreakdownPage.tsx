import type { BIDashboardData } from "../api/types";
import { DataTable } from "../components/DataTable";
import { KpiCard } from "../components/KpiCard";
import { avgDiagnostics, formatCompact, formatDate, formatMetric, parseMetric } from "../utils/metrics";

type BreakdownPageProps = {
  data: BIDashboardData;
};

export function BreakdownPage({ data }: BreakdownPageProps) {
  const totalConversions = data.insights.reduce((sum, item) => sum + parseMetric(item.Conversions), 0);
  const totalValue = data.insights.reduce((sum, item) => sum + parseMetric(item.ConversionsValue), 0);
  const searchClicks = data.searchTerms.reduce((sum, item) => sum + item.Clicks, 0);
  const searchSpend = data.searchTerms.reduce((sum, item) => sum + parseMetric(item.Spend), 0);

  return (
    <>
      <section className="kpi-grid compact">
        <KpiCard label="Insight Summary Rows" value={data.insights.length} />
        <KpiCard label="Insight Detail Rows" value={data.insightDetails.length} />
        <KpiCard label="Campaign Diag Rows" value={data.campaignDiagnostics.length} />
        <KpiCard label="Search Terms" value={data.searchTerms.length} />
        <KpiCard label="Conversions / Value" value={`${formatMetric(totalConversions)} / ${formatMetric(totalValue)}`} tone="info" />
        <KpiCard label="Search Clicks / Spend" value={`${formatCompact(searchClicks)} / ${formatMetric(searchSpend)}`} />
        <KpiCard label="Search IS / Top" value={`${formatMetric(avgDiagnostics(data.campaignDiagnostics, "SearchImpressionShare"), 4)} / ${formatMetric(avgDiagnostics(data.campaignDiagnostics, "SearchTopImpressionShare"), 4)}`} />
      </section>

      <DataTable
        title="Insight Detail Drilldown"
        rows={data.insightDetails}
        columns={[
          { key: "date", header: "Date", render: (row) => formatDate(row.StatDate) },
          { key: "platform", header: "Platform", render: (row) => row.Platform },
          { key: "account", header: "Account", render: (row) => row.PlatformAccountID },
          { key: "level", header: "Level", render: (row) => <span className="badge">{row.EntityLevel}</span> },
          { key: "entity", header: "Entity", render: (row) => row.EntityID },
          { key: "campaign", header: "Campaign", render: (row) => row.PlatformCampaignID || "-" },
          { key: "adgroup", header: "Ad Group", render: (row) => row.PlatformAdGroupID || "-" },
          { key: "ad", header: "Ad", render: (row) => row.PlatformAdID || "-" },
          { key: "device", header: "Device", render: (row) => row.Device || "-" },
          { key: "network", header: "Network", render: (row) => row.Network || "-" },
          { key: "impressions", header: "Impr", render: (row) => formatCompact(row.Impressions) },
          { key: "clicks", header: "Clicks", render: (row) => formatCompact(row.Clicks) },
          { key: "ctr", header: "CTR", render: (row) => row.CTR },
          { key: "cpc", header: "CPC", render: (row) => row.CPC },
          { key: "cpm", header: "CPM", render: (row) => row.CPM },
          { key: "spend", header: "Spend", render: (row) => row.Spend },
          { key: "reach", header: "Reach", render: (row) => formatCompact(row.Reach) },
          { key: "conv", header: "Conv", render: (row) => row.Conversions },
          { key: "value", header: "Value", render: (row) => row.ConversionsValue },
          { key: "cpa", header: "CPA", render: (row) => row.CostPerConversion },
        ]}
        getRowKey={(row, index) => `${row.Platform}-${row.EntityID}-${row.StatDate}-${index}`}
        emptyText="No insight detail data"
      />

      <section className="two-column">
        <DataTable
          title="Campaigns"
          rows={data.campaigns}
          columns={[
            { key: "platform", header: "Platform", render: (row) => row.Platform },
            { key: "account", header: "Account", render: (row) => row.AccountID },
            { key: "campaign", header: "Campaign", render: (row) => row.CampaignName || row.PlatformCampaignID },
            { key: "status", header: "Status", render: (row) => <span className={`badge ${row.Status !== "ACTIVE" ? "warn" : ""}`}>{row.Status || "-"}</span> },
            { key: "objective", header: "Objective", render: (row) => row.Objective || "-" },
            { key: "bidding", header: "Bidding", render: (row) => row.BiddingStrategy || row.BuyingType || "-" },
            { key: "budget", header: "Budget", render: (row) => `${row.DailyBudget || row.LifetimeBudget || "-"} ${row.Currency || ""}` },
            { key: "start", header: "Start", render: (row) => formatDate(row.StartTime) },
            { key: "end", header: "End", render: (row) => formatDate(row.EndTime) },
          ]}
          getRowKey={(row, index) => `${row.Platform}-${row.PlatformCampaignID}-${index}`}
          emptyText="No campaign data"
        />
        <DataTable
          title="Campaign Diagnostics"
          rows={data.campaignDiagnostics}
          columns={[
            { key: "date", header: "Date", render: (row) => formatDate(row.StatDate) },
            { key: "platform", header: "Platform", render: (row) => row.Platform },
            { key: "account", header: "Account", render: (row) => row.PlatformAccountID },
            { key: "campaign", header: "Campaign ID", render: (row) => row.PlatformCampaignID },
            { key: "sis", header: "Search IS", render: (row) => row.SearchImpressionShare },
            { key: "top", header: "Top IS", render: (row) => row.SearchTopImpressionShare },
            { key: "abs", header: "Abs Top IS", render: (row) => row.SearchAbsoluteTopImpressionShare },
          ]}
          getRowKey={(row, index) => `${row.Platform}-${row.PlatformCampaignID}-${row.StatDate}-${index}`}
          emptyText="No campaign diagnostic data"
        />
      </section>

      <DataTable
        title="Search Term Diagnostics"
        rows={data.searchTerms}
        columns={[
          { key: "date", header: "Date", render: (row) => formatDate(row.StatDate) },
          { key: "platform", header: "Platform", render: (row) => row.Platform },
          { key: "account", header: "Account", render: (row) => row.PlatformAccountID },
          { key: "campaign", header: "Campaign", render: (row) => row.PlatformCampaignID },
          { key: "adgroup", header: "Ad Group", render: (row) => row.PlatformAdGroupID || "-" },
          { key: "term", header: "Search Term", render: (row) => row.SearchTerm || "-" },
          { key: "match", header: "Match", render: (row) => row.SearchTermMatchType || "-" },
          { key: "impressions", header: "Impr", render: (row) => formatCompact(row.Impressions) },
          { key: "clicks", header: "Clicks", render: (row) => formatCompact(row.Clicks) },
          { key: "spend", header: "Spend", render: (row) => row.Spend },
          { key: "conv", header: "Conv", render: (row) => row.Conversions },
          { key: "value", header: "Value", render: (row) => row.ConversionsValue },
        ]}
        getRowKey={(row, index) => `${row.Platform}-${row.SearchTerm}-${row.StatDate}-${index}`}
        emptyText="No search term data"
      />
    </>
  );
}
