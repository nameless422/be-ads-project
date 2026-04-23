package jetstream

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"be_ads_project/internal/config"

	"github.com/nats-io/nats.go"
)

const (
	CollectJobsStream   = "COLLECT_JOBS"
	CollectJobsSubject  = "collect.jobs.dispatch"
	CollectJobsConsumer = "collector-workers"

	RawEventsStream   = "RAW_EVENTS"
	RawEventsSubject  = "raw.events.ingested"
	RawEventsConsumer = "transformer-workers"
)

type Client struct {
	nc *nats.Conn
	js nats.JetStreamContext
}

func Connect(cfg config.NATSConfig) (*Client, error) {
	nc, err := nats.Connect(cfg.URL, nats.Name("be-ads-project"))
	if err != nil {
		return nil, err
	}
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, err
	}
	return &Client{nc: nc, js: js}, nil
}

func (c *Client) Close() {
	if c.nc != nil {
		c.nc.Close()
	}
}

func (c *Client) EnsureStreams(ctx context.Context) error {
	streams := []*nats.StreamConfig{
		{
			Name:      CollectJobsStream,
			Subjects:  []string{"collect.jobs.>"},
			Storage:   nats.FileStorage,
			Retention: nats.WorkQueuePolicy,
			Replicas:  1,
		},
		{
			Name:      RawEventsStream,
			Subjects:  []string{"raw.events.>"},
			Storage:   nats.FileStorage,
			Retention: nats.WorkQueuePolicy,
			Replicas:  1,
		},
	}
	for _, cfg := range streams {
		if _, err := c.js.AddStream(cfg, nats.Context(ctx)); err != nil && err != nats.ErrStreamNameAlreadyInUse {
			if _, infoErr := c.js.StreamInfo(cfg.Name, nats.Context(ctx)); infoErr != nil {
				return err
			}
		}
	}

	consumers := []struct {
		stream string
		cfg    *nats.ConsumerConfig
	}{
		{
			stream: CollectJobsStream,
			cfg: &nats.ConsumerConfig{
				Durable:       CollectJobsConsumer,
				AckPolicy:     nats.AckExplicitPolicy,
				AckWait:       30 * time.Second,
				MaxDeliver:    5,
				FilterSubject: CollectJobsSubject,
			},
		},
		{
			stream: RawEventsStream,
			cfg: &nats.ConsumerConfig{
				Durable:       RawEventsConsumer,
				AckPolicy:     nats.AckExplicitPolicy,
				AckWait:       30 * time.Second,
				MaxDeliver:    5,
				FilterSubject: RawEventsSubject,
			},
		},
	}

	for _, item := range consumers {
		if _, err := c.js.AddConsumer(item.stream, item.cfg, nats.Context(ctx)); err != nil && err != nats.ErrConsumerNameAlreadyInUse {
			if _, infoErr := c.js.ConsumerInfo(item.stream, item.cfg.Durable, nats.Context(ctx)); infoErr != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Client) PublishJSON(ctx context.Context, subject string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = c.js.Publish(subject, body, nats.Context(ctx))
	return err
}

func (c *Client) PullSubscribe(stream, subject, durable string) (*nats.Subscription, error) {
	return c.js.PullSubscribe(subject, durable, nats.BindStream(stream))
}

func DecodeMessage[T any](msg *nats.Msg) (*T, error) {
	var payload T
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		return nil, fmt.Errorf("decode jetstream message: %w", err)
	}
	return &payload, nil
}
