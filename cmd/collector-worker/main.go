package main

import (
	"context"
	"errors"
	"log"
	"time"

	"be_ads_project/internal/config"
	"be_ads_project/internal/logx"
	collectorapp "be_ads_project/internal/modules/collection/application"
	ingestioncollector "be_ads_project/internal/modules/collection/infrastructure/collector"
	collectorpub "be_ads_project/internal/modules/collection/infrastructure/jetstream"
	ingestionprovider "be_ads_project/internal/modules/collection/infrastructure/provider"
	rawmysql "be_ads_project/internal/modules/collection/infrastructure/rawstore/mysql"
	"be_ads_project/internal/platform/kratosx"
	mysqlplatform "be_ads_project/internal/platform/mysql"
	messagingdomain "be_ads_project/internal/shared/messaging/domain"
	msgjs "be_ads_project/internal/shared/messaging/infrastructure/jetstream"

	kratos "github.com/go-kratos/kratos/v2"
	"github.com/nats-io/nats.go"
)

func main() {
	cfg := config.Load()
	logger, closeFn, err := logx.New("logs/collector-worker.log")
	if err != nil {
		log.Fatalf("init logger: %v", err)
	}
	defer closeFn()

	ctx := context.Background()
	client, err := msgjs.Connect(cfg.NATS)
	if err != nil {
		log.Fatalf("connect nats: %v", err)
	}
	defer client.Close()
	if err := client.EnsureStreams(ctx); err != nil {
		log.Fatalf("ensure jetstream streams: %v", err)
	}

	rawDB, err := mysqlplatform.Open(cfg.RawMySQL)
	if err != nil {
		log.Fatalf("open raw mysql: %v", err)
	}
	defer rawDB.Close()

	rawRepo := rawmysql.NewRepository(rawDB)
	if err := rawRepo.Migrate(ctx); err != nil {
		log.Fatalf("migrate raw mysql: %v", err)
	}

	provider := ingestionprovider.NewStaticProvider()
	registry := ingestioncollector.NewRegistry()
	collector := ingestioncollector.NewMetaCollector(registry, logger)
	publisher := collectorpub.NewPublisher(client)
	service := collectorapp.NewService(provider, collector, rawRepo, logger)
	outboxRelay := collectorapp.NewOutboxRelay(rawRepo, publisher, logger)

	sub, err := client.PullSubscribe(msgjs.CollectJobsStream, msgjs.CollectJobsSubject, msgjs.CollectJobsConsumer)
	if err != nil {
		log.Fatalf("pull subscribe collect jobs: %v", err)
	}

	runLoop := func(ctx context.Context) error {
		logger.Printf("[collector-worker] started outbox_transport=%s", cfg.OutboxTransport)
		if cfg.OutboxTransport == "relay" {
			go func() {
				ticker := time.NewTicker(2 * time.Second)
				defer ticker.Stop()
				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						if err := outboxRelay.FlushPending(ctx, 50); err != nil {
							logger.Printf("[collector-worker] outbox flush failed: %v", err)
						}
					}
				}
			}()
		}

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			msgs, err := sub.Fetch(10, nats.MaxWait(5*time.Second))
			if err != nil {
				if errors.Is(err, nats.ErrTimeout) {
					continue
				}
				logger.Printf("[collector-worker] fetch failed: %v", err)
				time.Sleep(time.Second)
				continue
			}

			for _, msg := range msgs {
				job, err := msgjs.DecodeMessage[messagingdomain.CollectJob](msg)
				if err != nil {
					logger.Printf("[collector-worker] decode failed: %v", err)
					_ = msg.Term()
					continue
				}
				if err := service.HandleJob(ctx, *job); err != nil {
					logger.Printf("[collector-worker] handle failed job_id=%s err=%v", job.JobID, err)
					_ = msg.Nak()
					continue
				}
				_ = msg.Ack()
			}
		}
	}

	app := kratos.New(
		kratos.Name("collector-worker"),
		kratos.Logger(kratosx.NewLogger(logger, "collector-worker")),
		kratos.Server(kratosx.NewWorkerServer("collector-worker", logger, runLoop)),
		kratos.StopTimeout(10*time.Second),
		kratos.AfterStop(func(context.Context) error {
			logger.Printf("[collector-worker] stopped")
			return nil
		}),
	)
	if err := app.Run(); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("collector-worker failed: %v", err)
	}
}
