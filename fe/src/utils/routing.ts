import type { FilterState, PageKey } from "../api/types";

export const pageMeta: Record<PageKey, { title: string; description: string }> = {
  overview: { title: "Overview", description: "UA 核心指标、按天趋势和维度汇总。" },
  breakdown: { title: "Breakdown", description: "按实体、campaign、搜索词和诊断指标定位问题。" },
  creatives: { title: "Creatives", description: "围绕素材、版位、目标和投放质量做优化判断。" },
  quality: { title: "Quality", description: "查看留存、付费、LTV、行为和异常质量信号。" },
  control: { title: "Control", description: "观察 worker、shard、outbox、DLQ 和本地栈状态。" },
};

export const defaultFilters: FilterState = {
  platform: "",
  accountId: "",
  dateFrom: "",
  dateTo: "",
  entityLevel: "",
  country: "",
  os: "",
  device: "",
  network: "",
  campaignId: "",
  adGroupId: "",
  adId: "",
  detailLimit: "200",
  matchType: "",
  searchTerm: "",
  searchTermLimit: "100",
};

const queryKeys: Array<[keyof FilterState, string]> = [
  ["platform", "platform"],
  ["accountId", "account_id"],
  ["dateFrom", "date_from"],
  ["dateTo", "date_to"],
  ["entityLevel", "entity_level"],
  ["country", "country"],
  ["os", "os"],
  ["device", "device"],
  ["network", "network"],
  ["campaignId", "platform_campaign_id"],
  ["adGroupId", "platform_ad_group_id"],
  ["adId", "platform_ad_id"],
  ["detailLimit", "detail_limit"],
  ["matchType", "match_type"],
  ["searchTerm", "search_term"],
  ["searchTermLimit", "search_term_limit"],
];

export function readPageFromPath(pathname: string): PageKey {
  const raw = pathname.replace(/^\/bi\/?/, "").split("/")[0];
  if (raw === "breakdown" || raw === "creatives" || raw === "quality" || raw === "control") {
    return raw;
  }
  return "overview";
}

export function readFiltersFromSearch(search: string): FilterState {
  const params = new URLSearchParams(search);
  const filters = { ...defaultFilters };
  for (const [field, queryKey] of queryKeys) {
    const value = params.get(queryKey);
    if (value !== null) {
      filters[field] = value;
    }
  }
  return filters;
}

export function normalizeFiltersForPage(page: PageKey, filters: FilterState): FilterState {
  if (page !== "overview") {
    return filters;
  }
  return {
    ...filters,
    accountId: "",
    entityLevel: "",
    network: "",
    detailLimit: defaultFilters.detailLimit,
    matchType: "",
    searchTerm: "",
    searchTermLimit: defaultFilters.searchTermLimit,
  };
}

export function filtersToSearch(filters: FilterState): string {
  const params = new URLSearchParams();
  for (const [field, queryKey] of queryKeys) {
    const value = filters[field].trim();
    if (value !== "" && value !== defaultFilters[field]) {
      params.set(queryKey, value);
    }
  }
  return params.toString();
}

export function pathForPage(page: PageKey, filters: FilterState) {
  const search = filtersToSearch(normalizeFiltersForPage(page, filters));
  const path = `/bi/${page}`;
  return search ? `${path}?${search}` : path;
}
