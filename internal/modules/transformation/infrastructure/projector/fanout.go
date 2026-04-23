package projector

import (
	"context"

	transformationdomain "be_ads_project/internal/modules/transformation/domain"
)

type Fanout struct {
	projectors []transformationdomain.BatchProjector
}

func NewFanout(projectors ...transformationdomain.BatchProjector) *Fanout {
	return &Fanout{projectors: projectors}
}

func (f *Fanout) Project(ctx context.Context, batch *transformationdomain.NormalizedBatch) error {
	for _, item := range f.projectors {
		if err := item.Project(ctx, batch); err != nil {
			return err
		}
	}
	return nil
}
