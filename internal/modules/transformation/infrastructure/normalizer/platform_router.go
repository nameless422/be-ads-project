package normalizer

import (
	"context"
	"fmt"

	ads "be_ads_project/internal/shared/ads"
)

type PlatformNormalizer interface {
	Platform() ads.Platform
	Normalize(context.Context, []ads.RawRecord) (*ads.NormalizedPayload, error)
}

type PlatformRouter struct {
	normalizers map[ads.Platform]PlatformNormalizer
}

func NewPlatformRouter(items ...PlatformNormalizer) *PlatformRouter {
	normalizers := make(map[ads.Platform]PlatformNormalizer, len(items))
	for _, item := range items {
		normalizers[item.Platform()] = item
	}
	return &PlatformRouter{normalizers: normalizers}
}

func (r *PlatformRouter) Normalize(ctx context.Context, platform ads.Platform, records []ads.RawRecord) (*ads.NormalizedPayload, error) {
	normalizer, ok := r.normalizers[platform]
	if !ok {
		return nil, fmt.Errorf("normalizer for platform %s not registered", platform)
	}
	return normalizer.Normalize(ctx, records)
}
