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
	DailyBudget        string
	LifetimeBudget     string
	Currency           string
	IngestedAt         time.Time
}

type InsightSummaryFilter struct {
	Platform  string
	AccountID string
	DateFrom  time.Time
	DateTo    time.Time
}

type InsightSummaryRow struct {
	Platform          rootdomain.Platform
	PlatformAccountID string
	StatDate          time.Time
	Impressions       int64
	Clicks            int64
	Spend             string
	Conversions       string
	Reach             int64
}
