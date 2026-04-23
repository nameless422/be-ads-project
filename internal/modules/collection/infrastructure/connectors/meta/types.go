package meta

import (
	"time"

	"be_ads_project/internal/shared/ads"
)

type AccountContext struct {
	Account    domain.PlatformAccount
	Credential domain.AccountCredential
}

type Checkpoint struct {
	TimeWatermark  *time.Time
	PageCursor     string
	LookbackWindow time.Duration
}

type FetchRequest struct {
	AccountContext AccountContext
	ObjectType     domain.ObjectType
	StartTime      *time.Time
	EndTime        *time.Time
	Fields         []string
	PageSize       int
	Checkpoint     Checkpoint
}

type FetchResult struct {
	RawRecords        []domain.RawRecord
	NextPageCursor    string
	NextTimeWatermark *time.Time
	HasMore           bool
}
