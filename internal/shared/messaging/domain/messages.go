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
	PlatformAccountID string                `json:"platform_account_id"`
	AccountID         string                `json:"account_id"`
	ObjectType        rootdomain.ObjectType `json:"object_type"`
	RawRecordIDs      []int64               `json:"raw_record_ids"`
	CollectedAt       time.Time             `json:"collected_at"`
}
