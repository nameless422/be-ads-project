package application

import (
	"context"
	"encoding/json"
	"log"
	"time"

	ingestiondomain "be_ads_project/internal/modules/collection/domain"
	rawdomain "be_ads_project/internal/modules/collection/domain"
	ingestioncollector "be_ads_project/internal/modules/collection/infrastructure/collector"
	messagingdomain "be_ads_project/internal/shared/messaging/domain"
	msgjs "be_ads_project/internal/shared/messaging/infrastructure/jetstream"
)

type RawStore interface {
	SaveBatch(context.Context, rawdomain.SaveBatchInput) ([]rawdomain.StoredRawRecord, error)
	ListPendingOutbox(context.Context, int) ([]rawdomain.OutboxEvent, error)
	MarkOutboxPublished(context.Context, int64, time.Time) error
	MarkOutboxFailed(context.Context, int64, string, time.Time) error
}

type RawEventPublisher interface {
	PublishRawEvent(context.Context, messagingdomain.RawEvent) error
}

type Service struct {
	resolver      ingestiondomain.SyncTargetResolver
	collector     *ingestioncollector.MetaCollector
	rawRepository RawStore
	logger        *log.Logger
}

func NewService(
	resolver ingestiondomain.SyncTargetResolver,
	collector *ingestioncollector.MetaCollector,
	rawRepository RawStore,
	logger *log.Logger,
) *Service {
	return &Service{
		resolver:      resolver,
		collector:     collector,
		rawRepository: rawRepository,
		logger:        logger,
	}
}

func (s *Service) HandleJob(ctx context.Context, job messagingdomain.CollectJob) error {
	target, err := s.resolver.ResolveTarget(ctx, job.ProfileID)
	if err != nil {
		return err
	}

	batch, err := s.collector.Collect(ctx, *target)
	if err != nil {
		return err
	}

	event := messagingdomain.RawEvent{
		EventID:           "evt_" + job.JobID,
		JobID:             job.JobID,
		TraceID:           job.TraceID,
		ProfileID:         job.ProfileID,
		Platform:          job.Platform,
		ShardID:           job.ShardID,
		PlatformAccountID: job.PlatformAccountID,
		AccountID:         job.AccountID,
		ObjectType:        job.ObjectType,
		CollectedAt:       batch.CollectedAtUTC,
	}
	stored, err := s.rawRepository.SaveBatch(ctx, rawdomain.SaveBatchInput{
		JobID:             job.JobID,
		TraceID:           job.TraceID,
		ProfileID:         job.ProfileID,
		Platform:          job.Platform,
		PlatformAccountID: job.PlatformAccountID,
		AccountID:         job.AccountID,
		ObjectType:        job.ObjectType,
		SourceMode:        batch.SourceMode,
		SourceCursor:      batch.NextCursor,
		SourceWatermark:   job.WatermarkValue,
		CollectedAt:       batch.CollectedAtUTC.Format(time.RFC3339),
		Records:           batch.Records,
		OutboxEvents: []rawdomain.OutboxEventInput{
			{
				EventID:       event.EventID,
				AggregateType: "raw_batch",
				AggregateID:   job.JobID,
				EventType:     "raw.ingested",
				Topic:         msgjs.RawEventsShardSubject(job.Platform, job.ShardID),
				RawEvent:      &event,
				AvailableAt:   time.Now().UTC(),
			},
		},
	})
	if err != nil {
		return err
	}

	rawIDs := make([]int64, 0, len(stored))
	for _, item := range stored {
		rawIDs = append(rawIDs, item.ID)
	}
	event.RawRecordIDs = rawIDs

	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	s.logger.Printf("[collector-worker] job_id=%s raw_records=%d outbox_event_id=%s payload_bytes=%d", job.JobID, len(rawIDs), event.EventID, len(body))
	return nil
}

type OutboxRelay struct {
	repository RawStore
	publisher  RawEventPublisher
	logger     *log.Logger
}

func NewOutboxRelay(repository RawStore, publisher RawEventPublisher, logger *log.Logger) *OutboxRelay {
	return &OutboxRelay{
		repository: repository,
		publisher:  publisher,
		logger:     logger,
	}
}

func (r *OutboxRelay) FlushPending(ctx context.Context, limit int) error {
	events, err := r.repository.ListPendingOutbox(ctx, limit)
	if err != nil {
		return err
	}
	for _, event := range events {
		var payload messagingdomain.RawEvent
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			next := time.Now().UTC().Add(30 * time.Second)
			_ = r.repository.MarkOutboxFailed(ctx, event.ID, err.Error(), next)
			r.logger.Printf("[collector-worker] outbox decode failed id=%d event_id=%s err=%v", event.ID, event.EventID, err)
			continue
		}
		if err := r.publisher.PublishRawEvent(ctx, payload); err != nil {
			next := time.Now().UTC().Add(5 * time.Second)
			_ = r.repository.MarkOutboxFailed(ctx, event.ID, err.Error(), next)
			r.logger.Printf("[collector-worker] outbox publish failed id=%d event_id=%s err=%v", event.ID, event.EventID, err)
			continue
		}
		now := time.Now().UTC()
		if err := r.repository.MarkOutboxPublished(ctx, event.ID, now); err != nil {
			return err
		}
		r.logger.Printf("[collector-worker] outbox published id=%d event_id=%s", event.ID, event.EventID)
	}
	return nil
}
