package main

import (
	"context"
	"log"
	"time"

	"be_ads_project/internal/config"
	"be_ads_project/internal/logx"
	biclickhouse "be_ads_project/internal/modules/reporting/infrastructure/clickhouse"
	"be_ads_project/internal/modules/reporting/infrastructure/httpapi"
	bimysql "be_ads_project/internal/modules/reporting/infrastructure/mysql"
	chplatform "be_ads_project/internal/platform/clickhouse"
	"be_ads_project/internal/platform/kratosx"
	mysqlplatform "be_ads_project/internal/platform/mysql"

	kratos "github.com/go-kratos/kratos/v2"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

func main() {
	cfg := config.Load()
	logger, closeFn, err := logx.New("logs/bi-api.log")
	if err != nil {
		log.Fatalf("init logger: %v", err)
	}
	defer closeFn()

	ctx := context.Background()
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

	mysqlRepo := bimysql.NewRepository(servingDB)
	if err := mysqlRepo.Migrate(ctx); err != nil {
		log.Fatalf("migrate serving mysql bi: %v", err)
	}
	clickhouseRepo := biclickhouse.NewRepository(clickhouseDB, cfg.ClickHouse.Database)

	server := httpapi.NewServer(mysqlRepo, clickhouseRepo, logger)
	httpServer := khttp.NewServer(
		khttp.Address(":"+cfg.HTTPPort),
		khttp.Timeout(5*time.Second),
		khttp.Filter(
			kratosx.RecoveryFilter(logger),
			kratosx.AccessLogFilter(logger),
		),
	)
	httpServer.HandlePrefix("/", server.Handler())

	app := kratos.New(
		kratos.Name("bi-api"),
		kratos.Logger(kratosx.NewLogger(logger, "bi-api")),
		kratos.Server(httpServer),
		kratos.StopTimeout(10*time.Second),
		kratos.BeforeStart(func(context.Context) error {
			logger.Printf("[bi-api] kratos http listen=:%s", cfg.HTTPPort)
			return nil
		}),
		kratos.AfterStop(func(context.Context) error {
			logger.Printf("[bi-api] stopped")
			return nil
		}),
	)
	if err := app.Run(); err != nil {
		log.Fatalf("bi-api server failed: %v", err)
	}
}
