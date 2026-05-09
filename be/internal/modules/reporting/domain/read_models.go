package domain

import (
	"encoding/json"
	"time"

	rootdomain "be_ads_project/internal/shared/ads"
)

type AccountSnapshot struct {
	Platform          rootdomain.Platform
	PlatformAccountID string
	AccountID         string
	AccountName       string
	LastSourceMode    string
	LastObjectType    rootdomain.ObjectType
	LastCollectedAt   time.Time
	CampaignCount     int
	AdGroupCount      int
	AdCount           int
	InsightCount      int
}

type CampaignFilter struct {
	Platform  string
	AccountID string
}

type CampaignView struct {
	Platform           rootdomain.Platform
	PlatformAccountID  string
	AccountID          string
	PlatformCampaignID string
	CampaignName       string
	Status             string
	Objective          string
	BuyingType         string
	BiddingStrategy    string
	DailyBudget        string
	LifetimeBudget     string
	Currency           string
	StartTime          time.Time
	EndTime            time.Time
	SourceUpdatedAt    time.Time
	IngestedAt         time.Time
}

type InsightSummaryFilter struct {
	Platform  string
	AccountID string
	DateFrom  time.Time
	DateTo    time.Time
}

type InsightSummaryRow struct {
	Platform              rootdomain.Platform
	PlatformAccountID     string
	StatDate              time.Time
	Impressions           int64
	Clicks                int64
	Spend                 string
	Conversions           string
	AllConversions        string
	ConversionsValue      string
	CostPerConversion     string
	CostPerAllConversions string
	Reach                 int64
}

type InsightDetailFilter struct {
	Platform    string
	AccountID   string
	DateFrom    time.Time
	DateTo      time.Time
	EntityLevel string
	Device      string
	Network     string
	Limit       int
}

type InsightDetailRow struct {
	Platform              rootdomain.Platform
	PlatformAccountID     string
	PlatformCampaignID    string
	EntityLevel           rootdomain.ObjectType
	EntityID              string
	PlatformAdGroupID     string
	PlatformAdID          string
	StatDate              time.Time
	Device                string
	Network               string
	Impressions           int64
	Clicks                int64
	Spend                 string
	CTR                   string
	CPC                   string
	CPM                   string
	Conversions           string
	AllConversions        string
	ConversionsValue      string
	CostPerConversion     string
	CostPerAllConversions string
	Reach                 int64
}

type UAReportFilter struct {
	Platform           string
	AccountID          string
	DateFrom           time.Time
	DateTo             time.Time
	EntityLevel        string
	Device             string
	Network            string
	Country            string
	OS                 string
	PlatformCampaignID string
	PlatformAdGroupID  string
	PlatformAdID       string
	Limit              int
}

type UAAdReportRow struct {
	Platform              rootdomain.Platform
	PlatformAccountID     string
	PlatformCampaignID    string
	EntityLevel           rootdomain.ObjectType
	EntityID              string
	PlatformAdGroupID     string
	PlatformAdID          string
	StatDate              time.Time
	Device                string
	Network               string
	Impressions           int64
	Clicks                int64
	Spend                 string
	CTR                   string
	CPC                   string
	CPM                   string
	Conversions           string
	AllConversions        string
	ConversionsValue      string
	CostPerConversion     string
	CostPerAllConversions string
	Reach                 int64
	Frequency             string
	ROAS                  string
}

type GameKPIQueryFilter struct {
	Platform           string
	AccountID          string
	DateFrom           time.Time
	DateTo             time.Time
	Country            string
	OS                 string
	PlatformCampaignID string
	PlatformAdGroupID  string
	PlatformAdID       string
	Limit              int
}

