package domain

import (
	"context"
	"time"

	rootdomain "be_ads_project/internal/shared/ads"
	messagingdomain "be_ads_project/internal/shared/messaging/domain"
)

type RawStore interface {
	SaveBatch(context.Context, SaveBatchInput) ([]StoredRawRecord, error)
	GetByIDs(context.Context, []int64) ([]rootdomain.RawRecord, error)
	ListPendingOutbox(context.Context, int) ([]OutboxEvent, error)
	MarkOutboxPublished(context.Context, int64, time.Time) error
	MarkOutboxFailed(context.Context, int64, string, time.Time) error
}

type SaveBatchInput struct {
	JobID             string
	TraceID           string
	ProfileID         string
	Platform          rootdomain.Platform
	PlatformAccountID string
	AccountID         string
	ObjectType        rootdomain.ObjectType
	SourceMode        string
	SourceCursor      string
	SourceWatermark   string
	CollectedAt       string
	Records           []rootdomain.RawRecord
	OutboxEvents      []OutboxEventInput
}

type OutboxEventInput struct {
	EventID       string
	AggregateType string
	AggregateID   string
	EventType     string
	Topic         string
	Payload       []byte
	RawEvent      *messagingdomain.RawEvent
	AvailableAt   time.Time
}
