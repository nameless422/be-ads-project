package application

import (
	"context"
	"log"

	biquerydomain "be_ads_project/internal/modules/reporting/domain"
	transformationdomain "be_ads_project/internal/modules/transformation/domain"
	rootdomain "be_ads_project/internal/shared/ads"
)

type Service struct {
	repository biquerydomain.SnapshotStore
	logger     *log.Logger
}

func NewService(repository biquerydomain.SnapshotStore, logger *log.Logger) *Service {
	return &Service{
		repository: repository,
		logger:     logger,
	}
}

func (s *Service) Project(ctx context.Context, batch *transformationdomain.NormalizedBatch) error {
	snapshot, err := s.mergeAccountSnapshot(ctx, batch)
	if err != nil {
		return err
	}
	if err := s.repository.UpsertAccountSnapshot(ctx, snapshot); err != nil {
		return err
	}

	s.logger.Printf(
		"[biquery] projected platform=%s account=%s object=%s source_mode=%s campaigns=%d adgroups=%d ads=%d insights=%d",
		snapshot.Platform,
		snapshot.AccountID,
		snapshot.LastObjectType,
		snapshot.LastSourceMode,
		snapshot.CampaignCount,
		snapshot.AdGroupCount,
		snapshot.AdCount,
		snapshot.InsightCount,
	)

	snapshots, err := s.repository.ListAccountSnapshots(ctx)
	if err != nil {
		return err
	}
	s.logger.Printf("[biquery] snapshot_accounts=%d", len(snapshots))
	return nil
}

func (s *Service) mergeAccountSnapshot(ctx context.Context, batch *transformationdomain.NormalizedBatch) (biquerydomain.AccountSnapshot, error) {
	account := batch.Collected.Target.Bundle.Account
	payload := batch.Payload

	current := biquerydomain.AccountSnapshot{
		Platform:          account.Platform,
		PlatformAccountID: account.ID,
		AccountID:         account.AccountID,
		AccountName:       account.AccountName,
		LastSourceMode:    batch.Collected.SourceMode,
		LastObjectType:    batch.Collected.Target.Profile.ObjectType,
		LastCollectedAt:   batch.Collected.CollectedAtUTC,
	}
	existing, err := s.repository.GetAccountSnapshot(ctx, account.ID)
	if err != nil {
		return biquerydomain.AccountSnapshot{}, err
	}
	if existing != nil {
		current.CampaignCount = existing.CampaignCount
		current.AdGroupCount = existing.AdGroupCount
		current.AdCount = existing.AdCount
		current.InsightCount = existing.InsightCount
	}

	switch batch.Collected.Target.Profile.ObjectType {
	case rootdomain.ObjectTypeCampaign:
		current.CampaignCount = len(payload.Campaigns)
	case rootdomain.ObjectTypeAdGroup:
		current.AdGroupCount = len(payload.AdGroups)
	case rootdomain.ObjectTypeAd:
		current.AdCount = len(payload.Ads)
	case rootdomain.ObjectTypeInsight:
		current.InsightCount = len(payload.Insights)
	}

	return current, nil
}
