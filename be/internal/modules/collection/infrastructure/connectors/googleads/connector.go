package googleads

import (
	"context"
	"fmt"
	"strings"

	"be_ads_project/internal/mock"
	"be_ads_project/internal/modules/collection/infrastructure/connectors/meta"
	"be_ads_project/internal/shared/ads"
)

type Connector struct {
	realClient *realClient
}

func NewConnector() *Connector {
	return &Connector{
		realClient: newRealClient(),
	}
}

func (c *Connector) Platform() domain.Platform {
	return domain.PlatformGoogleAds
}

func (c *Connector) Validate(_ context.Context, accountCtx meta.AccountContext) error {
	mode := googleAdsMode(accountCtx.Credential)
	switch mode {
	case "mock", "seeded_test":
		return nil
	case "real":
		missing := make([]string, 0, 4)
		if strings.TrimSpace(accountCtx.Credential.ExtraConfig["developer_token"]) == "" {
			missing = append(missing, "developer_token")
		}
		if strings.TrimSpace(accountCtx.Credential.ClientID) == "" {
			missing = append(missing, "client_id")
		}
		if strings.TrimSpace(accountCtx.Credential.RefreshToken) == "" {
			missing = append(missing, "refresh_token")
		}
		if len(missing) > 0 {
			return fmt.Errorf("google ads real mode missing config: %s", strings.Join(missing, ", "))
		}
		return nil
	default:
		return fmt.Errorf("unsupported google ads mode %q", mode)
	}
}

func (c *Connector) FetchAccounts(ctx context.Context, req meta.FetchRequest) (*meta.FetchResult, error) {
	return c.fetch(ctx, req)
}

func (c *Connector) FetchCampaigns(ctx context.Context, req meta.FetchRequest) (*meta.FetchResult, error) {
	return c.fetch(ctx, req)
}

func (c *Connector) FetchAdGroups(ctx context.Context, req meta.FetchRequest) (*meta.FetchResult, error) {
	return c.fetch(ctx, req)
}

func (c *Connector) FetchAds(ctx context.Context, req meta.FetchRequest) (*meta.FetchResult, error) {
	return c.fetch(ctx, req)
}

func (c *Connector) FetchInsights(ctx context.Context, req meta.FetchRequest) (*meta.FetchResult, error) {
	return c.fetch(ctx, req)
}

func (c *Connector) FetchSearchTerms(ctx context.Context, req meta.FetchRequest) (*meta.FetchResult, error) {
	return c.fetch(ctx, req)
}

func (c *Connector) fetch(ctx context.Context, req meta.FetchRequest) (*meta.FetchResult, error) {
	switch googleAdsMode(req.AccountContext.Credential) {
	case "real":
		return c.realClient.fetch(ctx, req)
	case "mock", "seeded_test":
		return fetchMock(req)
	default:
		return nil, fmt.Errorf("unsupported google ads mode %q", googleAdsMode(req.AccountContext.Credential))
	}
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

func googleAdsMode(credential domain.AccountCredential) string {
	mode := strings.TrimSpace(credential.ExtraConfig["mode"])
	if mode == "" {
		return "mock"
	}
	return mode
}
