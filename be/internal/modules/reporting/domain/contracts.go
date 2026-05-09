package domain

import "context"

type SnapshotStore interface {
	UpsertAccountSnapshot(context.Context, AccountSnapshot) error
	GetAccountSnapshot(context.Context, string) (*AccountSnapshot, error)
	ListAccountSnapshots(context.Context) ([]AccountSnapshot, error)
}

type ControlPanelReader interface {
	ListWorkerLeases(context.Context) ([]WorkerLeaseView, error)
	ListShardAssignments(context.Context) ([]ShardAssignmentView, error)
	CountRawRecords(context.Context) (int64, error)
	CountOutboxByStatus(context.Context, string) (int64, error)
}
