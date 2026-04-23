package collector

import (
	"be_ads_project/internal/modules/collection/infrastructure/connectors/facebook"
	"be_ads_project/internal/modules/collection/infrastructure/connectors/googleads"
	"be_ads_project/internal/modules/collection/infrastructure/connectors/meta"
	"be_ads_project/internal/modules/collection/infrastructure/connectors/tiktok"
)

func NewRegistry() *meta.Registry {
	return meta.NewRegistry(
		facebook.NewConnector(),
		googleads.NewConnector(),
		tiktok.NewConnector(),
	)
}
