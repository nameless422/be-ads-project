package domain

import (
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
