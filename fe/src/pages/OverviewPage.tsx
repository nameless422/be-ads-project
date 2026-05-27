import { useMemo, useState } from "react";
import type { BIDashboardData } from "../api/types";
import { TrendChart } from "../components/Charts";
import { DataTable, type Column } from "../components/DataTable";
import { KpiCard } from "../components/KpiCard";
import {
  aggregateOverviewTrend,
  buildBusinessOverviewMetrics,
  buildOverviewSummaryRows,
  formatCompact,
  formatMetric,
  formatMoney,
  overviewDimensionLabels,
  overviewSummaryMetricLabels,
  overviewTrendMetricLabels,
  type OverviewSummaryDimension,
  type OverviewSummaryMetric,
  type OverviewSummaryRow,
  type OverviewTrendMetric,
} from "../utils/metrics";

type OverviewPageProps = {
  data: BIDashboardData;
};

const trendMetrics: OverviewTrendMetric[] = ["spend", "installs", "cpi", "roas", "retention_d1", "cpm", "ctr", "ltv_d7"];
const unavailableTrendMetrics = ["D0 ROAS", "D7 ROAS"];

const dimensionOptions: OverviewSummaryDimension[] = ["media_source", "country", "device_platform", "campaign", "ad_group", "ad", "account"];
const summaryMetricOptions: OverviewSummaryMetric[] = ["spend", "installs", "cpi", "roas", "purchase", "retention_d1", "cpm", "ctr", "ltv_d7"];

export function OverviewPage({ data }: OverviewPageProps) {
  const [trendMetric, setTrendMetric] = useState<OverviewTrendMetric>("spend");
  const [dimensions, setDimensions] = useState<OverviewSummaryDimension[]>(["media_source", "country"]);
  const [summaryMetrics, setSummaryMetrics] = useState<OverviewSummaryMetric[]>(["spend", "installs", "cpi", "roas", "purchase", "retention_d1", "cpm", "ctr", "ltv_d7"]);

  const metrics = useMemo(() => buildBusinessOverviewMetrics(data.insights, data.uaReports), [data.insights, data.uaReports]);
  const trendPoints = useMemo(() => aggregateOverviewTrend(data.insights, data.uaReports, trendMetric), [data.insights, data.uaReports, trendMetric]);
  const summaryRows = useMemo(() => buildOverviewSummaryRows(data.uaReports, dimensions), [data.uaReports, dimensions]);
  const summaryColumns = useMemo(() => buildSummaryColumns(dimensions, summaryMetrics), [dimensions, summaryMetrics]);

  function toggleDimension(dimension: OverviewSummaryDimension) {
    setDimensions((current) => {
      if (current.includes(dimension)) {
        return current.length > 1 ? current.filter((item) => item !== dimension) : current;
      }
      return [...current, dimension];
    });
  }

  function toggleSummaryMetric(metric: OverviewSummaryMetric) {
    setSummaryMetrics((current) => {
      if (current.includes(metric)) {
        return current.length > 1 ? current.filter((item) => item !== metric) : current;
      }
      return [...current, metric];
    });
  }

  const purchaseValue = metrics.purchasers > 0 ? formatCompact(metrics.purchasers) : metrics.purchaseCount > 0 ? formatCompact(metrics.purchaseCount) : "未接入";
  const purchaseDetail = metrics.purchasers > 0 ? "purchasers" : metrics.purchaseCount > 0 ? "purchase_count" : "game KPI required";
  const ltvD7Value = metrics.ltvD7 > 0 ? formatMetric(metrics.ltvD7) : "未接入";

  return (
    <>
      <section className="kpi-grid business-kpi-grid">
        <KpiCard label="Spend" value={formatMoney(metrics.spend)} detail="ad spend" tone="good" />
        <KpiCard label="Installs" value={metrics.installs > 0 ? formatCompact(metrics.installs) : "未接入"} detail="game KPI" tone={metrics.installs > 0 ? "default" : "warn"} />
        <KpiCard label="CPI" value={metrics.installs > 0 ? formatMetric(metrics.cpi) : "未接入"} detail="spend / installs" tone={metrics.installs > 0 ? "default" : "warn"} />
        <KpiCard label="ROAS" value={formatMetric(metrics.roas)} detail="ad conversion value / spend" tone="info" />
        <KpiCard label="Purchase" value={purchaseValue} detail={purchaseDetail} tone={metrics.purchasers > 0 || metrics.purchaseCount > 0 ? "default" : "warn"} />
        <KpiCard label="CPA" value="待确认" detail="purchase basis pending" tone="warn" />
        <KpiCard label="D1 Retention" value={metrics.retentionD1 > 0 ? formatMetric(metrics.retentionD1) : "未接入"} detail="retention_d1" tone={metrics.retentionD1 > 0 ? "default" : "warn"} />
        <KpiCard label="CPM" value={formatMetric(metrics.cpm)} detail="spend * 1000 / impressions" />
        <KpiCard label="CTR" value={formatMetric(metrics.ctr)} detail="clicks / impressions" />
        <KpiCard label="CVR" value="待确认" detail="install or ad conversion basis" tone="warn" />
        <KpiCard label="D0 LTV" value="Phase 2" detail="requires D0 revenue/LTV" tone="warn" />
        <KpiCard label="D7 LTV" value={ltvD7Value} detail="ltv_d7" tone={metrics.ltvD7 > 0 ? "default" : "warn"} />
      </section>

      <section className="panel trend-module">
        <div className="panel-head">
          <h2>Daily Trend</h2>
          <span>by stat_date</span>
        </div>
        <div className="metric-toggle-row" aria-label="Trend metric">
          {trendMetrics.map((metric) => (
            <button key={metric} type="button" className={metric === trendMetric ? "metric-toggle active" : "metric-toggle"} onClick={() => setTrendMetric(metric)}>
              {overviewTrendMetricLabels[metric]}
            </button>
          ))}
          {unavailableTrendMetrics.map((metric) => (
            <button key={metric} type="button" className="metric-toggle unavailable" disabled>
              {metric}
            </button>
          ))}
        </div>
      </section>
      <TrendChart title={overviewTrendMetricLabels[trendMetric]} caption="daily" points={trendPoints} tone={trendTone(trendMetric)} />

      <section className="panel summary-builder">
        <div className="panel-head">
          <h2>UA Summary</h2>
          <span>{summaryRows.length} group(s)</span>
        </div>
        <div className="summary-controls">
          <div className="summary-control-group">
            <div className="summary-control-title">Dimensions</div>
            <div className="summary-chip-row">
              {dimensionOptions.map((dimension) => (
                <label key={dimension} className="chip-control">
                  <input type="checkbox" checked={dimensions.includes(dimension)} onChange={() => toggleDimension(dimension)} />
                  <span>{overviewDimensionLabels[dimension]}</span>
                </label>
              ))}
            </div>
          </div>
          <div className="summary-control-group">
            <div className="summary-control-title">Metrics</div>
            <div className="summary-chip-row">
              {summaryMetricOptions.map((metric) => (
                <label key={metric} className="chip-control">
                  <input type="checkbox" checked={summaryMetrics.includes(metric)} onChange={() => toggleSummaryMetric(metric)} />
                  <span>{overviewSummaryMetricLabels[metric]}</span>
                </label>
              ))}
            </div>
          </div>
        </div>
      </section>
      <DataTable
        title="Grouped UA Metrics"
        caption="media source + region by default"
        rows={summaryRows}
        columns={summaryColumns}
        getRowKey={(row, index) => `${row.key}-${index}`}
        emptyText="No UA summary data"
      />
    </>
  );
}

