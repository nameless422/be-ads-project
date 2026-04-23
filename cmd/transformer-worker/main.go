package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"be_ads_project/internal/config"
	"be_ads_project/internal/logx"
	rawmysql "be_ads_project/internal/modules/collection/infrastructure/rawstore/mysql"
	controlplanedomain "be_ads_project/internal/modules/controlplane/domain"
	controlmysql "be_ads_project/internal/modules/controlplane/infrastructure/mysql"
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
	leaseStore := controlmysql.NewLeaseStore(rawDB)
	if err := leaseStore.EnsureSchema(ctx); err != nil {
		log.Fatalf("ensure lease schema: %v", err)
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

	runLoop := func(ctx context.Context) error {
		logger.Printf(
			"[transformer-worker] started worker_id=%s concurrency=%d fetch_batch=%d shard_count=%d",
			cfg.WorkerID,
			cfg.TransformerRuntime.Concurrency,
			cfg.TransformerRuntime.FetchBatch,
			cfg.ShardCount,
		)
		go func() {
			ticker := time.NewTicker(cfg.HeartbeatInterval)
			defer ticker.Stop()
			for {
				if err := heartbeatTransformer(ctx, leaseStore, cfg); err != nil {
					logger.Printf("[transformer-worker] heartbeat failed: %v", err)
				}
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}
			}
		}()

		sem := make(chan struct{}, maxInt(1, cfg.TransformerRuntime.Concurrency))
		var wg sync.WaitGroup
		defer wg.Wait()
		manager := newTransformShardSubscriptionManager(client, cfg, logger)

		handleMessage := func(msg *nats.Msg) {
			defer wg.Done()
			defer func() { <-sem }()

			event, err := msgjs.DecodeMessage[messagingdomain.RawEvent](msg)
			if err != nil {
				if publishDeadLetter(ctx, client, logger, messagingdomain.DeadLetterEvent{
					ID:              time.Now().UTC().Format("20060102150405.000000000"),
					Kind:            messagingdomain.DeadLetterKindRawEvent,
					Platform:        "",
					OriginalStream:  msgjs.RawEventsStream,
					OriginalSubject: msg.Subject,
					OriginalMessage: append([]byte(nil), msg.Data...),
					ErrorMessage:    err.Error(),
					DeliveryCount:   deliveryCount(msg),
					FailedAt:        time.Now().UTC(),
				}) == nil {
					_ = msg.Ack()
				} else {
					_ = msg.Nak()
				}
				return
			}
			if err := service.HandleEvent(ctx, *event); err != nil {
				logger.Printf("[transformer-worker] handle failed event_id=%s err=%v", event.EventID, err)
				if shouldDeadLetter(msg, cfg.NATS.MaxDeliver) {
					if publishDeadLetter(ctx, client, logger, messagingdomain.DeadLetterEvent{
						ID:              "dlq_" + event.EventID,
						Kind:            messagingdomain.DeadLetterKindRawEvent,
						Platform:        event.Platform,
						OriginalStream:  msgjs.RawEventsStream,
						OriginalSubject: msg.Subject,
						OriginalMessage: append([]byte(nil), msg.Data...),
						ErrorMessage:    err.Error(),
						DeliveryCount:   deliveryCount(msg),
						FailedAt:        time.Now().UTC(),
					}) == nil {
						_ = msg.Ack()
						return
					}
				}
				_ = msg.Nak()
				return
			}
			_ = msg.Ack()
		}

		refreshTicker := time.NewTicker(cfg.HeartbeatInterval)
		defer refreshTicker.Stop()

		for {
			if err := manager.Refresh(ctx, leaseStore, controlplanedomain.WorkerRoleTransformer, cfg.WorkerID); err != nil {
				logger.Printf("[transformer-worker] refresh shard subscriptions failed: %v", err)
			}

			for _, sub := range manager.List() {
				msgs, err := sub.Sub.Fetch(maxInt(1, cfg.TransformerRuntime.FetchBatch), nats.MaxWait(500*time.Millisecond))
				if err != nil {
					if errors.Is(err, nats.ErrTimeout) {
						continue
					}
					logger.Printf("[transformer-worker] fetch failed subject=%s err=%v", sub.Subject, err)
					continue
				}

				for _, msg := range msgs {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case sem <- struct{}{}:
						wg.Add(1)
						go handleMessage(msg)
					}
				}
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-refreshTicker.C:
			default:
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

func shouldDeadLetter(msg *nats.Msg, maxDeliver int) bool {
	if maxDeliver <= 0 {
		return false
	}
	meta, err := msg.Metadata()
	if err != nil {
		return false
	}
	return meta.NumDelivered >= uint64(maxDeliver)
}

func deliveryCount(msg *nats.Msg) uint64 {
	meta, err := msg.Metadata()
	if err != nil {
		return 0
	}
	return meta.NumDelivered
}

func publishDeadLetter(ctx context.Context, client *msgjs.Client, logger *log.Logger, event messagingdomain.DeadLetterEvent) error {
	subject := msgjs.DeadLetterSubject(string(event.Kind), event.Platform)
	if err := client.PublishJSON(ctx, subject, event); err != nil {
		logger.Printf("[transformer-worker] dlq publish failed subject=%s err=%v", subject, err)
		return err
	}
	logger.Printf("[transformer-worker] dlq published subject=%s deliveries=%d err=%s", subject, event.DeliveryCount, event.ErrorMessage)
	return nil
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

type transformShardSubscription struct {
	Assignment controlplanedomain.ShardAssignment
	Subject    string
	Sub        *nats.Subscription
}

type transformShardSubscriptionManager struct {
	client *msgjs.Client
	cfg    config.Config
	logger *log.Logger

	mu   sync.Mutex
	subs map[string]*transformShardSubscription
}

func newTransformShardSubscriptionManager(client *msgjs.Client, cfg config.Config, logger *log.Logger) *transformShardSubscriptionManager {
	return &transformShardSubscriptionManager{
		client: client,
		cfg:    cfg,
		logger: logger,
		subs:   make(map[string]*transformShardSubscription),
	}
}

func (m *transformShardSubscriptionManager) Refresh(ctx context.Context, leaseStore controlplanedomain.LeaseStore, role controlplanedomain.WorkerRole, workerID string) error {
	assignments, err := leaseStore.ListAssignmentsByWorker(ctx, role, workerID)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	desired := make(map[string]controlplanedomain.ShardAssignment, len(assignments))
	for _, item := range assignments {
		key := shardKey(item.Platform, item.ShardID)
		desired[key] = item
		if _, exists := m.subs[key]; exists {
			continue
		}

		subject := msgjs.RawEventsShardSubject(item.Platform, item.ShardID)
		durable := msgjs.TransformShardDurable(item.Platform, item.ShardID)
		if err := m.client.EnsureConsumer(ctx, msgjs.RawEventsStream, &nats.ConsumerConfig{
			Durable:       durable,
			AckPolicy:     nats.AckExplicitPolicy,
			AckWait:       m.cfg.NATS.AckWait,
			MaxDeliver:    m.cfg.NATS.MaxDeliver,
			MaxAckPending: maxInt(m.cfg.NATS.MaxAckPending, m.cfg.TransformerRuntime.Concurrency*m.cfg.TransformerRuntime.FetchBatch),
			FilterSubject: subject,
		}); err != nil {
			return err
		}
		sub, err := m.client.PullSubscribe(msgjs.RawEventsStream, subject, durable)
		if err != nil {
			return err
		}
		m.subs[key] = &transformShardSubscription{
			Assignment: item,
			Subject:    subject,
			Sub:        sub,
		}
		m.logger.Printf("[transformer-worker] assigned shard platform=%s shard=%d durable=%s", item.Platform, item.ShardID, durable)
	}

	for key, sub := range m.subs {
		if _, keep := desired[key]; keep {
			continue
		}
		_ = sub.Sub.Unsubscribe()
		delete(m.subs, key)
		m.logger.Printf("[transformer-worker] released shard subject=%s", sub.Subject)
	}
	return nil
}

func (m *transformShardSubscriptionManager) List() []*transformShardSubscription {
	m.mu.Lock()
	defer m.mu.Unlock()

	items := make([]*transformShardSubscription, 0, len(m.subs))
	for _, item := range m.subs {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Assignment.Platform == items[j].Assignment.Platform {
			return items[i].Assignment.ShardID < items[j].Assignment.ShardID
		}
		return items[i].Assignment.Platform < items[j].Assignment.Platform
	})
	return items
}

func heartbeatTransformer(ctx context.Context, leaseStore controlplanedomain.LeaseStore, cfg config.Config) error {
	now := time.Now().UTC()
	return leaseStore.HeartbeatWorker(ctx, controlplanedomain.WorkerLease{
		Role:           controlplanedomain.WorkerRoleTransformer,
		WorkerID:       cfg.WorkerID,
		SupportedScope: cfg.WorkerPlatforms,
		Capacity:       maxInt(1, cfg.TransformerRuntime.Concurrency),
		LastSeenAt:     now,
		ExpiresAt:      now.Add(cfg.LeaseTTL),
	})
}

func shardKey(platform interface{ String() string }, shardID int) string {
	return fmt.Sprintf("%s:%d", platform.String(), shardID)
}
