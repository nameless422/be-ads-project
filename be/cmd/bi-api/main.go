package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"be_ads_project/internal/config"
	"be_ads_project/internal/logx"
	ingestiondomain "be_ads_project/internal/modules/collection/domain"
	ingestionprovider "be_ads_project/internal/modules/collection/infrastructure/provider"
	rawmysql "be_ads_project/internal/modules/collection/infrastructure/rawstore/mysql"
	controlplanedomain "be_ads_project/internal/modules/controlplane/domain"
	controlplanepub "be_ads_project/internal/modules/controlplane/infrastructure/jetstream"
	reportingapp "be_ads_project/internal/modules/reporting/application"
	reportingdomain "be_ads_project/internal/modules/reporting/domain"
	biclickhouse "be_ads_project/internal/modules/reporting/infrastructure/clickhouse"
	"be_ads_project/internal/modules/reporting/infrastructure/httpapi"
	"be_ads_project/internal/modules/reporting/infrastructure/localops"
	bimysql "be_ads_project/internal/modules/reporting/infrastructure/mysql"
	chplatform "be_ads_project/internal/platform/clickhouse"
	"be_ads_project/internal/platform/kratosx"
	mysqlplatform "be_ads_project/internal/platform/mysql"
	messagingdomain "be_ads_project/internal/shared/messaging/domain"
	msgjs "be_ads_project/internal/shared/messaging/infrastructure/jetstream"

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
	rawDB, err := mysqlplatform.Open(cfg.RawMySQL)
	if err != nil {
		log.Fatalf("open raw mysql: %v", err)
	}
	defer rawDB.Close()
	natsClient, err := msgjs.Connect(cfg.NATS)
	if err != nil {
		log.Fatalf("connect nats: %v", err)
	}
	defer natsClient.Close()
	if err := natsClient.EnsureStreams(ctx); err != nil {
		log.Fatalf("ensure jetstream streams: %v", err)
	}

	mysqlRepo := bimysql.NewRepository(servingDB)
	if err := mysqlRepo.Migrate(ctx); err != nil {
		log.Fatalf("migrate serving mysql bi: %v", err)
	}
	clickhouseRepo := biclickhouse.NewRepository(clickhouseDB, cfg.ClickHouse.Database)
	uaReportService := reportingapp.NewUAReportService(clickhouseRepo, mysqlRepo)
	controlRepo := bimysql.NewControlRepository(rawDB)
	rawRepo := rawmysql.NewRepository(rawDB)
	if err := rawRepo.Migrate(ctx); err != nil {
		log.Fatalf("migrate raw mysql: %v", err)
	}
	baseProvider := ingestionprovider.NewStaticProvider()
	provider := ingestionprovider.NewCheckpointProvider(baseProvider, rawRepo)
	jobPublisher := controlplanepub.NewPublisher(natsClient)
	rootDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("get working directory: %v", err)
	}
	server := httpapi.NewServer(
		mysqlRepo,
		clickhouseRepo,
		uaReportService,
		controlRepo,
		deadLetterReader{client: natsClient},
		controlActionHandler{provider: provider, publisher: jobPublisher, shardCount: cfg.ShardCount},
		localops.NewOperator(rootDir),
		logger,
	)
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

type deadLetterReader struct {
	client *msgjs.Client
}

func (d deadLetterReader) DeadLetterCount(ctx context.Context) (uint64, error) {
	return d.client.StreamMessageCount(ctx, msgjs.DeadLetterStream)
}

func (d deadLetterReader) ListDeadLetters(ctx context.Context, limit int) ([]reportingdomain.DeadLetterView, error) {
	msgs, err := d.client.ListStreamMessages(ctx, msgjs.DeadLetterStream, limit)
	if err != nil {
		return nil, err
	}
	items := make([]reportingdomain.DeadLetterView, 0, len(msgs))
	for _, msg := range msgs {
		var event struct {
			ID              string    `json:"id"`
			Kind            string    `json:"kind"`
			Platform        string    `json:"platform"`
			ErrorMessage    string    `json:"error_message"`
			DeliveryCount   uint64    `json:"delivery_count"`
			FailedAt        time.Time `json:"failed_at"`
			OriginalSubject string    `json:"original_subject"`
		}
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			continue
		}
		items = append(items, reportingdomain.DeadLetterView{
			StreamSequence:  msg.Sequence,
			Subject:         msg.Subject,
			ID:              event.ID,
			Kind:            event.Kind,
			Platform:        event.Platform,
			ErrorMessage:    event.ErrorMessage,
			DeliveryCount:   event.DeliveryCount,
			FailedAt:        event.FailedAt,
			OriginalSubject: event.OriginalSubject,
		})
	}
	return items, nil
}

func (d deadLetterReader) ReplayDeadLetters(ctx context.Context, req reportingdomain.ReplayRequest) (*reportingdomain.ActionResult, error) {
	items, err := d.ListDeadLetters(ctx, maxInt(req.Limit, 20))
	if err != nil {
		return nil, err
	}
	result := &reportingdomain.ActionResult{Items: make([]string, 0)}
	for _, item := range items {
		if req.Kind != "" && item.Kind != req.Kind {
			continue
		}
		if req.Platform != "" && item.Platform != req.Platform {
			continue
		}
		raw, err := d.client.ListStreamMessages(ctx, msgjs.DeadLetterStream, maxInt(req.Limit, 20))
		if err != nil {
			return nil, err
		}
		for _, msg := range raw {
			if msg.Sequence != item.StreamSequence {
				continue
			}
			var payload struct {
				OriginalSubject string          `json:"original_subject"`
				OriginalMessage json.RawMessage `json:"original_message"`
			}
			if err := json.Unmarshal(msg.Data, &payload); err != nil {
				continue
			}
			if err := d.client.PublishRaw(ctx, payload.OriginalSubject, payload.OriginalMessage); err != nil {
				return nil, err
			}
			result.Accepted++
			result.Items = append(result.Items, item.ID)
			break
		}
		if req.Limit > 0 && result.Accepted >= req.Limit {
			break
		}
	}
	return result, nil
}

type controlActionHandler struct {
	provider  ingestiondomain.SyncTargetProvider
	publisher interface {
		PublishCollectJob(context.Context, messagingdomain.CollectJob) error
	}
	shardCount int
}

func (h controlActionHandler) DispatchBackfill(ctx context.Context, req reportingdomain.BackfillRequest) (*reportingdomain.ActionResult, error) {
	targets, err := h.provider.ListTargets(ctx)
	if err != nil {
		return nil, err
	}
	result := &reportingdomain.ActionResult{Items: make([]string, 0)}
	now := time.Now().UTC()
	for _, target := range targets {
		if !matchTarget(target, req) {
			continue
		}
		job := controlplanedomain.BuildCollectJob(target, now, h.shardCount)
		if err := h.publisher.PublishCollectJob(ctx, job); err != nil {
			return nil, err
		}
		result.Accepted++
		result.Items = append(result.Items, job.JobID)
		if req.Limit > 0 && result.Accepted >= req.Limit {
			break
		}
	}
	return result, nil
}

func matchTarget(target ingestiondomain.SyncTarget, req reportingdomain.BackfillRequest) bool {
	if req.ProfileID != "" && target.Profile.ID != req.ProfileID {
		return false
	}
	if req.Platform != "" && target.Profile.Platform.String() != req.Platform {
		return false
	}
	if req.AccountID != "" && target.Bundle.Account.AccountID != req.AccountID {
		return false
	}
	if req.ObjectType != "" && target.Profile.ObjectType.String() != req.ObjectType {
		return false
	}
	return true
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
