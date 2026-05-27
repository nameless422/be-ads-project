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

type WeightedMetric = {
  total: number;
  weight: number;
};

type BusinessAccumulator = {
  spend: number;
  impressions: number;
  clicks: number;
  conversionsValue: number;
  installs: number;
  purchasers: number;
  purchaseCount: number;
  retentionD1: WeightedMetric;
  ltvD7: WeightedMetric;
};

export type OverviewTrendMetric = "spend" | "installs" | "cpi" | "roas" | "retention_d1" | "cpm" | "ctr" | "ltv_d7";

export type OverviewSummaryDimension = "media_source" | "country" | "device_platform" | "campaign" | "ad_group" | "ad" | "account";

export type OverviewSummaryMetric = "spend" | "installs" | "cpi" | "roas" | "purchase" | "retention_d1" | "cpm" | "ctr" | "ltv_d7";

export const overviewDimensionLabels: Record<OverviewSummaryDimension, string> = {
  media_source: "Media Source",
  country: "Country",
  device_platform: "Device / Platform",
  campaign: "Campaign",
  ad_group: "Ad Group",
  ad: "Ad",
  account: "Account",
};

export const overviewSummaryMetricLabels: Record<OverviewSummaryMetric, string> = {
  spend: "Spend",
  installs: "Installs",
  cpi: "CPI",
  roas: "ROAS",
  purchase: "Purchase",
  retention_d1: "D1 Retention",
  cpm: "CPM",
  ctr: "CTR",
  ltv_d7: "D7 LTV",
};

export const overviewTrendMetricLabels: Record<OverviewTrendMetric, string> = {
  spend: "Spend",
  installs: "Installs",
  cpi: "CPI",
  roas: "ROAS",
  retention_d1: "D1 Retention",
  cpm: "CPM",
  ctr: "CTR",
  ltv_d7: "D7 LTV",
};

function newBusinessAccumulator(): BusinessAccumulator {
  return {
    spend: 0,
    impressions: 0,
    clicks: 0,
    conversionsValue: 0,
    installs: 0,
    purchasers: 0,
    purchaseCount: 0,
    retentionD1: { total: 0, weight: 0 },
    ltvD7: { total: 0, weight: 0 },
  };
}

function addWeightedMetric(metric: WeightedMetric, raw: string | number | undefined | null, preferredWeight: number) {
  if (raw === undefined || raw === null || String(raw).trim() === "") {
    return;
  }
  const value = parseMetric(raw);
  const weight = preferredWeight > 0 ? preferredWeight : 1;
  metric.total += value * weight;
  metric.weight += weight;
}

function weightedAverage(metric: WeightedMetric) {
  return metric.weight > 0 ? metric.total / metric.weight : 0;
}

export function buildBusinessOverviewMetrics(insights: InsightSummaryRow[], uaReports: UAReportRow[]) {
  const acc = newBusinessAccumulator();
  for (const item of insights) {
    acc.spend += parseMetric(item.Spend);
    acc.impressions += item.Impressions;
    acc.clicks += item.Clicks;
    acc.conversionsValue += parseMetric(item.ConversionsValue);
  }
  for (const item of uaReports) {
    acc.installs += item.installs;
    acc.purchasers += item.purchasers;
    acc.purchaseCount += item.purchase_count;
    addWeightedMetric(acc.retentionD1, item.retention_d1, item.installs);
    addWeightedMetric(acc.ltvD7, item.ltv_d7, item.installs);
  }

  return {
    spend: acc.spend,
    impressions: acc.impressions,
    clicks: acc.clicks,
    installs: acc.installs,
    purchasers: acc.purchasers,
    purchaseCount: acc.purchaseCount,
    cpi: acc.installs > 0 ? acc.spend / acc.installs : 0,
    roas: acc.spend > 0 ? acc.conversionsValue / acc.spend : 0,
    retentionD1: weightedAverage(acc.retentionD1),
    cpm: acc.impressions > 0 ? (acc.spend * 1000) / acc.impressions : 0,
    ctr: acc.impressions > 0 ? acc.clicks / acc.impressions : 0,
    ltvD7: weightedAverage(acc.ltvD7),
  };
}

type DailyAccumulator = BusinessAccumulator;

function dateLabelFromAny(raw: string | undefined) {
  return formatDate(raw);
}

