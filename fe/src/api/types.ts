export type PageKey = "overview" | "breakdown" | "creatives" | "quality" | "control";

export type ApiList<T> = {
  items: T[];
};

export type FilterState = {
  platform: string;
  accountId: string;
  dateFrom: string;
  dateTo: string;
  entityLevel: string;
  country: string;
  os: string;
  device: string;
  network: string;
  campaignId: string;
  adGroupId: string;
  adId: string;
  detailLimit: string;
  matchType: string;
  searchTerm: string;
  searchTermLimit: string;
};

export type AccountSnapshot = {
  Platform: string;
  PlatformAccountID: string;
  AccountID: string;
  AccountName: string;
  LastSourceMode: string;
  LastObjectType: string;
  LastCollectedAt: string;
  CampaignCount: number;
  AdGroupCount: number;
  AdCount: number;
  InsightCount: number;
};

export type CampaignView = {
  Platform: string;
  PlatformAccountID: string;
  AccountID: string;
  PlatformCampaignID: string;
  CampaignName: string;
  Status: string;
  Objective: string;
  BuyingType: string;
  BiddingStrategy: string;
  DailyBudget: string;
  LifetimeBudget: string;
  Currency: string;
  StartTime: string;
  EndTime: string;
  SourceUpdatedAt: string;
  IngestedAt: string;
};

export type InsightSummaryRow = {
  Platform: string;
  PlatformAccountID: string;
  StatDate: string;
  Impressions: number;
  Clicks: number;
  Spend: string;
  Conversions: string;
  AllConversions: string;
  ConversionsValue: string;
  CostPerConversion: string;
  CostPerAllConversions: string;
  Reach: number;
};

export type InsightDetailRow = {
  Platform: string;
  PlatformAccountID: string;
  PlatformCampaignID: string;
  EntityLevel: string;
  EntityID: string;
  PlatformAdGroupID: string;
  PlatformAdID: string;
  StatDate: string;
  Device: string;
  Network: string;
  Impressions: number;
  Clicks: number;
  Spend: string;
  CTR: string;
  CPC: string;
  CPM: string;
  Conversions: string;
  AllConversions: string;
  ConversionsValue: string;
  CostPerConversion: string;
  CostPerAllConversions: string;
  Reach: number;
};

export type CampaignDiagnosticRow = {
  Platform: string;
  PlatformAccountID: string;
  PlatformCampaignID: string;
  StatDate: string;
  SearchImpressionShare: string;
  SearchTopImpressionShare: string;
  SearchAbsoluteTopImpressionShare: string;
};

export type SearchTermRow = {
  Platform: string;
  PlatformAccountID: string;
  PlatformCampaignID: string;
  PlatformAdGroupID: string;
  SearchTerm: string;
  SearchTermMatchType: string;
  StatDate: string;
  Impressions: number;
  Clicks: number;
  Spend: string;
  Conversions: string;
  ConversionsValue: string;
};

export type UAReportRow = {
  platform: string;
  platform_account_id: string;
  platform_campaign_id?: string;
  platform_ad_group_id?: string;
  platform_ad_id?: string;
  entity_level: string;
  entity_id: string;
  stat_date: string;
  country?: string;
  os?: string;
  placement?: string;
  creative_id?: string;
  creative_type?: string;
  optimization_goal?: string;
  bid_type?: string;
  targeting?: string;
  device?: string;
  network?: string;
  impressions: number;
  clicks: number;
  ctr: string;
  cpm: string;
  cpc: string;
  spend: string;
  reach: number;
  frequency: string;
  conversions: string;
  all_conversions: string;
  conversions_value: string;
  cost_per_conversion: string;
  cost_per_all_conversions: string;
  roas: string;
  installs: number;
  cpi: string;
  activations: number;
  activation_rate: string;
  registrations: number;
  cpr: string;
  registration_rate: string;
  tutorial_completions: number;
  role_creations: number;
  level_x_users: number;
  purchasers: number;
  payer_rate: string;
  purchase_count: number;
  first_purchase_amount: string;
  revenue_d1: string;
  revenue_d7: string;
  revenue_d30: string;
  ad_revenue: string;
  total_revenue: string;
  arpu: string;
  arppu: string;
  roi: string;
  retention_d1: string;
  retention_d3: string;
  retention_d7: string;
  retention_d30: string;
  ltv_d7: string;
  ltv_d30: string;
  ltv_to_cpi_ratio: string;
  avg_online_duration_seconds: number;
  task_completion_rate: string;
  high_value_payer_ratio: string;
};

