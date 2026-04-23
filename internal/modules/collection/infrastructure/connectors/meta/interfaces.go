package meta

import (
	"context"
	"fmt"

	"be_ads_project/internal/shared/ads"
)

type Connector interface {
	Platform() domain.Platform
	Validate(context.Context, AccountContext) error
	FetchAccounts(context.Context, FetchRequest) (*FetchResult, error)
	FetchCampaigns(context.Context, FetchRequest) (*FetchResult, error)
	FetchAdGroups(context.Context, FetchRequest) (*FetchResult, error)
	FetchAds(context.Context, FetchRequest) (*FetchResult, error)
	FetchInsights(context.Context, FetchRequest) (*FetchResult, error)
	FetchSearchTerms(context.Context, FetchRequest) (*FetchResult, error)
}

type Registry struct {
	connectors map[domain.Platform]Connector
}

func NewRegistry(connectors ...Connector) *Registry {
	items := make(map[domain.Platform]Connector, len(connectors))
	for _, connector := range connectors {
		items[connector.Platform()] = connector
	}
	return &Registry{connectors: items}
}

func (r *Registry) Get(platform domain.Platform) (Connector, error) {
	connector, ok := r.connectors[platform]
	if !ok {
		return nil, fmt.Errorf("connector for platform %s not registered", platform)
	}
	return connector, nil
}
