package provider

import (
	"context"
	"fmt"

	"be_ads_project/internal/mock"
	ingestiondomain "be_ads_project/internal/modules/collection/domain"
	rootdomain "be_ads_project/internal/shared/ads"
)

type StaticProvider struct{}

func NewStaticProvider() *StaticProvider {
	return &StaticProvider{}
}

var _ ingestiondomain.SyncTargetProvider = (*StaticProvider)(nil)
var _ ingestiondomain.SyncTargetResolver = (*StaticProvider)(nil)

func (p *StaticProvider) ListTargets(_ context.Context) ([]ingestiondomain.SyncTarget, error) {
	accounts := mock.Accounts()
	profiles := mock.Profiles(accounts)
	credentials := mock.Credentials(accounts)

	accountMap := make(map[string]rootdomain.PlatformAccount, len(accounts))
	for _, account := range accounts {
		accountMap[account.ID] = account
	}

	targets := make([]ingestiondomain.SyncTarget, 0, len(profiles))
	for _, profile := range profiles {
		account, ok := accountMap[profile.PlatformAccountID]
		if !ok {
			return nil, fmt.Errorf("missing account for profile platform_account_id=%s", profile.PlatformAccountID)
		}

		targets = append(targets, ingestiondomain.SyncTarget{
			Profile: profile,
			Bundle: ingestiondomain.AccountBundle{
				Account:    account,
				Credential: credentials[profile.PlatformAccountID],
			},
		})
	}

	return targets, nil
}

func (p *StaticProvider) ResolveTarget(ctx context.Context, profileID string) (*ingestiondomain.SyncTarget, error) {
	targets, err := p.ListTargets(ctx)
	if err != nil {
		return nil, err
	}
	for _, target := range targets {
		if target.Profile.ID == profileID {
			item := target
			return &item, nil
		}
	}
	return nil, fmt.Errorf("sync target not found profile_id=%s", profileID)
}