export type GameKPIRecord = {
  platform: string;
  platform_account_id: string;
  platform_campaign_id?: string;
  platform_ad_group_id?: string;
  platform_ad_id?: string;
  stat_date: string;
  country?: string;
  os?: string;
  placement?: string;
  creative_id?: string;
  creative_type?: string;
  optimization_goal?: string;
  bid_type?: string;
  targeting?: string;
  installs: number;
  activations: number;
  registrations: number;
  tutorial_completions: number;
  role_creations: number;
  level_x_users: number;
  purchasers: number;
  purchase_count: number;
  first_purchase_amount: string;
  revenue_d1: string;
  revenue_d7: string;
  revenue_d30: string;
  ad_revenue: string;
  total_revenue: string;
  retention_d1: string;
  retention_d3: string;
  retention_d7: string;
  retention_d30: string;
  ltv_d7: string;
  ltv_d30: string;
  avg_online_duration_seconds: number;
  task_completion_rate: string;
  high_value_payer_ratio: string;
};

export type UAFieldDefinition = {
  key: string;
  label: string;
  category: string;
  status: "available" | "integration_ready" | "planned" | string;
  source: string;
  notes?: string;
  example_api?: string;
  related_keys?: string[];
};

export type WorkerLeaseView = {
  WorkerRole: string;
  WorkerID: string;
  PlatformScope: string;
  Capacity: number;
  LastSeenAt: string;
  ExpiresAt: string;
};

export type ShardAssignmentView = {
  WorkerRole: string;
  Platform: string;
  ShardID: number;
  WorkerID: string;
  UpdatedAt: string;
};

export type DeadLetterView = {
  StreamSequence: number;
  Subject: string;
  ID: string;
  Kind: string;
  Platform: string;
  ErrorMessage: string;
  DeliveryCount: number;
  FailedAt: string;
  OriginalSubject: string;
};

export type LocalProcessState = {
  name: string;
  state: string;
  detail?: string;
};

export type LocalWorkerGroupState = {
  role: string;
  running_count: number;
  total_count: number;
  instances: LocalProcessState[];
};

export type LocalPortState = {
  name: string;
  port: number;
  state: string;
  detail?: string;
};

export type LocalLogState = {
  name: string;
  state: string;
  lines: string[];
};

export type LocalStackStatus = {
  enabled: boolean;
  updated_at: string;
  services: LocalProcessState[];
  workers: LocalWorkerGroupState[];
  infra: LocalProcessState[];
  ports: LocalPortState[];
  logs: LocalLogState[];
  output: string;
};

export type ControlOverview = {
  GeneratedAt: string;
  WorkerLeases: WorkerLeaseView[];
  ShardAssignments: ShardAssignmentView[];
  RawRecordCount: number;
  OutboxPending: number;
  OutboxPublished: number;
  DeadLetterCount: number;
  Snapshots: AccountSnapshot[];
  LocalStack?: LocalStackStatus | null;
};

export type LocalCommandResult = {
  action: string;
  success: boolean;
  error?: string;
  output: string;
  started_at: string;
  finished_at: string;
};

export type BIDashboardData = {
  snapshots: AccountSnapshot[];
  campaigns: CampaignView[];
  insights: InsightSummaryRow[];
  insightDetails: InsightDetailRow[];
  campaignDiagnostics: CampaignDiagnosticRow[];
  searchTerms: SearchTermRow[];
  uaReports: UAReportRow[];
  gameKPIs: GameKPIRecord[];
  uaFields: UAFieldDefinition[];
  controlOverview: ControlOverview | null;
};
