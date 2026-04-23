package domain

import (
	"time"

	rootdomain "be_ads_project/internal/shared/ads"
)

type AccountBundle struct {
	Account    rootdomain.PlatformAccount
	Credential rootdomain.AccountCredential
}

type SyncTarget struct {
	Profile rootdomain.SyncProfile
	Bundle  AccountBundle
}

type CollectedBatch struct {
	Target         SyncTarget
	Records        []rootdomain.RawRecord
	HasMore        bool
	NextCursor     string
	NextWatermark  *time.Time
	SourceMode     string
	CollectedAtUTC time.Time
}
