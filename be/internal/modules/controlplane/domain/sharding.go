package domain

import (
	"context"
	"hash/fnv"
	"sort"
	"time"

	ads "be_ads_project/internal/shared/ads"
)

type WorkerLease struct {
	Role           WorkerRole
	WorkerID       string
	SupportedScope []ads.Platform
	Capacity       int
	LastSeenAt     time.Time
	ExpiresAt      time.Time
}

type ShardAssignment struct {
	Role      WorkerRole
	Platform  ads.Platform
	ShardID   int
	WorkerID  string
	UpdatedAt time.Time
}

type WorkerRole string

const (
	WorkerRoleCollector   WorkerRole = "collector"
	WorkerRoleTransformer WorkerRole = "transformer"
)

type LeaseStore interface {
	EnsureSchema(context.Context) error
	ListActiveWorkers(context.Context, WorkerRole, time.Time) ([]WorkerLease, error)
	ReplaceAssignments(context.Context, WorkerRole, []ShardAssignment) error
	ListAssignments(context.Context, WorkerRole) ([]ShardAssignment, error)
	ListAssignmentsByWorker(context.Context, WorkerRole, string) ([]ShardAssignment, error)
	HeartbeatWorker(context.Context, WorkerLease) error
}

func ComputeShardID(profileID string, platform ads.Platform, objectType ads.ObjectType, shardCount int) int {
	if shardCount <= 1 {
		return 0
	}
	sum := fnv.New32a()
	_, _ = sum.Write([]byte(profileID))
	_, _ = sum.Write([]byte(platform))
	_, _ = sum.Write([]byte(objectType))
	return int(sum.Sum32() % uint32(shardCount))
}

func BuildAssignments(role WorkerRole, platforms []ads.Platform, shardCount int, workers []WorkerLease, now time.Time) []ShardAssignment {
	if shardCount <= 0 {
		shardCount = 1
	}
	assignments := make([]ShardAssignment, 0, len(platforms)*shardCount)
	for _, platform := range platforms {
		eligible := eligibleWorkers(workers, platform)
		if len(eligible) == 0 {
			continue
		}
		sort.SliceStable(eligible, func(i, j int) bool {
			if eligible[i].Capacity == eligible[j].Capacity {
				return eligible[i].WorkerID < eligible[j].WorkerID
			}
			return eligible[i].Capacity > eligible[j].Capacity
		})
		for shardID := 0; shardID < shardCount; shardID++ {
			worker := eligible[shardID%len(eligible)]
			assignments = append(assignments, ShardAssignment{
				Role:      role,
				Platform:  platform,
				ShardID:   shardID,
				WorkerID:  worker.WorkerID,
				UpdatedAt: now.UTC(),
			})
		}
	}
	return assignments
}

func eligibleWorkers(workers []WorkerLease, platform ads.Platform) []WorkerLease {
	items := make([]WorkerLease, 0, len(workers))
	for _, worker := range workers {
		if supportsPlatform(worker.SupportedScope, platform) {
			items = append(items, worker)
		}
	}
	return items
}

func supportsPlatform(scope []ads.Platform, platform ads.Platform) bool {
	if len(scope) == 0 {
		return true
	}
	for _, item := range scope {
		if item == platform {
			return true
		}
	}
	return false
}
