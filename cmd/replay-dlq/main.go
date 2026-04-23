package main

import (
	"context"
	"flag"
	"log"
	"time"

	"be_ads_project/internal/config"
	ads "be_ads_project/internal/shared/ads"
	messagingdomain "be_ads_project/internal/shared/messaging/domain"
	msgjs "be_ads_project/internal/shared/messaging/infrastructure/jetstream"

	"github.com/nats-io/nats.go"
)

func main() {
	var (
		kind     = flag.String("kind", "raw_event", "dead letter kind: collect_job or raw_event")
		platform = flag.String("platform", "", "platform filter: facebook, google_ads, tiktok_ads")
		limit    = flag.Int("limit", 20, "max dlq messages to replay")
	)
	flag.Parse()

	cfg := config.Load()
	ctx := context.Background()

	client, err := msgjs.Connect(cfg.NATS)
	if err != nil {
		log.Fatalf("connect nats: %v", err)
	}
	defer client.Close()
	if err := client.EnsureStreams(ctx); err != nil {
		log.Fatalf("ensure streams: %v", err)
	}

	var platformFilter ads.Platform
	if *platform != "" {
		parsed, err := ads.ParsePlatform(*platform)
		if err != nil {
			log.Fatalf("parse platform: %v", err)
		}
		platformFilter = parsed
	}

	subject := msgjs.DeadLetterSubject(*kind, platformFilter)
	if platformFilter == "" {
		subject = "dlq." + *kind + ".*"
	}
	durable := "replay-" + *kind + "-" + time.Now().UTC().Format("20060102150405")

	if err := client.EnsureConsumer(ctx, msgjs.DeadLetterStream, &nats.ConsumerConfig{
		Durable:       durable,
		AckPolicy:     nats.AckExplicitPolicy,
		AckWait:       30 * time.Second,
		MaxDeliver:    1,
		FilterSubject: subject,
	}); err != nil {
		log.Fatalf("ensure dlq consumer: %v", err)
	}

	sub, err := client.PullSubscribe(msgjs.DeadLetterStream, subject, durable)
	if err != nil {
		log.Fatalf("pull subscribe dlq: %v", err)
	}

	replayed := 0
	for replayed < *limit {
		msgs, err := sub.Fetch(minInt(10, *limit-replayed), nats.MaxWait(2*time.Second))
		if err != nil {
			if err == nats.ErrTimeout {
				break
			}
			log.Fatalf("fetch dlq messages: %v", err)
		}
		if len(msgs) == 0 {
			break
		}

		for _, msg := range msgs {
			event, err := msgjs.DecodeMessage[messagingdomain.DeadLetterEvent](msg)
			if err != nil {
				log.Printf("skip invalid dlq message err=%v", err)
				_ = msg.Ack()
				continue
			}
			if err := client.PublishRaw(ctx, event.OriginalSubject, event.OriginalMessage); err != nil {
				log.Fatalf("replay message id=%s subject=%s: %v", event.ID, event.OriginalSubject, err)
			}
			log.Printf("replayed dlq id=%s kind=%s platform=%s subject=%s", event.ID, event.Kind, event.Platform, event.OriginalSubject)
			_ = msg.Ack()
			replayed++
		}
	}

	log.Printf("replay complete count=%d subject=%s", replayed, subject)
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}
