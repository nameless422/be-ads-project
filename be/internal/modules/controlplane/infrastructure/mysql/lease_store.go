package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	controlplanedomain "be_ads_project/internal/modules/controlplane/domain"
	ads "be_ads_project/internal/shared/ads"
)

type LeaseStore struct {
	db *sql.DB
}

func NewLeaseStore(db *sql.DB) *LeaseStore {
	return &LeaseStore{db: db}
}

func (s *LeaseStore) EnsureSchema(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS worker_leases (
		worker_role VARCHAR(32) NOT NULL,
		worker_id VARCHAR(128) NOT NULL,
		platform_scope VARCHAR(255) NOT NULL,
		capacity INT NOT NULL,
		last_seen_at DATETIME NOT NULL,
		expires_at DATETIME NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		PRIMARY KEY (worker_role, worker_id),
		KEY idx_worker_leases_expires_at (worker_role, expires_at)
	)`)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS shard_assignments (
		worker_role VARCHAR(32) NOT NULL,
		platform VARCHAR(32) NOT NULL,
		shard_id INT NOT NULL,
		worker_id VARCHAR(128) NOT NULL,
		updated_at DATETIME NOT NULL,
		PRIMARY KEY (worker_role, platform, shard_id),
		KEY idx_shard_assignments_worker_id (worker_role, worker_id)
	)`)
	return err
}

func (s *LeaseStore) HeartbeatWorker(ctx context.Context, lease controlplanedomain.WorkerLease) error {
	scope := make([]string, 0, len(lease.SupportedScope))
	for _, item := range lease.SupportedScope {
		scope = append(scope, item.String())
	}
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO worker_leases (worker_role, worker_id, platform_scope, capacity, last_seen_at, expires_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			platform_scope = VALUES(platform_scope),
			capacity = VALUES(capacity),
			last_seen_at = VALUES(last_seen_at),
			expires_at = VALUES(expires_at),
			updated_at = VALUES(updated_at)
	`, lease.Role, lease.WorkerID, strings.Join(scope, ","), maxInt(1, lease.Capacity), lease.LastSeenAt.UTC(), lease.ExpiresAt.UTC(), now, now)
	return err
}

func (s *LeaseStore) ListActiveWorkers(ctx context.Context, role controlplanedomain.WorkerRole, now time.Time) ([]controlplanedomain.WorkerLease, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT worker_role, worker_id, platform_scope, capacity, last_seen_at, expires_at
		FROM worker_leases
		WHERE worker_role = ? AND expires_at >= ?
		ORDER BY worker_id
	`, role, now.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]controlplanedomain.WorkerLease, 0)
	for rows.Next() {
		var item controlplanedomain.WorkerLease
		var scope string
		if err := rows.Scan(&item.Role, &item.WorkerID, &scope, &item.Capacity, &item.LastSeenAt, &item.ExpiresAt); err != nil {
			return nil, err
		}
		item.SupportedScope = parsePlatforms(scope)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *LeaseStore) ReplaceAssignments(ctx context.Context, role controlplanedomain.WorkerRole, assignments []controlplanedomain.ShardAssignment) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM shard_assignments WHERE worker_role = ?`, role); err != nil {
		return err
	}
	if len(assignments) == 0 {
		return tx.Commit()
	}

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO shard_assignments (worker_role, platform, shard_id, worker_id, updated_at) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, item := range assignments {
		if _, err := stmt.ExecContext(ctx, item.Role, item.Platform, item.ShardID, item.WorkerID, item.UpdatedAt.UTC()); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *LeaseStore) ListAssignments(ctx context.Context, role controlplanedomain.WorkerRole) ([]controlplanedomain.ShardAssignment, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT worker_role, platform, shard_id, worker_id, updated_at FROM shard_assignments WHERE worker_role = ? ORDER BY platform, shard_id`, role)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]controlplanedomain.ShardAssignment, 0)
	for rows.Next() {
		var item controlplanedomain.ShardAssignment
		if err := rows.Scan(&item.Role, &item.Platform, &item.ShardID, &item.WorkerID, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *LeaseStore) ListAssignmentsByWorker(ctx context.Context, role controlplanedomain.WorkerRole, workerID string) ([]controlplanedomain.ShardAssignment, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT worker_role, platform, shard_id, worker_id, updated_at
		FROM shard_assignments
		WHERE worker_role = ? AND worker_id = ?
		ORDER BY platform, shard_id
	`, role, workerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]controlplanedomain.ShardAssignment, 0)
	for rows.Next() {
		var item controlplanedomain.ShardAssignment
		if err := rows.Scan(&item.Role, &item.Platform, &item.ShardID, &item.WorkerID, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func parsePlatforms(raw string) []ads.Platform {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	items := make([]ads.Platform, 0, len(parts))
	for _, part := range parts {
		platform, err := ads.ParsePlatform(strings.TrimSpace(part))
		if err != nil {
			continue
		}
		items = append(items, platform)
	}
	return items
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func (s *LeaseStore) String() string {
	return fmt.Sprintf("LeaseStore(%p)", s)
}
