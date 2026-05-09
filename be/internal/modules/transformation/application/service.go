package application

import (
	"context"
	"encoding/json"
	"log"

	ingestiondomain "be_ads_project/internal/modules/collection/domain"
	transformationdomain "be_ads_project/internal/modules/transformation/domain"
)

type Service struct {
	normalizer transformationdomain.BatchNormalizer
	projector  transformationdomain.BatchProjector
	logger     *log.Logger
}

func NewService(
	normalizer transformationdomain.BatchNormalizer,
	projector transformationdomain.BatchProjector,
	logger *log.Logger,
) *Service {
	return &Service{
		normalizer: normalizer,
		projector:  projector,
		logger:     logger,
	}
}

func (s *Service) HandleCollectedBatch(ctx context.Context, batch *ingestiondomain.CollectedBatch) error {
	s.logRaw(batch)

	normalizedBatch, err := s.normalizer.Normalize(ctx, &transformationdomain.NormalizedBatchInput{
		CollectedBatch: batch,
	})
	if err != nil {
		return err
	}

	s.logNormalized(normalizedBatch)
	return s.projector.Project(ctx, normalizedBatch)
}

func (s *Service) logRaw(batch *ingestiondomain.CollectedBatch) {
	for _, record := range batch.Records {
		s.logger.Printf(
			"[transformation] raw_record platform=%s account=%s object=%s resource_id=%s payload=%s",
			record.Platform,
			record.PlatformAccountID,
			record.ObjectType,
			record.ResourceID,
			string(record.Payload),
		)
	}
}

func (s *Service) logNormalized(batch *transformationdomain.NormalizedBatch) {
	payload := batch.Payload
	profile := batch.Collected.Target.Profile
	s.logger.Printf(
		"[transformation] normalized profile=%s platform=%s object=%s raw=%d campaigns=%d adgroups=%d ads=%d insights=%d search_terms=%d has_more=%t next_cursor=%s",
		profile.ID,
		profile.Platform,
		profile.ObjectType,
		len(batch.Collected.Records),
		len(payload.Campaigns),
		len(payload.AdGroups),
		len(payload.Ads),
		len(payload.Insights),
		len(payload.SearchTerms),
		batch.Collected.HasMore,
		batch.Collected.NextCursor,
	)

	for _, item := range payload.Campaigns {
		s.logNormalizedItem(profile.ID, "campaign", item)
	}
	for _, item := range payload.AdGroups {
		s.logNormalizedItem(profile.ID, "ad_group", item)
	}
	for _, item := range payload.Ads {
		s.logNormalizedItem(profile.ID, "ad", item)
	}
	for _, item := range payload.Insights {
		s.logNormalizedItem(profile.ID, "insight", item)
	}
	for _, item := range payload.SearchTerms {
		s.logNormalizedItem(profile.ID, "search_term", item)
	}
}

func (s *Service) logNormalizedItem(profileID, objectType string, item any) {
	body, err := json.Marshal(item)
	if err != nil {
		s.logger.Printf("[transformation] normalized_record_encode_failed profile=%s object=%s err=%v", profileID, objectType, err)
		return
	}
	s.logger.Printf("[transformation] normalized_record profile=%s object=%s payload=%s", profileID, objectType, string(body))
}
