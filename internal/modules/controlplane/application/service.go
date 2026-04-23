package application

import (
	"context"
	"log"
	"time"

	ingestiondomain "be_ads_project/internal/modules/collection/domain"
	controlplanedomain "be_ads_project/internal/modules/controlplane/domain"
)

type Service struct {
	targetProvider ingestiondomain.SyncTargetProvider
	publisher      controlplanedomain.JobPublisher
	logger         *log.Logger
}

func NewService(targetProvider ingestiondomain.SyncTargetProvider, publisher controlplanedomain.JobPublisher, logger *log.Logger) *Service {
	return &Service{
		targetProvider: targetProvider,
		publisher:      publisher,
		logger:         logger,
	}
}

func (s *Service) DispatchDueTargets(ctx context.Context) error {
	targets, err := s.targetProvider.ListTargets(ctx)
	if err != nil {
		return err
	}

	s.logger.Printf("[control-plane] targets=%d", len(targets))
	for _, target := range targets {
		if !target.Profile.IsEnabled {
			continue
		}

		job := controlplanedomain.BuildCollectJob(target, time.Now().UTC())

		if err := s.publisher.PublishCollectJob(ctx, job); err != nil {
			return err
		}
		s.logger.Printf(
			"[control-plane] dispatched job_id=%s profile=%s platform=%s account=%s object=%s",
			job.JobID,
			job.ProfileID,
			job.Platform,
			job.AccountID,
			job.ObjectType,
		)
	}
	return nil
}
