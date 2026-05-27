package normalizer

import (
	"context"
	"fmt"
	"time"

	transformationdomain "be_ads_project/internal/modules/transformation/domain"
	ads "be_ads_project/internal/shared/ads"
)

type Router struct {
	router *PlatformRouter
}

func NewRouter() *Router {
	return &Router{
		router: NewPlatformRouter(
			NewFacebookNormalizer(),
			NewGoogleAdsNormalizer(),
			NewTikTokNormalizer(),
		),
	}
}

func (r *Router) Normalize(ctx context.Context, input *transformationdomain.NormalizedBatchInput) (*transformationdomain.NormalizedBatch, error) {
	if input == nil || input.CollectedBatch == nil {
		return nil, fmt.Errorf("collected batch is required")
	}

	collected := input.CollectedBatch
	payload, err := r.router.Normalize(ctx, collected.Target.Profile.Platform, collected.Records)
	if err != nil {
		return nil, fmt.Errorf("normalize records: %w", err)
	}
	if payload == nil {
		payload = &ads.NormalizedPayload{}
	}

	return &transformationdomain.NormalizedBatch{
		Collected:    collected,
		Payload:      payload,
		NormalizedAt: time.Now().UTC(),
	}, nil
}
