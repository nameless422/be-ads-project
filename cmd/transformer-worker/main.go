package main

import (
	"context"
	"errors"
	"log"
	"time"

	"be_ads_project/internal/config"
	"be_ads_project/internal/logx"
	rawmysql "be_ads_project/internal/modules/collection/infrastructure/rawstore/mysql"
	biqueryapp "be_ads_project/internal/modules/reporting/application"
	bimysql "be_ads_project/internal/modules/reporting/infrastructure/mysql"
	transformationapp "be_ads_project/internal/modules/transformation/application"
	transformationnormalizer "be_ads_project/internal/modules/transformation/infrastructure/normalizer"
	transformationprojector "be_ads_project/internal/modules/transformation/infrastructure/projector"
	chprojector "be_ads_project/internal/modules/transformation/infrastructure/projector/clickhouse"
	mysqlprojector "be_ads_project/internal/modules/transformation/infrastructure/projector/mysql"
	chplatform "be_ads_project/internal/platform/clickhouse"
	"be_ads_project/internal/platform/kratosx"
	mysqlplatform "be_ads_project/internal/platform/mysql"
	messagingdomain "be_ads_project/internal/shared/messaging/domain"
	msgjs "be_ads_project/internal/shared/messaging/infrastructure/jetstream"

	kratos "github.com/go-kratos/kratos/v2"
	"github.com/nats-io/nats.go"
)

func main() {
	cfg := config.Load()
	logger, closeFn, err := logx.New("logs/transformer-worker.log")
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

	servingDB, err := mysqlplatform.Open(cfg.ServingMySQL)
	if err != nil {
		log.Fatalf("open serving mysql: %v", err)
	}
	defer servingDB.Close()
	clickhouseDB, err := chplatform.Open(cfg.ClickHouse)
	if err != nil {
		log.Fatalf("open clickhouse: %v", err)
	}
	defer clickhouseDB.Close()

	mysqlWriteProjector := mysqlprojector.NewProjector(servingDB)
	if err := mysqlWriteProjector.Migrate(ctx); err != nil {
		log.Fatalf("migrate serving mysql oltp: %v", err)
	}
	mysqlReadRepo := bimysql.NewRepository(servingDB)
	if err := mysqlReadRepo.Migrate(ctx); err != nil {
		log.Fatalf("migrate serving mysql bi: %v", err)
	}
	clickhouseWriteProjector := chprojector.NewProjector(clickhouseDB, cfg.ClickHouse.Database)
	if err := clickhouseWriteProjector.Migrate(ctx); err != nil {
		log.Fatalf("migrate clickhouse: %v", err)
	}

	biService := biqueryapp.NewService(mysqlReadRepo, logger)
	projectorFanout := transformationprojector.NewFanout(mysqlWriteProjector, clickhouseWriteProjector, biService)
	normalizer := transformationnormalizer.NewRouter()
	transformation := transformationapp.NewService(normalizer, projectorFanout, logger)
	service := transformationapp.NewWorkerService(rawRepo, transformation, logger)

	sub, err := client.PullSubscribe(msgjs.RawEventsStream, msgjs.RawEventsSubject, msgjs.RawEventsConsumer)
	if err != nil {
		log.Fatalf("pull subscribe raw events: %v", err)
	}

	runLoop := func(ctx context.Context) error {
		logger.Printf("[transformer-worker] started")
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
				logger.Printf("[transformer-worker] fetch failed: %v", err)
				time.Sleep(time.Second)
				continue
			}

			for _, msg := range msgs {
				event, err := msgjs.DecodeMessage[messagingdomain.RawEvent](msg)
				if err != nil {
					logger.Printf("[transformer-worker] decode failed: %v", err)
					_ = msg.Term()
					continue
				}
				if err := service.HandleEvent(ctx, *event); err != nil {
					logger.Printf("[transformer-worker] handle failed event_id=%s err=%v", event.EventID, err)
					_ = msg.Nak()
					continue
				}
				_ = msg.Ack()
			}
		}
	}

	app := kratos.New(
		kratos.Name("transformer-worker"),
		kratos.Logger(kratosx.NewLogger(logger, "transformer-worker")),
		kratos.Server(kratosx.NewWorkerServer("transformer-worker", logger, runLoop)),
		kratos.StopTimeout(10*time.Second),
		kratos.AfterStop(func(context.Context) error {
			logger.Printf("[transformer-worker] stopped")
			return nil
		}),
	)
	if err := app.Run(); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("transformer-worker failed: %v", err)
	}
}
