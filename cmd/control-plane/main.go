package main

import (
	"context"
	"log"
	"time"

	"be_ads_project/internal/config"
	"be_ads_project/internal/logx"
	ingestionprovider "be_ads_project/internal/modules/collection/infrastructure/provider"
	controlplaneapp "be_ads_project/internal/modules/controlplane/application"
	controlplanepub "be_ads_project/internal/modules/controlplane/infrastructure/jetstream"
	controlmysql "be_ads_project/internal/modules/controlplane/infrastructure/mysql"
	"be_ads_project/internal/platform/kratosx"
	mysqlplatform "be_ads_project/internal/platform/mysql"
	msgjs "be_ads_project/internal/shared/messaging/infrastructure/jetstream"

	kratos "github.com/go-kratos/kratos/v2"
)

func main() {
	cfg := config.Load()
	logger, closeFn, err := logx.New("logs/control-plane.log")
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
	leaseStore := controlmysql.NewLeaseStore(rawDB)
	if err := leaseStore.EnsureSchema(ctx); err != nil {
		log.Fatalf("ensure lease schema: %v", err)
	}

	provider := ingestionprovider.NewStaticProvider()
	publisher := controlplanepub.NewPublisher(client)
	service := controlplaneapp.NewService(provider, publisher, leaseStore, cfg.ShardCount, logger)

	runLoop := func(ctx context.Context) error {
		runOnce := func() {
			if err := service.DispatchDueTargets(ctx); err != nil {
				logger.Printf("[control-plane] dispatch failed: %v", err)
			}
		}

		runOnce()
		ticker := time.NewTicker(cfg.SyncInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case tick := <-ticker.C:
				logger.Printf("[control-plane] tick=%s", tick.UTC().Format(time.RFC3339))
				runOnce()
			}
		}
	}

	app := kratos.New(
		kratos.Name("control-plane"),
		kratos.Logger(kratosx.NewLogger(logger, "control-plane")),
		kratos.Server(kratosx.NewWorkerServer("control-plane", logger, runLoop)),
		kratos.StopTimeout(10*time.Second),
		kratos.BeforeStart(func(context.Context) error {
			logger.Printf("[control-plane] kratos app start interval=%s", cfg.SyncInterval)
			return nil
		}),
		kratos.AfterStop(func(context.Context) error {
			logger.Printf("[control-plane] stopped")
			return nil
		}),
	)
	if err := app.Run(); err != nil {
		log.Fatalf("control-plane failed: %v", err)
	}
}
