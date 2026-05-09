package jetstream

import (
	"context"

	messagingdomain "be_ads_project/internal/shared/messaging/domain"
	msgjs "be_ads_project/internal/shared/messaging/infrastructure/jetstream"
)

type Publisher struct {
	client *msgjs.Client
}

func NewPublisher(client *msgjs.Client) *Publisher {
	return &Publisher{client: client}
}

func (p *Publisher) PublishCollectJob(ctx context.Context, job messagingdomain.CollectJob) error {
	return p.client.PublishJSON(ctx, msgjs.CollectJobsShardSubject(job.Platform, job.ShardID), job)
}
