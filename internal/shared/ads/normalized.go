package domain

import "time"

type StandardAccount struct {
	Platform          Platform
	PlatformAccountID string
	AccountID         string
	AccountName       string
	Status            string
	Timezone          string
	Currency          string
	RawPayload        []byte
}

type StandardCampaign struct {
	Platform           Platform
	PlatformAccountID  string
	PlatformCampaignID string
	CampaignName       string
	Status             string
	Objective          string
	BuyingType         string
	BiddingStrategy    string
	DailyBudget        string
	LifetimeBudget     string
	Currency           string
	StartTime          *time.Time
	EndTime            *time.Time
	UpdatedAt          *time.Time
	RawPayload         []byte
}

type StandardCampaignDiagnostic struct {
	Platform                         Platform
	PlatformAccountID                string
	PlatformCampaignID               string
	StatDate                         time.Time
	SearchImpressionShare            string
	SearchTopImpressionShare         string
	SearchAbsoluteTopImpressionShare string
	RawPayload                       []byte
}

type StandardAdGroup struct {
	Platform          Platform
	PlatformAccountID string
	PlatformAdGroupID string
	PlatformParentID  string
	AdGroupName       string
	Status            string
	BidStrategy       string
	DailyBudget       string
	StartTime         *time.Time
	EndTime           *time.Time
	UpdatedAt         *time.Time
	RawPayload        []byte
}

type StandardAd struct {
	Platform          Platform
	PlatformAccountID string
	PlatformAdID      string
	PlatformParentID  string
	AdName            string
	Status            string
	CreativeID        string
	CreativeName      string
	UpdatedAt         *time.Time
	RawPayload        []byte
}

type StandardInsight struct {
	Platform              Platform
	PlatformAccountID     string
	EntityLevel           ObjectType
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
	RawPayload            []byte
}

type StandardSearchTerm struct {
	Platform            Platform
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
	RawPayload          []byte
}

type NormalizedPayload struct {
	Accounts            []StandardAccount
	Campaigns           []StandardCampaign
	CampaignDiagnostics []StandardCampaignDiagnostic
	AdGroups            []StandardAdGroup
	Ads                 []StandardAd
	Insights            []StandardInsight
	SearchTerms         []StandardSearchTerm
}
