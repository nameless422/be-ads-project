package domain

import (
	"fmt"
	"time"

	collectiondomain "be_ads_project/internal/modules/collection/domain"
	messagingdomain "be_ads_project/internal/shared/messaging/domain"
)

func BuildCollectJob(target collectiondomain.SyncTarget, now time.Time) messagingdomain.CollectJob {
	return messagingdomain.CollectJob{
		JobID:             fmt.Sprintf("job_%s_%d", target.Profile.ID, now.UnixNano()),
		TraceID:           fmt.Sprintf("trace_%s_%d", target.Profile.ID, now.UnixNano()),
		ProfileID:         target.Profile.ID,
		Platform:          target.Profile.Platform,
		PlatformAccountID: target.Profile.PlatformAccountID,
		AccountID:         target.Bundle.Account.AccountID,
		ObjectType:        target.Profile.ObjectType,
		SyncMode:          target.Profile.SyncMode,
		WatermarkValue:    target.Profile.WatermarkValue,
		PageToken:         target.Profile.PageToken,
		DispatchedAt:      now,
	}
}
