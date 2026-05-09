import type {
  AccountSnapshot,
  CampaignDiagnosticRow,
  GameKPIRecord,
  InsightSummaryRow,
  UAFieldDefinition,
  UAReportRow,
} from "../api/types";

export function parseMetric(value: string | number | undefined | null) {
  if (typeof value === "number") {
    return Number.isFinite(value) ? value : 0;
  }
  const parsed = Number.parseFloat(String(value ?? "").trim());
  return Number.isFinite(parsed) ? parsed : 0;
}

export function formatCompact(value: number) {
  return new Intl.NumberFormat("en", {
    notation: Math.abs(value) >= 10000 ? "compact" : "standard",
    maximumFractionDigits: Math.abs(value) >= 10000 ? 1 : 0,
  }).format(value);
}

export function formatMoney(value: number) {
  return new Intl.NumberFormat("en", {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(value);
}

export function formatMetric(value: number, digits = 2) {
  return value.toFixed(digits);
}

export function formatDate(value: string | undefined) {
  if (!value || value.startsWith("0001-01-01")) {
    return "-";
  }
  return value.slice(0, 10);
}

export function countDistinct(values: Array<string | undefined>) {
  const set = new Set(values.map((value) => value?.trim()).filter(Boolean));
  return set.size;
}

export function avgMetric(total: number, count: number) {
  if (count <= 0) {
    return 0;
  }
  return total / count;
}

export function buildOverviewMetrics(insights: InsightSummaryRow[], uaReports: UAReportRow[], gameKPIs: GameKPIRecord[]) {
  const spend = insights.reduce((sum, item) => sum + parseMetric(item.Spend), 0);
  const impressions = insights.reduce((sum, item) => sum + item.Impressions, 0);
  const clicks = insights.reduce((sum, item) => sum + item.Clicks, 0);
  const reach = insights.reduce((sum, item) => sum + item.Reach, 0);
  const conversions = insights.reduce((sum, item) => sum + parseMetric(item.Conversions), 0);
  const totalRevenue = uaReports.reduce((sum, item) => sum + parseMetric(item.total_revenue), 0);
  const revenueD7 = uaReports.reduce((sum, item) => sum + parseMetric(item.revenue_d7), 0);
  const installs = uaReports.reduce((sum, item) => sum + item.installs, 0);
  const activations = uaReports.reduce((sum, item) => sum + item.activations, 0);
  const registrations = uaReports.reduce((sum, item) => sum + item.registrations, 0);
  const purchasers = uaReports.reduce((sum, item) => sum + item.purchasers, 0);
  const retentionD1 = uaReports.reduce((sum, item) => sum + parseMetric(item.retention_d1), 0);
  const retentionD7 = uaReports.reduce((sum, item) => sum + parseMetric(item.retention_d7), 0);
  const roi = uaReports.reduce((sum, item) => sum + parseMetric(item.roi), 0);
  const roas = uaReports.reduce((sum, item) => sum + parseMetric(item.roas), 0);
  const count = uaReports.length;

  return {
    spend,
    impressions,
    clicks,
    reach,
    conversions,
    totalRevenue,
    revenueD7,
    installs,
    activations,
    registrations,
    purchasers,
    ctr: impressions > 0 ? clicks / impressions : 0,
    cpc: clicks > 0 ? spend / clicks : 0,
    cpm: impressions > 0 ? (spend * 1000) / impressions : 0,
    frequency: reach > 0 ? impressions / reach : 0,
    cpi: installs > 0 ? spend / installs : 0,
    retentionD1: avgMetric(retentionD1, count),
    retentionD7: avgMetric(retentionD7, count),
    roi: avgMetric(roi, count),
    roas: avgMetric(roas, count),
    dataStatus: gameKPIs.length > 0 ? "广告侧 + 游戏内 KPI 已合并" : "广告侧可用，游戏内字段待接入",
  };
}

export function aggregateInsightTrend(insights: InsightSummaryRow[], metric: "impressions" | "clicks" | "spend") {
  const grouped = new Map<string, number>();
  for (const item of insights) {
    const label = formatDate(item.StatDate);
    const value = metric === "impressions" ? item.Impressions : metric === "clicks" ? item.Clicks : parseMetric(item.Spend);
    grouped.set(label, (grouped.get(label) ?? 0) + value);
  }
  return [...grouped.entries()].sort(([a], [b]) => a.localeCompare(b)).map(([label, value]) => ({ label, value }));
}

export function aggregateUATrend(rows: UAReportRow[], metric: "installs" | "revenue") {
  const grouped = new Map<string, number>();
  for (const item of rows) {
    const label = formatDate(item.stat_date);
    const value = metric === "installs" ? item.installs : parseMetric(item.total_revenue);
    grouped.set(label, (grouped.get(label) ?? 0) + value);
  }
  return [...grouped.entries()].sort(([a], [b]) => a.localeCompare(b)).map(([label, value]) => ({ label, value }));
}

export function buildPlatformSummary(snapshots: AccountSnapshot[], insights: InsightSummaryRow[]) {
  const rows = new Map<string, { accounts: Set<string>; campaigns: number; adGroups: number; ads: number; insights: number; impressions: number; clicks: number; spend: number }>();

  for (const snapshot of snapshots) {
    const key = snapshot.Platform;
    const row = rows.get(key) ?? { accounts: new Set<string>(), campaigns: 0, adGroups: 0, ads: 0, insights: 0, impressions: 0, clicks: 0, spend: 0 };
    row.accounts.add(snapshot.AccountID);
    row.campaigns += snapshot.CampaignCount;
    row.adGroups += snapshot.AdGroupCount;
    row.ads += snapshot.AdCount;
    row.insights += snapshot.InsightCount;
    rows.set(key, row);
  }

  for (const insight of insights) {
    const key = insight.Platform;
    const row = rows.get(key) ?? { accounts: new Set<string>(), campaigns: 0, adGroups: 0, ads: 0, insights: 0, impressions: 0, clicks: 0, spend: 0 };
    row.impressions += insight.Impressions;
    row.clicks += insight.Clicks;
    row.spend += parseMetric(insight.Spend);
    rows.set(key, row);
  }

  return [...rows.entries()].sort(([a], [b]) => a.localeCompare(b)).map(([platform, row]) => ({
    platform,
    accounts: row.accounts.size,
    campaigns: row.campaigns,
    adGroups: row.adGroups,
    ads: row.ads,
    insights: row.insights,
    impressions: row.impressions,
    clicks: row.clicks,
    spend: row.spend,
  }));
}

export function buildFieldStats(fields: UAFieldDefinition[]) {
  return {
    available: fields.filter((item) => item.status === "available").length,
    integrationReady: fields.filter((item) => item.status === "integration_ready").length,
    planned: fields.filter((item) => item.status === "planned").length,
  };
}

export function avgDiagnostics(rows: CampaignDiagnosticRow[], field: keyof CampaignDiagnosticRow) {
  if (rows.length === 0) {
    return 0;
  }
  return rows.reduce((sum, row) => sum + parseMetric(row[field] as string), 0) / rows.length;
}
