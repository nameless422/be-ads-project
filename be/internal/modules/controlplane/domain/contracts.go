package domain

import (
	"context"

	messagingdomain "be_ads_project/internal/shared/messaging/domain"
)

type JobPublisher interface {
	PublishCollectJob(context.Context, messagingdomain.CollectJob) error
}
