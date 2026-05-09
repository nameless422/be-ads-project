import type {
  ApiList,
  BIDashboardData,
  CampaignDiagnosticRow,
  CampaignView,
  ControlOverview,
  DeadLetterView,
  FilterState,
  GameKPIRecord,
  InsightDetailRow,
  InsightSummaryRow,
  LocalCommandResult,
  SearchTermRow,
  UAFieldDefinition,
  UAReportRow,
  AccountSnapshot,
} from "./types";

async function fetchJSON<T>(url: string, options?: RequestInit): Promise<T> {
  const response = await fetch(url, {
    headers: {
      Accept: "application/json",
      ...(options?.headers ?? {}),
    },
    ...options,
  });
  if (!response.ok) {
    const body = await response.text();
    throw new Error(`${response.status} ${response.statusText}${body ? `: ${body}` : ""}`);
  }
  return response.json() as Promise<T>;
}

function appendIfPresent(params: URLSearchParams, key: string, value: string | number | undefined) {
  if (value === undefined) {
    return;
  }
  const normalized = String(value).trim();
  if (normalized !== "") {
    params.set(key, normalized);
  }
}

function commonParams(filters: FilterState, accountKey: "account_id" | "platform_account_id") {
  const params = new URLSearchParams();
  appendIfPresent(params, "platform", filters.platform);
  appendIfPresent(params, accountKey, filters.accountId);
  appendIfPresent(params, "date_from", filters.dateFrom);
  appendIfPresent(params, "date_to", filters.dateTo);
  return params;
}

function detailParams(filters: FilterState) {
  const params = commonParams(filters, "platform_account_id");
  appendIfPresent(params, "entity_level", filters.entityLevel);
  appendIfPresent(params, "device", filters.device);
  appendIfPresent(params, "network", filters.network);
  appendIfPresent(params, "country", filters.country);
  appendIfPresent(params, "os", filters.os);
  appendIfPresent(params, "platform_campaign_id", filters.campaignId);
  appendIfPresent(params, "platform_ad_group_id", filters.adGroupId);
  appendIfPresent(params, "platform_ad_id", filters.adId);
  appendIfPresent(params, "limit", filters.detailLimit || "200");
  return params;
}

function searchTermParams(filters: FilterState) {
  const params = commonParams(filters, "platform_account_id");
  appendIfPresent(params, "match_type", filters.matchType);
  appendIfPresent(params, "search_term", filters.searchTerm);
  appendIfPresent(params, "limit", filters.searchTermLimit || "100");
  return params;
}

function withQuery(path: string, params: URLSearchParams) {
  const query = params.toString();
  return query ? `${path}?${query}` : path;
}

export async function loadBIDashboard(filters: FilterState): Promise<BIDashboardData> {
  const [
    snapshots,
    campaigns,
    insights,
    insightDetails,
    campaignDiagnostics,
    searchTerms,
    uaReports,
    gameKPIs,
    uaFields,
    controlOverview,
  ] = await Promise.all([
    fetchJSON<ApiList<AccountSnapshot>>("/api/bi/snapshots"),
    fetchJSON<ApiList<CampaignView>>(withQuery("/api/bi/campaigns", commonParams(filters, "account_id"))),
    fetchJSON<ApiList<InsightSummaryRow>>(withQuery("/api/bi/insights/summary", commonParams(filters, "platform_account_id"))),
    fetchJSON<ApiList<InsightDetailRow>>(withQuery("/api/bi/insights/detail", detailParams(filters))),
    fetchJSON<ApiList<CampaignDiagnosticRow>>(withQuery("/api/bi/campaign-diagnostics", detailParams(filters))),
    fetchJSON<ApiList<SearchTermRow>>(withQuery("/api/bi/search-terms", searchTermParams(filters))),
    fetchJSON<ApiList<UAReportRow>>(withQuery("/api/bi/ua-report", detailParams(filters))),
    fetchJSON<ApiList<GameKPIRecord>>(withQuery("/api/bi/game-kpis", detailParams(filters))),
    fetchJSON<ApiList<UAFieldDefinition>>("/api/bi/ua-fields"),
    fetchJSON<ControlOverview>("/api/control/overview").catch(() => null),
  ]);

  return {
    snapshots: snapshots.items ?? [],
    campaigns: campaigns.items ?? [],
    insights: insights.items ?? [],
    insightDetails: insightDetails.items ?? [],
    campaignDiagnostics: campaignDiagnostics.items ?? [],
    searchTerms: searchTerms.items ?? [],
    uaReports: uaReports.items ?? [],
    gameKPIs: gameKPIs.items ?? [],
    uaFields: uaFields.items ?? [],
    controlOverview,
  };
}

export async function runLocalStackAction(action: string, role?: string): Promise<LocalCommandResult> {
  const body = role ? JSON.stringify({ role }) : undefined;
  return fetchJSON<LocalCommandResult>(`/api/control/local-stack/${action}`, {
    method: "POST",
    headers: body ? { "Content-Type": "application/json" } : undefined,
    body,
  });
}

export async function loadDeadLetters(limit = 20): Promise<DeadLetterView[]> {
  const result = await fetchJSON<ApiList<DeadLetterView>>(`/api/control/dlq?limit=${limit}`);
  return result.items ?? [];
}
