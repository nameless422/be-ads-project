package mysql

import (
	"context"
	"database/sql"

	reportingdomain "be_ads_project/internal/modules/reporting/domain"
	ads "be_ads_project/internal/shared/ads"
)

type ControlRepository struct {
	db *sql.DB
}

func NewControlRepository(db *sql.DB) *ControlRepository {
	return &ControlRepository{db: db}
}

func (r *ControlRepository) ListWorkerLeases(ctx context.Context) ([]reportingdomain.WorkerLeaseView, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT worker_role, worker_id, platform_scope, capacity, last_seen_at, expires_at
		FROM worker_leases
		ORDER BY worker_role, worker_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]reportingdomain.WorkerLeaseView, 0)
	for rows.Next() {
		var item reportingdomain.WorkerLeaseView
		if err := rows.Scan(&item.WorkerRole, &item.WorkerID, &item.PlatformScope, &item.Capacity, &item.LastSeenAt, &item.ExpiresAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *ControlRepository) ListShardAssignments(ctx context.Context) ([]reportingdomain.ShardAssignmentView, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT worker_role, platform, shard_id, worker_id, updated_at
		FROM shard_assignments
		ORDER BY worker_role, platform, shard_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]reportingdomain.ShardAssignmentView, 0)
	for rows.Next() {
		var item reportingdomain.ShardAssignmentView
		var platform string
		if err := rows.Scan(&item.WorkerRole, &platform, &item.ShardID, &item.WorkerID, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.Platform = ads.Platform(platform)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *ControlRepository) CountRawRecords(ctx context.Context) (int64, error) {
	return countFromTable(ctx, r.db, "raw_records")
}

func (r *ControlRepository) CountOutboxByStatus(ctx context.Context, status string) (int64, error) {
	row := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM outbox_events WHERE status = ?`, status)
	var count int64
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func countFromTable(ctx context.Context, db *sql.DB, table string) (int64, error) {
	row := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table)
	var count int64
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}
