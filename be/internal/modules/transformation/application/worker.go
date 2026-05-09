package application

import (
	"context"
	"log"

	collectiondomain "be_ads_project/internal/modules/collection/domain"
	ads "be_ads_project/internal/shared/ads"
	messagingdomain "be_ads_project/internal/shared/messaging/domain"
)

type RawReader interface {
	GetByIDs(context.Context, []int64) ([]ads.RawRecord, error)
}

type WorkerService struct {
	rawReader      RawReader
	transformation *Service
	logger         *log.Logger
}

func NewWorkerService(rawReader RawReader, transformation *Service, logger *log.Logger) *WorkerService {
	return &WorkerService{
		rawReader:      rawReader,
		transformation: transformation,
		logger:         logger,
	}
}

func (s *WorkerService) HandleEvent(ctx context.Context, event messagingdomain.RawEvent) error {
	records, err := s.rawReader.GetByIDs(ctx, event.RawRecordIDs)
	if err != nil {
		return err
	}

	batch := &collectiondomain.CollectedBatch{
		Target: collectiondomain.SyncTarget{
			Profile: ads.SyncProfile{
				ID:                event.ProfileID,
				PlatformAccountID: event.PlatformAccountID,
				Platform:          event.Platform,
				ObjectType:        event.ObjectType,
				IsEnabled:         true,
			},
			Bundle: collectiondomain.AccountBundle{
				Account: ads.PlatformAccount{
					ID:        event.PlatformAccountID,
					Platform:  event.Platform,
					AccountID: event.AccountID,
				},
			},
		},
		Records:        records,
		SourceMode:     "jetstream_async",
		CollectedAtUTC: event.CollectedAt,
	}

	s.logger.Printf("[transformer-worker] event_id=%s raw_records=%d", event.EventID, len(records))
	return s.transformation.HandleCollectedBatch(ctx, batch)
}
