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
	collectorapp "be_ads_project/internal/modules/collection/application"
	ingestioncollector "be_ads_project/internal/modules/collection/infrastructure/collector"
	collectorpub "be_ads_project/internal/modules/collection/infrastructure/jetstream"
	ingestionprovider "be_ads_project/internal/modules/collection/infrastructure/provider"
	rawmysql "be_ads_project/internal/modules/collection/infrastructure/rawstore/mysql"
	controlplanedomain "be_ads_project/internal/modules/controlplane/domain"
	controlmysql "be_ads_project/internal/modules/controlplane/infrastructure/mysql"
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

	leaseStore := controlmysql.NewLeaseStore(rawDB)
	if err := leaseStore.EnsureSchema(ctx); err != nil {
		log.Fatalf("ensure lease schema: %v", err)
	}

	baseProvider := ingestionprovider.NewStaticProvider()
	provider := ingestionprovider.NewCheckpointProvider(baseProvider, rawRepo)
	registry := ingestioncollector.NewRegistry()
	collector := ingestioncollector.NewMetaCollector(registry, logger)
	publisher := collectorpub.NewPublisher(client)
	service := collectorapp.NewService(provider, collector, rawRepo, logger)
	outboxRelay := collectorapp.NewOutboxRelay(rawRepo, publisher, logger)

	runLoop := func(ctx context.Context) error {
		logger.Printf(
			"[collector-worker] started worker_id=%s outbox_transport=%s concurrency=%d fetch_batch=%d shard_count=%d",
			cfg.WorkerID,
			cfg.OutboxTransport,
			cfg.CollectorRuntime.Concurrency,
			cfg.CollectorRuntime.FetchBatch,
			cfg.ShardCount,
		)

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

		go func() {
			ticker := time.NewTicker(cfg.HeartbeatInterval)
			defer ticker.Stop()
			for {
				if err := heartbeatWorker(ctx, leaseStore, cfg); err != nil {
					logger.Printf("[collector-worker] heartbeat failed: %v", err)
				}
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}
			}
		}()

		sem := make(chan struct{}, maxInt(1, cfg.CollectorRuntime.Concurrency))
		var wg sync.WaitGroup
		defer wg.Wait()

		manager := newShardSubscriptionManager(client, cfg, logger)

		handleMessage := func(msg *nats.Msg) {
			defer wg.Done()
			defer func() { <-sem }()

			job, err := msgjs.DecodeMessage[messagingdomain.CollectJob](msg)
			if err != nil {
				if publishDeadLetter(ctx, client, logger, messagingdomain.DeadLetterEvent{
					ID:              time.Now().UTC().Format("20060102150405.000000000"),
					Kind:            messagingdomain.DeadLetterKindCollectJob,
					Platform:        "",
					OriginalStream:  msgjs.CollectJobsStream,
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
			if err := service.HandleJob(ctx, *job); err != nil {
				logger.Printf("[collector-worker] handle failed job_id=%s shard=%d err=%v", job.JobID, job.ShardID, err)
				if shouldDeadLetter(msg, cfg.NATS.MaxDeliver) {
					if publishDeadLetter(ctx, client, logger, messagingdomain.DeadLetterEvent{
						ID:              "dlq_" + job.JobID,
						Kind:            messagingdomain.DeadLetterKindCollectJob,
						Platform:        job.Platform,
						OriginalStream:  msgjs.CollectJobsStream,
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
			if err := manager.Refresh(ctx, leaseStore, controlplanedomain.WorkerRoleCollector, cfg.WorkerID); err != nil {
				logger.Printf("[collector-worker] refresh shard subscriptions failed: %v", err)
			}

			for _, sub := range manager.List() {
				msgs, err := sub.Sub.Fetch(maxInt(1, cfg.CollectorRuntime.FetchBatch), nats.MaxWait(500*time.Millisecond))
				if err != nil {
					if errors.Is(err, nats.ErrTimeout) {
						continue
					}
					logger.Printf("[collector-worker] fetch failed subject=%s err=%v", sub.Subject, err)
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

type shardSubscription struct {
	Assignment controlplanedomain.ShardAssignment
	Subject    string
	Sub        *nats.Subscription
}

type shardSubscriptionManager struct {
	client *msgjs.Client
	cfg    config.Config
	logger *log.Logger

	mu   sync.Mutex
	subs map[string]*shardSubscription
}

func newShardSubscriptionManager(client *msgjs.Client, cfg config.Config, logger *log.Logger) *shardSubscriptionManager {
	return &shardSubscriptionManager{
		client: client,
		cfg:    cfg,
		logger: logger,
		subs:   make(map[string]*shardSubscription),
	}
}

func (m *shardSubscriptionManager) Refresh(ctx context.Context, leaseStore controlplanedomain.LeaseStore, role controlplanedomain.WorkerRole, workerID string) error {
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

		subject := msgjs.CollectJobsShardSubject(item.Platform, item.ShardID)
		durable := msgjs.CollectShardDurable(item.Platform, item.ShardID)
		if err := m.client.EnsureConsumer(ctx, msgjs.CollectJobsStream, &nats.ConsumerConfig{
			Durable:       durable,
			AckPolicy:     nats.AckExplicitPolicy,
			AckWait:       m.cfg.NATS.AckWait,
			MaxDeliver:    m.cfg.NATS.MaxDeliver,
			MaxAckPending: maxInt(m.cfg.NATS.MaxAckPending, m.cfg.CollectorRuntime.Concurrency*m.cfg.CollectorRuntime.FetchBatch),
			FilterSubject: subject,
		}); err != nil {
			return err
		}
		sub, err := m.client.PullSubscribe(msgjs.CollectJobsStream, subject, durable)
		if err != nil {
			return err
		}
		m.subs[key] = &shardSubscription{
			Assignment: item,
			Subject:    subject,
			Sub:        sub,
		}
		m.logger.Printf("[collector-worker] assigned shard platform=%s shard=%d durable=%s", item.Platform, item.ShardID, durable)
	}

	for key, sub := range m.subs {
		if _, keep := desired[key]; keep {
			continue
		}
		_ = sub.Sub.Unsubscribe()
		delete(m.subs, key)
		m.logger.Printf("[collector-worker] released shard subject=%s", sub.Subject)
	}

	return nil
}

func (m *shardSubscriptionManager) List() []*shardSubscription {
	m.mu.Lock()
	defer m.mu.Unlock()

	items := make([]*shardSubscription, 0, len(m.subs))
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

func heartbeatWorker(ctx context.Context, leaseStore controlplanedomain.LeaseStore, cfg config.Config) error {
	now := time.Now().UTC()
	return leaseStore.HeartbeatWorker(ctx, controlplanedomain.WorkerLease{
		Role:           controlplanedomain.WorkerRoleCollector,
		WorkerID:       cfg.WorkerID,
		SupportedScope: cfg.WorkerPlatforms,
		Capacity:       maxInt(1, cfg.CollectorRuntime.Concurrency),
		LastSeenAt:     now,
		ExpiresAt:      now.Add(cfg.LeaseTTL),
	})
}

func shardKey(platform interface{ String() string }, shardID int) string {
	return fmt.Sprintf("%s:%d", platform.String(), shardID)
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
		logger.Printf("[collector-worker] dlq publish failed subject=%s err=%v", subject, err)
		return err
	}
	logger.Printf("[collector-worker] dlq published subject=%s deliveries=%d err=%s", subject, event.DeliveryCount, event.ErrorMessage)
	return nil
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
