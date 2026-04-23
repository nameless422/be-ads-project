package domain

import (
	"time"

	ingestiondomain "be_ads_project/internal/modules/collection/domain"
	rootdomain "be_ads_project/internal/shared/ads"
)

type NormalizedBatch struct {
	Collected    *ingestiondomain.CollectedBatch
	Payload      *rootdomain.NormalizedPayload
	NormalizedAt time.Time
}
