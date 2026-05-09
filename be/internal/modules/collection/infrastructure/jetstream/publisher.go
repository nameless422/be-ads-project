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

func (p *Publisher) PublishRawEvent(ctx context.Context, event messagingdomain.RawEvent) error {
	return p.client.PublishJSON(ctx, msgjs.RawEventsShardSubject(event.Platform, event.ShardID), event)
}
