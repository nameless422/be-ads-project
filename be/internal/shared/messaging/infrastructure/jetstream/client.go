package jetstream

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"be_ads_project/internal/config"
	ads "be_ads_project/internal/shared/ads"

	"github.com/nats-io/nats.go"
)

const (
	CollectJobsStream = "COLLECT_JOBS"
	RawEventsStream   = "RAW_EVENTS"
	DeadLetterStream  = "DEAD_LETTER_EVENTS"

	CollectJobsConsumerBase = "collector-workers"
	RawEventsConsumerBase   = "transformer-workers"
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
		{
			Name:      DeadLetterStream,
			Subjects:  []string{"dlq.>"},
			Storage:   nats.FileStorage,
			Retention: nats.LimitsPolicy,
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

func (c *Client) PublishRaw(ctx context.Context, subject string, body []byte) error {
	_, err := c.js.Publish(subject, body, nats.Context(ctx))
	return err
}

func (c *Client) StreamMessageCount(ctx context.Context, stream string) (uint64, error) {
	info, err := c.js.StreamInfo(stream, nats.Context(ctx))
	if err != nil {
		return 0, err
	}
	return info.State.Msgs, nil
}

func (c *Client) ListStreamMessages(ctx context.Context, stream string, limit int) ([]*nats.RawStreamMsg, error) {
	if limit <= 0 {
		limit = 20
	}
	info, err := c.js.StreamInfo(stream, nats.Context(ctx))
	if err != nil {
		return nil, err
	}
	if info.State.Msgs == 0 {
		return nil, nil
	}

	items := make([]*nats.RawStreamMsg, 0, limit)
	for seq := info.State.LastSeq; seq >= info.State.FirstSeq && len(items) < limit; seq-- {
		msg, err := c.js.GetMsg(stream, seq, nats.Context(ctx))
		if err != nil {
			continue
		}
		items = append(items, msg)
		if seq == info.State.FirstSeq {
			break
		}
	}
	return items, nil
}

func (c *Client) EnsureConsumer(ctx context.Context, stream string, cfg *nats.ConsumerConfig) error {
	if _, err := c.js.AddConsumer(stream, cfg, nats.Context(ctx)); err != nil && err != nats.ErrConsumerNameAlreadyInUse {
		if _, infoErr := c.js.ConsumerInfo(stream, cfg.Durable, nats.Context(ctx)); infoErr != nil {
			return err
		}
	}
	return nil
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

func CollectJobsSubjectForPlatform(platform ads.Platform) string {
	return fmt.Sprintf("collect.jobs.%s", platform)
}

func CollectJobsShardSubject(platform ads.Platform, shardID int) string {
	return fmt.Sprintf("collect.jobs.%s.shard.%d", platform, shardID)
}

func RawEventsSubjectForPlatform(platform ads.Platform) string {
	return fmt.Sprintf("raw.events.%s", platform)
}

func RawEventsShardSubject(platform ads.Platform, shardID int) string {
	return fmt.Sprintf("raw.events.%s.shard.%d", platform, shardID)
}

func DeadLetterSubject(kind string, platform ads.Platform) string {
	if platform == "" {
		return fmt.Sprintf("dlq.%s.unknown", kind)
	}
	return fmt.Sprintf("dlq.%s.%s", kind, platform)
}

func CollectJobsFilter(platforms []ads.Platform) string {
	return subjectFilter("collect.jobs", platforms)
}

func RawEventsFilter(platforms []ads.Platform) string {
	return subjectFilter("raw.events", platforms)
}

func ConsumerDurable(base string, platforms []ads.Platform) string {
	if len(platforms) == 0 {
		return base + "-all"
	}
	parts := make([]string, 0, len(platforms))
	for _, platform := range platforms {
		parts = append(parts, strings.ReplaceAll(string(platform), "_", "-"))
	}
	return base + "-" + strings.Join(parts, "-")
}

func subjectFilter(prefix string, platforms []ads.Platform) string {
	if len(platforms) != 1 {
		return prefix + ".*"
	}
	return fmt.Sprintf("%s.%s", prefix, platforms[0])
}

func CollectShardDurable(platform ads.Platform, shardID int) string {
	return fmt.Sprintf("collector-%s-s%d", strings.ReplaceAll(string(platform), "_", "-"), shardID)
}

func TransformShardDurable(platform ads.Platform, shardID int) string {
	return fmt.Sprintf("transformer-%s-s%d", strings.ReplaceAll(string(platform), "_", "-"), shardID)
}
