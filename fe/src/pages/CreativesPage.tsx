import type { BIDashboardData } from "../api/types";
import { DataTable } from "../components/DataTable";
import { KpiCard } from "../components/KpiCard";
import { countDistinct, formatCompact, formatDate, parseMetric, formatMetric } from "../utils/metrics";

type CreativesPageProps = {
  data: BIDashboardData;
};

export function CreativesPage({ data }: CreativesPageProps) {
  const creativeCount = countDistinct([...data.uaReports.map((item) => item.creative_id), ...data.gameKPIs.map((item) => item.creative_id)]);
  const creativeTypeCount = countDistinct([...data.uaReports.map((item) => item.creative_type), ...data.gameKPIs.map((item) => item.creative_type)]);
  const placementCount = countDistinct([...data.uaReports.map((item) => item.placement), ...data.gameKPIs.map((item) => item.placement)]);
  const revenueD7 = data.uaReports.reduce((sum, item) => sum + parseMetric(item.revenue_d7), 0);
  const revenueD30 = data.uaReports.reduce((sum, item) => sum + parseMetric(item.revenue_d30), 0);
  const tutorial = data.uaReports.reduce((sum, item) => sum + item.tutorial_completions, 0);
  const roles = data.uaReports.reduce((sum, item) => sum + item.role_creations, 0);
  const purchases = data.uaReports.reduce((sum, item) => sum + item.purchase_count, 0);

  return (
    <>
      <section className="kpi-grid compact">
        <KpiCard label="Creative IDs" value={creativeCount} />
        <KpiCard label="Creative Types" value={creativeTypeCount} />
        <KpiCard label="Placements" value={placementCount} />
        <KpiCard label="Revenue D7 / D30" value={`${formatMetric(revenueD7)} / ${formatMetric(revenueD30)}`} tone="good" />
        <KpiCard label="Tutorial / Role Create" value={`${formatCompact(tutorial)} / ${formatCompact(roles)}`} />
        <KpiCard label="Purchase Count" value={formatCompact(purchases)} />
      </section>

      <DataTable
        title="Creative Performance"
        rows={data.uaReports}
        columns={[
          { key: "date", header: "Date", render: (row) => formatDate(row.stat_date) },
          { key: "platform", header: "Platform", render: (row) => row.platform },
          { key: "campaign", header: "Campaign", render: (row) => row.platform_campaign_id || "-" },
          { key: "adgroup", header: "Ad Group", render: (row) => row.platform_ad_group_id || "-" },
          { key: "ad", header: "Ad", render: (row) => row.platform_ad_id || "-" },
          { key: "creative", header: "Creative ID", render: (row) => row.creative_id || "-" },
          { key: "type", header: "Creative Type", render: (row) => row.creative_type || "-" },
          { key: "placement", header: "Placement", render: (row) => row.placement || "-" },
          { key: "goal", header: "Goal", render: (row) => row.optimization_goal || "-" },
          { key: "bid", header: "Bid", render: (row) => row.bid_type || "-" },
          { key: "targeting", header: "Targeting", render: (row) => row.targeting || "-" },
          { key: "spend", header: "Spend", render: (row) => row.spend },
          { key: "ctr", header: "CTR", render: (row) => row.ctr },
          { key: "installs", header: "Installs", render: (row) => formatCompact(row.installs) },
          { key: "cpi", header: "CPI", render: (row) => row.cpi },
          { key: "d1", header: "D1", render: (row) => row.retention_d1 },
          { key: "d7", header: "D7", render: (row) => row.retention_d7 },
          { key: "revenue", header: "Revenue D7", render: (row) => row.revenue_d7 },
          { key: "roas", header: "ROAS", render: (row) => row.roas },
          { key: "roi", header: "ROI", render: (row) => row.roi },
        ]}
        getRowKey={(row, index) => `${row.platform}-${row.entity_id}-${row.stat_date}-${index}`}
        emptyText="No creative-side UA data"
      />

      <DataTable
        title="Creative-linked Game KPIs"
        rows={data.gameKPIs}
        columns={[
          { key: "date", header: "Date", render: (row) => formatDate(row.stat_date) },
          { key: "platform", header: "Platform", render: (row) => row.platform },
          { key: "creative", header: "Creative ID", render: (row) => row.creative_id || "-" },
          { key: "type", header: "Creative Type", render: (row) => row.creative_type || "-" },
          { key: "placement", header: "Placement", render: (row) => row.placement || "-" },
          { key: "targeting", header: "Targeting", render: (row) => row.targeting || "-" },
          { key: "installs", header: "Installs", render: (row) => formatCompact(row.installs) },
          { key: "activations", header: "Activations", render: (row) => formatCompact(row.activations) },
          { key: "registrations", header: "Registrations", render: (row) => formatCompact(row.registrations) },
          { key: "purchasers", header: "Purchasers", render: (row) => formatCompact(row.purchasers) },
          { key: "purchaseCount", header: "Purchase Count", render: (row) => formatCompact(row.purchase_count) },
          { key: "tutorial", header: "Tutorial", render: (row) => formatCompact(row.tutorial_completions) },
          { key: "role", header: "Role Create", render: (row) => formatCompact(row.role_creations) },
          { key: "level", header: "LevelX", render: (row) => formatCompact(row.level_x_users) },
          { key: "revenueD1", header: "Revenue D1", render: (row) => row.revenue_d1 },
          { key: "revenueD7", header: "Revenue D7", render: (row) => row.revenue_d7 },
          { key: "revenueD30", header: "Revenue D30", render: (row) => row.revenue_d30 },
        ]}
        getRowKey={(row, index) => `${row.platform}-${row.creative_id}-${row.stat_date}-${index}`}
        emptyText="No creative-linked game KPI data. 可通过 POST /api/bi/game-kpis 接入。"
      />
    </>
  );
}