function buildSummaryColumns(dimensions: OverviewSummaryDimension[], metrics: OverviewSummaryMetric[]): Array<Column<OverviewSummaryRow>> {
  const dimensionColumns: Array<Column<OverviewSummaryRow>> = dimensions.map((dimension) => ({
    key: dimension,
    header: overviewDimensionLabels[dimension],
    render: (row) => (dimension === "media_source" ? <span className="badge">{row.dimensions[dimension]}</span> : row.dimensions[dimension]),
  }));

  const metricColumns: Array<Column<OverviewSummaryRow>> = metrics.map((metric) => ({
    key: metric,
    header: overviewSummaryMetricLabels[metric],
    render: (row) => renderSummaryMetric(row, metric),
  }));

  return [...dimensionColumns, ...metricColumns];
}

function renderSummaryMetric(row: OverviewSummaryRow, metric: OverviewSummaryMetric) {
  switch (metric) {
    case "spend":
      return formatMoney(row.spend);
    case "installs":
      return formatCompact(row.installs);
    case "cpi":
      return row.installs > 0 ? formatMetric(row.cpi) : <span className="muted">未接入</span>;
    case "roas":
      return formatMetric(row.roas);
    case "purchase":
      return row.purchase > 0 ? formatCompact(row.purchase) : <span className="muted">未接入</span>;
    case "retention_d1":
      return row.retentionD1 > 0 ? formatMetric(row.retentionD1) : <span className="muted">未接入</span>;
    case "cpm":
      return formatMetric(row.cpm);
    case "ctr":
      return formatMetric(row.ctr);
    case "ltv_d7":
      return row.ltvD7 > 0 ? formatMetric(row.ltvD7) : <span className="muted">未接入</span>;
    default:
      return "-";
  }
}

function trendTone(metric: OverviewTrendMetric) {
  if (metric === "installs" || metric === "retention_d1" || metric === "ltv_d7") {
    return "blue";
  }
  if (metric === "cpi" || metric === "cpm") {
    return "amber";
  }
  return "green";
}