function metricValueFromAccumulator(acc: DailyAccumulator, metric: OverviewTrendMetric) {
  switch (metric) {
    case "spend":
      return acc.spend;
    case "installs":
      return acc.installs;
    case "cpi":
      return acc.installs > 0 ? acc.spend / acc.installs : 0;
    case "roas":
      return acc.spend > 0 ? acc.conversionsValue / acc.spend : 0;
    case "retention_d1":
      return weightedAverage(acc.retentionD1);
    case "cpm":
      return acc.impressions > 0 ? (acc.spend * 1000) / acc.impressions : 0;
    case "ctr":
      return acc.impressions > 0 ? acc.clicks / acc.impressions : 0;
    case "ltv_d7":
      return weightedAverage(acc.ltvD7);
    default:
      return 0;
  }
}

export function aggregateOverviewTrend(insights: InsightSummaryRow[], uaReports: UAReportRow[], metric: OverviewTrendMetric) {
  const grouped = new Map<string, DailyAccumulator>();
  for (const item of insights) {
    const label = dateLabelFromAny(item.StatDate);
    const acc = grouped.get(label) ?? newBusinessAccumulator();
    acc.spend += parseMetric(item.Spend);
    acc.impressions += item.Impressions;
    acc.clicks += item.Clicks;
    acc.conversionsValue += parseMetric(item.ConversionsValue);
    grouped.set(label, acc);
  }
  for (const item of uaReports) {
    const label = dateLabelFromAny(item.stat_date);
    const acc = grouped.get(label) ?? newBusinessAccumulator();
    acc.installs += item.installs;
    acc.purchasers += item.purchasers;
    acc.purchaseCount += item.purchase_count;
    addWeightedMetric(acc.retentionD1, item.retention_d1, item.installs);
    addWeightedMetric(acc.ltvD7, item.ltv_d7, item.installs);
    grouped.set(label, acc);
  }
  return [...grouped.entries()]
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([label, acc]) => ({ label, value: metricValueFromAccumulator(acc, metric) }));
}

function dimensionValue(row: UAReportRow, dimension: OverviewSummaryDimension) {
  switch (dimension) {
    case "media_source":
      return row.platform || "unknown";
    case "country":
      return row.country || "未接入";
    case "device_platform":
      return row.device || row.os || "未接入";
    case "campaign":
      return row.platform_campaign_id || "-";
    case "ad_group":
      return row.platform_ad_group_id || "-";
    case "ad":
      return row.platform_ad_id || "-";
    case "account":
      return row.platform_account_id || "-";
    default:
      return "-";
  }
}

export type OverviewSummaryRow = {
  key: string;
  dimensions: Record<OverviewSummaryDimension, string>;
  spend: number;
  installs: number;
  cpi: number;
  roas: number;
  purchase: number;
  retentionD1: number;
  cpm: number;
  ctr: number;
  ltvD7: number;
};

export function buildOverviewSummaryRows(rows: UAReportRow[], dimensions: OverviewSummaryDimension[]) {
  const grouped = new Map<string, { dimensions: Record<OverviewSummaryDimension, string>; acc: BusinessAccumulator }>();
  for (const row of rows) {
    const values = {} as Record<OverviewSummaryDimension, string>;
    for (const dimension of dimensions) {
      values[dimension] = dimensionValue(row, dimension);
    }
    const key = dimensions.map((dimension) => values[dimension]).join("|") || "all";
    const item = grouped.get(key) ?? { dimensions: values, acc: newBusinessAccumulator() };
    item.acc.spend += parseMetric(row.spend);
    item.acc.impressions += row.impressions;
    item.acc.clicks += row.clicks;
    item.acc.conversionsValue += parseMetric(row.conversions_value);
    item.acc.installs += row.installs;
    item.acc.purchasers += row.purchasers;
    item.acc.purchaseCount += row.purchase_count;
    addWeightedMetric(item.acc.retentionD1, row.retention_d1, row.installs);
    addWeightedMetric(item.acc.ltvD7, row.ltv_d7, row.installs);
    grouped.set(key, item);
  }

  return [...grouped.entries()]
    .map(([key, item]) => ({
      key,
      dimensions: item.dimensions,
      spend: item.acc.spend,
      installs: item.acc.installs,
      cpi: item.acc.installs > 0 ? item.acc.spend / item.acc.installs : 0,
      roas: item.acc.spend > 0 ? item.acc.conversionsValue / item.acc.spend : 0,
      purchase: item.acc.purchasers || item.acc.purchaseCount,
      retentionD1: weightedAverage(item.acc.retentionD1),
      cpm: item.acc.impressions > 0 ? (item.acc.spend * 1000) / item.acc.impressions : 0,
      ctr: item.acc.impressions > 0 ? item.acc.clicks / item.acc.impressions : 0,
      ltvD7: weightedAverage(item.acc.ltvD7),
    }))
    .sort((a, b) => b.spend - a.spend);
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