type GameKPIRecord struct {
	Platform                 rootdomain.Platform `json:"platform"`
	PlatformAccountID        string              `json:"platform_account_id"`
	PlatformCampaignID       string              `json:"platform_campaign_id,omitempty"`
	PlatformAdGroupID        string              `json:"platform_ad_group_id,omitempty"`
	PlatformAdID             string              `json:"platform_ad_id,omitempty"`
	StatDate                 time.Time           `json:"stat_date"`
	Country                  string              `json:"country,omitempty"`
	OS                       string              `json:"os,omitempty"`
	Placement                string              `json:"placement,omitempty"`
	CreativeID               string              `json:"creative_id,omitempty"`
	CreativeType             string              `json:"creative_type,omitempty"`
	OptimizationGoal         string              `json:"optimization_goal,omitempty"`
	BidType                  string              `json:"bid_type,omitempty"`
	Targeting                string              `json:"targeting,omitempty"`
	Installs                 int64               `json:"installs"`
	Activations              int64               `json:"activations"`
	Registrations            int64               `json:"registrations"`
	TutorialCompletions      int64               `json:"tutorial_completions"`
	RoleCreations            int64               `json:"role_creations"`
	LevelXUsers              int64               `json:"level_x_users"`
	Purchasers               int64               `json:"purchasers"`
	PurchaseCount            int64               `json:"purchase_count"`
	FirstPurchaseAmount      string              `json:"first_purchase_amount"`
	RevenueD1                string              `json:"revenue_d1"`
	RevenueD7                string              `json:"revenue_d7"`
	RevenueD30               string              `json:"revenue_d30"`
	AdRevenue                string              `json:"ad_revenue"`
	TotalRevenue             string              `json:"total_revenue"`
	RetentionD1              string              `json:"retention_d1"`
	RetentionD3              string              `json:"retention_d3"`
	RetentionD7              string              `json:"retention_d7"`
	RetentionD30             string              `json:"retention_d30"`
	LTVD7                    string              `json:"ltv_d7"`
	LTVD30                   string              `json:"ltv_d30"`
	AvgOnlineDurationSeconds int64               `json:"avg_online_duration_seconds"`
	TaskCompletionRate       string              `json:"task_completion_rate"`
	HighValuePayerRatio      string              `json:"high_value_payer_ratio"`
	RawPayload               json.RawMessage     `json:"raw_payload,omitempty"`
}

type GameKPIUpsertRequest struct {
	Items []GameKPIRecord `json:"items"`
}

type UAReportRow struct {
	Platform                 rootdomain.Platform   `json:"platform"`
	PlatformAccountID        string                `json:"platform_account_id"`
	PlatformCampaignID       string                `json:"platform_campaign_id,omitempty"`
	PlatformAdGroupID        string                `json:"platform_ad_group_id,omitempty"`
	PlatformAdID             string                `json:"platform_ad_id,omitempty"`
	EntityLevel              rootdomain.ObjectType `json:"entity_level"`
	EntityID                 string                `json:"entity_id"`
	StatDate                 time.Time             `json:"stat_date"`
	Country                  string                `json:"country,omitempty"`
	OS                       string                `json:"os,omitempty"`
	Placement                string                `json:"placement,omitempty"`
	CreativeID               string                `json:"creative_id,omitempty"`
	CreativeType             string                `json:"creative_type,omitempty"`
	OptimizationGoal         string                `json:"optimization_goal,omitempty"`
	BidType                  string                `json:"bid_type,omitempty"`
	Targeting                string                `json:"targeting,omitempty"`
	Device                   string                `json:"device,omitempty"`
	Network                  string                `json:"network,omitempty"`
	Impressions              int64                 `json:"impressions"`
	Clicks                   int64                 `json:"clicks"`
	CTR                      string                `json:"ctr"`
	CPM                      string                `json:"cpm"`
	CPC                      string                `json:"cpc"`
	Spend                    string                `json:"spend"`
	Reach                    int64                 `json:"reach"`
	Frequency                string                `json:"frequency"`
	Conversions              string                `json:"conversions"`
	AllConversions           string                `json:"all_conversions"`
	ConversionsValue         string                `json:"conversions_value"`
	CostPerConversion        string                `json:"cost_per_conversion"`
	CostPerAllConversions    string                `json:"cost_per_all_conversions"`
	ROAS                     string                `json:"roas"`
	Installs                 int64                 `json:"installs"`
	CPI                      string                `json:"cpi"`
	Activations              int64                 `json:"activations"`
	ActivationRate           string                `json:"activation_rate"`
	Registrations            int64                 `json:"registrations"`
	CPR                      string                `json:"cpr"`
	RegistrationRate         string                `json:"registration_rate"`
	TutorialCompletions      int64                 `json:"tutorial_completions"`
	RoleCreations            int64                 `json:"role_creations"`
	LevelXUsers              int64                 `json:"level_x_users"`
	Purchasers               int64                 `json:"purchasers"`
	PayerRate                string                `json:"payer_rate"`
	PurchaseCount            int64                 `json:"purchase_count"`
	FirstPurchaseAmount      string                `json:"first_purchase_amount"`
	RevenueD1                string                `json:"revenue_d1"`
	RevenueD7                string                `json:"revenue_d7"`
	RevenueD30               string                `json:"revenue_d30"`
	AdRevenue                string                `json:"ad_revenue"`
	TotalRevenue             string                `json:"total_revenue"`
	ARPU                     string                `json:"arpu"`
	ARPPU                    string                `json:"arppu"`
	ROI                      string                `json:"roi"`
	RetentionD1              string                `json:"retention_d1"`
	RetentionD3              string                `json:"retention_d3"`
	RetentionD7              string                `json:"retention_d7"`
	RetentionD30             string                `json:"retention_d30"`
	LTVD7                    string                `json:"ltv_d7"`
	LTVD30                   string                `json:"ltv_d30"`
	LTVToCPIRatio            string                `json:"ltv_to_cpi_ratio"`
	AvgOnlineDurationSeconds int64                 `json:"avg_online_duration_seconds"`
	TaskCompletionRate       string                `json:"task_completion_rate"`
	HighValuePayerRatio      string                `json:"high_value_payer_ratio"`
}

