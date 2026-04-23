package domain

import (
	"context"

	ingestiondomain "be_ads_project/internal/modules/collection/domain"
)

type BatchNormalizer interface {
	Normalize(context.Context, *NormalizedBatchInput) (*NormalizedBatch, error)
}

type BatchProjector interface {
	Project(context.Context, *NormalizedBatch) error
}

type NormalizedBatchInput struct {
	CollectedBatch *ingestiondomain.CollectedBatch
}
