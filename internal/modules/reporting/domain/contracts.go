package domain

import "context"

type SnapshotStore interface {
	UpsertAccountSnapshot(context.Context, AccountSnapshot) error
	GetAccountSnapshot(context.Context, string) (*AccountSnapshot, error)
	ListAccountSnapshots(context.Context) ([]AccountSnapshot, error)
}