type UAFieldDefinition struct {
	Key         string   `json:"key"`
	Label       string   `json:"label"`
	Category    string   `json:"category"`
	Status      string   `json:"status"`
	Source      string   `json:"source"`
	Notes       string   `json:"notes,omitempty"`
	ExampleAPI  string   `json:"example_api,omitempty"`
	RelatedKeys []string `json:"related_keys,omitempty"`
}

type CampaignDiagnosticFilter struct {
	Platform  string
	AccountID string
	DateFrom  time.Time
	DateTo    time.Time
	Limit     int
}

type CampaignDiagnosticRow struct {
	Platform                         rootdomain.Platform
	PlatformAccountID                string
	PlatformCampaignID               string
	StatDate                         time.Time
	SearchImpressionShare            string
	SearchTopImpressionShare         string
	SearchAbsoluteTopImpressionShare string
}

type SearchTermFilter struct {
	Platform        string
	AccountID       string
	DateFrom        time.Time
	DateTo          time.Time
	MatchType       string
	SearchTermQuery string
	Limit           int
}

type SearchTermRow struct {
	Platform            rootdomain.Platform
	PlatformAccountID   string
	PlatformCampaignID  string
	PlatformAdGroupID   string
	SearchTerm          string
	SearchTermMatchType string
	StatDate            time.Time
	Impressions         int64
	Clicks              int64
	Spend               string
	Conversions         string
	ConversionsValue    string
}

type WorkerLeaseView struct {
	WorkerRole    string
	WorkerID      string
	PlatformScope string
	Capacity      int
	LastSeenAt    time.Time
	ExpiresAt     time.Time
}

type ShardAssignmentView struct {
	WorkerRole string
	Platform   rootdomain.Platform
	ShardID    int
	WorkerID   string
	UpdatedAt  time.Time
}

type ControlPanelOverview struct {
	GeneratedAt      time.Time
	WorkerLeases     []WorkerLeaseView
	ShardAssignments []ShardAssignmentView
	RawRecordCount   int64
	OutboxPending    int64
	OutboxPublished  int64
	DeadLetterCount  uint64
	Snapshots        []AccountSnapshot
	LocalStack       *LocalStackStatus
}

type DeadLetterView struct {
	StreamSequence  uint64
	Subject         string
	ID              string
	Kind            string
	Platform        string
	ErrorMessage    string
	DeliveryCount   uint64
	FailedAt        time.Time
	OriginalSubject string
}

type BackfillRequest struct {
	Platform   string `json:"platform"`
	AccountID  string `json:"account_id"`
	ObjectType string `json:"object_type"`
	ProfileID  string `json:"profile_id"`
	Limit      int    `json:"limit"`
}

type ReplayRequest struct {
	Kind     string `json:"kind"`
	Platform string `json:"platform"`
	Limit    int    `json:"limit"`
}

type ActionResult struct {
	Accepted int      `json:"accepted"`
	Items    []string `json:"items"`
}

type LocalProcessState struct {
	Name   string `json:"name"`
	State  string `json:"state"`
	Detail string `json:"detail,omitempty"`
}

type LocalPortState struct {
	Name   string `json:"name"`
	Port   int    `json:"port"`
	State  string `json:"state"`
	Detail string `json:"detail,omitempty"`
}

type LocalLogState struct {
	Name  string   `json:"name"`
	State string   `json:"state"`
	Lines []string `json:"lines"`
}

type LocalWorkerGroupState struct {
	Role         string              `json:"role"`
	RunningCount int                 `json:"running_count"`
	TotalCount   int                 `json:"total_count"`
	Instances    []LocalProcessState `json:"instances"`
}

type LocalStackStatus struct {
	Enabled   bool                    `json:"enabled"`
	UpdatedAt time.Time               `json:"updated_at"`
	Services  []LocalProcessState     `json:"services"`
	Workers   []LocalWorkerGroupState `json:"workers"`
	Infra     []LocalProcessState     `json:"infra"`
	Ports     []LocalPortState        `json:"ports"`
	Logs      []LocalLogState         `json:"logs"`
	Output    string                  `json:"output"`
}

type LocalCommandResult struct {
	Action     string    `json:"action"`
	Success    bool      `json:"success"`
	Error      string    `json:"error,omitempty"`
	Output     string    `json:"output"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
}
