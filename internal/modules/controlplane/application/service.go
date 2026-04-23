package application

import (
	"context"
	"fmt"
	"log"
	"slices"
	"time"

	ingestiondomain "be_ads_project/internal/modules/collection/domain"
	controlplanedomain "be_ads_project/internal/modules/controlplane/domain"
	ads "be_ads_project/internal/shared/ads"
)

type Service struct {
	targetProvider ingestiondomain.SyncTargetProvider
	publisher      controlplanedomain.JobPublisher
	leaseStore     controlplanedomain.LeaseStore
	shardCount     int
	logger         *log.Logger
}

func NewService(targetProvider ingestiondomain.SyncTargetProvider, publisher controlplanedomain.JobPublisher, leaseStore controlplanedomain.LeaseStore, shardCount int, logger *log.Logger) *Service {
	return &Service{
		targetProvider: targetProvider,
		publisher:      publisher,
		leaseStore:     leaseStore,
		shardCount:     shardCount,
		logger:         logger,
	}
}

func (s *Service) DispatchDueTargets(ctx context.Context) error {
	targets, err := s.targetProvider.ListTargets(ctx)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	if err := s.rebalanceShards(ctx, controlplanedomain.WorkerRoleCollector, targets, now); err != nil {
		return err
	}
	if err := s.rebalanceShards(ctx, controlplanedomain.WorkerRoleTransformer, targets, now); err != nil {
		return err
	}
	assignments, err := s.leaseStore.ListAssignments(ctx, controlplanedomain.WorkerRoleCollector)
	if err != nil {
		return err
	}
	owners := make(map[string]string, len(assignments))
	for _, item := range assignments {
		owners[assignmentKey(item.Platform, item.ShardID)] = item.WorkerID
	}

	s.logger.Printf("[control-plane] targets=%d shard_count=%d assignments=%d", len(targets), s.shardCount, len(assignments))
	for _, target := range targets {
		if !target.Profile.IsEnabled {
			continue
		}

		job := controlplanedomain.BuildCollectJob(target, now, s.shardCount)
		owner := owners[assignmentKey(job.Platform, job.ShardID)]
		if owner == "" {
			s.logger.Printf("[control-plane] skip profile=%s platform=%s shard=%d no_active_worker", job.ProfileID, job.Platform, job.ShardID)
			continue
		}

		if err := s.publisher.PublishCollectJob(ctx, job); err != nil {
			return err
		}
		s.logger.Printf(
			"[control-plane] dispatched job_id=%s profile=%s platform=%s shard=%d worker=%s account=%s object=%s",
			job.JobID,
			job.ProfileID,
			job.Platform,
			job.ShardID,
			owner,
			job.AccountID,
			job.ObjectType,
		)
	}
	return nil
}

func (s *Service) rebalanceShards(ctx context.Context, role controlplanedomain.WorkerRole, targets []ingestiondomain.SyncTarget, now time.Time) error {
	platforms := make([]ads.Platform, 0, 3)
	seen := make(map[ads.Platform]struct{})
	for _, target := range targets {
		if _, ok := seen[target.Profile.Platform]; ok {
			continue
		}
		seen[target.Profile.Platform] = struct{}{}
		platforms = append(platforms, target.Profile.Platform)
	}
	slices.Sort(platforms)

	workers, err := s.leaseStore.ListActiveWorkers(ctx, role, now)
	if err != nil {
		return err
	}
	assignments := controlplanedomain.BuildAssignments(role, platforms, s.shardCount, workers, now)
	return s.leaseStore.ReplaceAssignments(ctx, role, assignments)
}

func assignmentKey(platform ads.Platform, shardID int) string {
	return platform.String() + ":" + fmt.Sprintf("%d", shardID)
}
