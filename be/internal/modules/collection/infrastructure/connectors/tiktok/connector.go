package tiktok

import (
	"context"

	"be_ads_project/internal/mock"
	"be_ads_project/internal/modules/collection/infrastructure/connectors/meta"
	"be_ads_project/internal/shared/ads"
)

type Connector struct{}

func NewConnector() *Connector {
	return &Connector{}
}

func (c *Connector) Platform() domain.Platform {
	return domain.PlatformTikTokAds
}

func (c *Connector) Validate(context.Context, meta.AccountContext) error {
	return nil
}

func (c *Connector) FetchAccounts(_ context.Context, req meta.FetchRequest) (*meta.FetchResult, error) {
	return fetchMock(req)
}

func (c *Connector) FetchCampaigns(_ context.Context, req meta.FetchRequest) (*meta.FetchResult, error) {
	return fetchMock(req)
}

func (c *Connector) FetchAdGroups(_ context.Context, req meta.FetchRequest) (*meta.FetchResult, error) {
	return fetchMock(req)
}

func (c *Connector) FetchAds(_ context.Context, req meta.FetchRequest) (*meta.FetchResult, error) {
	return fetchMock(req)
}

func (c *Connector) FetchInsights(_ context.Context, req meta.FetchRequest) (*meta.FetchResult, error) {
	return fetchMock(req)
}

func (c *Connector) FetchSearchTerms(_ context.Context, req meta.FetchRequest) (*meta.FetchResult, error) {
	return fetchMock(req)
}

func fetchMock(req meta.FetchRequest) (*meta.FetchResult, error) {
	records, err := mock.RawRecords(req.AccountContext.Account, req.ObjectType)
	if err != nil {
		return nil, err
	}
	return &meta.FetchResult{
		RawRecords:        records,
		NextPageCursor:    "",
		NextTimeWatermark: req.EndTime,
		HasMore:           false,
	}, nil
}
