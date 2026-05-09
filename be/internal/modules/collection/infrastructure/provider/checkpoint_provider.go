package provider

import (
	"context"

	ingestiondomain "be_ads_project/internal/modules/collection/domain"
)

type CheckpointStore interface {
	LatestCheckpoint(context.Context, string) (*ingestiondomain.SyncCheckpoint, error)
}

type CheckpointProvider struct {
	base interface {
		ingestiondomain.SyncTargetProvider
		ingestiondomain.SyncTargetResolver
	}
	store CheckpointStore
}

func NewCheckpointProvider(
	base interface {
		ingestiondomain.SyncTargetProvider
		ingestiondomain.SyncTargetResolver
	},
	store CheckpointStore,
) *CheckpointProvider {
	return &CheckpointProvider{
		base:  base,
		store: store,
	}
}

var _ ingestiondomain.SyncTargetProvider = (*CheckpointProvider)(nil)
var _ ingestiondomain.SyncTargetResolver = (*CheckpointProvider)(nil)

func (p *CheckpointProvider) ListTargets(ctx context.Context) ([]ingestiondomain.SyncTarget, error) {
	targets, err := p.base.ListTargets(ctx)
	if err != nil {
		return nil, err
	}
	for i := range targets {
		if err := p.applyCheckpoint(ctx, &targets[i]); err != nil {
			return nil, err
		}
	}
	return targets, nil
}

func (p *CheckpointProvider) ResolveTarget(ctx context.Context, profileID string) (*ingestiondomain.SyncTarget, error) {
	target, err := p.base.ResolveTarget(ctx, profileID)
	if err != nil {
		return nil, err
	}
	if err := p.applyCheckpoint(ctx, target); err != nil {
		return nil, err
	}
	return target, nil
}

func (p *CheckpointProvider) applyCheckpoint(ctx context.Context, target *ingestiondomain.SyncTarget) error {
	if p.store == nil || target == nil {
		return nil
	}
	checkpoint, err := p.store.LatestCheckpoint(ctx, target.Profile.ID)
	if err != nil {
		return err
	}
	if checkpoint == nil {
		return nil
	}
	target.Profile.PageToken = checkpoint.NextCursor
	target.Profile.WatermarkValue = checkpoint.NextWatermark
	return nil
}
