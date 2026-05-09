package domain

import (
	"time"

	rootdomain "be_ads_project/internal/shared/ads"
)

type CollectJob struct {
	JobID             string                `json:"job_id"`
	TraceID           string                `json:"trace_id"`
	ProfileID         string                `json:"profile_id"`
	Platform          rootdomain.Platform   `json:"platform"`
	ShardID           int                   `json:"shard_id"`
	PlatformAccountID string                `json:"platform_account_id"`
	AccountID         string                `json:"account_id"`
	ObjectType        rootdomain.ObjectType `json:"object_type"`
	SyncMode          rootdomain.SyncMode   `json:"sync_mode"`
	WatermarkValue    string                `json:"watermark_value"`
	PageToken         string                `json:"page_token"`
	DispatchedAt      time.Time             `json:"dispatched_at"`
}

type RawEvent struct {
	EventID           string                `json:"event_id"`
	JobID             string                `json:"job_id"`
	TraceID           string                `json:"trace_id"`
	ProfileID         string                `json:"profile_id"`
	Platform          rootdomain.Platform   `json:"platform"`
	ShardID           int                   `json:"shard_id"`
	PlatformAccountID string                `json:"platform_account_id"`
	AccountID         string                `json:"account_id"`
	ObjectType        rootdomain.ObjectType `json:"object_type"`
	RawRecordIDs      []int64               `json:"raw_record_ids"`
	CollectedAt       time.Time             `json:"collected_at"`
}

type DeadLetterKind string

const (
	DeadLetterKindCollectJob DeadLetterKind = "collect_job"
	DeadLetterKindRawEvent   DeadLetterKind = "raw_event"
)

type DeadLetterEvent struct {
	ID              string              `json:"id"`
	Kind            DeadLetterKind      `json:"kind"`
	Platform        rootdomain.Platform `json:"platform"`
	OriginalStream  string              `json:"original_stream"`
	OriginalSubject string              `json:"original_subject"`
	OriginalMessage []byte              `json:"original_message"`
	ErrorMessage    string              `json:"error_message"`
	DeliveryCount   uint64              `json:"delivery_count"`
	FailedAt        time.Time           `json:"failed_at"`
}
