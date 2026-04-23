package domain

import (
	"time"

	rootdomain "be_ads_project/internal/shared/ads"
)

type StoredRawRecord struct {
	ID                int64
	JobID             string
	TraceID           string
	ProfileID         string
	Platform          rootdomain.Platform
	PlatformAccountID string
	AccountID         string
	ObjectType        rootdomain.ObjectType
	ResourceID        string
	Payload           []byte
	SourceMode        string
	SourceCursor      string
	SourceWatermark   string
	CollectedAt       time.Time
}

type OutboxStatus string

const (
	OutboxStatusPending   OutboxStatus = "pending"
	OutboxStatusPublished OutboxStatus = "published"
	OutboxStatusFailed    OutboxStatus = "failed"
)

type OutboxEvent struct {
	ID            int64
	EventID       string
	AggregateType string
	AggregateID   string
	EventType     string
	Topic         string
	Payload       []byte
	Status        OutboxStatus
	AttemptCount  int
	LastError     string
	AvailableAt   time.Time
	PublishedAt   *time.Time
	CreatedAt     time.Time
}
