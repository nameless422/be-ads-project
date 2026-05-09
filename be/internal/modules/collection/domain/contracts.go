package domain

import "context"

type SyncTargetProvider interface {
	ListTargets(context.Context) ([]SyncTarget, error)
}

type SyncTargetResolver interface {
	ResolveTarget(context.Context, string) (*SyncTarget, error)
}

type BatchCollector interface {
	Collect(context.Context, SyncTarget) (*CollectedBatch, error)
}
