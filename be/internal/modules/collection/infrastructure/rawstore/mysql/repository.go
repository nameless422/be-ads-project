package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	rawdomain "be_ads_project/internal/modules/collection/domain"
	rootdomain "be_ads_project/internal/shared/ads"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Migrate(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS raw_records (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		job_id VARCHAR(128) NOT NULL,
		trace_id VARCHAR(128) NOT NULL,
		profile_id VARCHAR(128) NOT NULL,
		platform VARCHAR(32) NOT NULL,
		platform_account_id VARCHAR(64) NOT NULL,
		account_id VARCHAR(64) NOT NULL,
		object_type VARCHAR(32) NOT NULL,
		resource_id VARCHAR(255) NOT NULL,
		payload_json JSON NOT NULL,
		source_mode VARCHAR(32) NOT NULL,
		source_cursor VARCHAR(255) NOT NULL,
		source_watermark VARCHAR(64) NOT NULL,
		collected_at DATETIME NOT NULL,
		created_at DATETIME NOT NULL,
		KEY idx_raw_records_job_id (job_id),
		KEY idx_raw_records_profile_id (profile_id),
		KEY idx_raw_records_lookup (platform_account_id, object_type, collected_at)
	)`)
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS sync_checkpoints (
		profile_id VARCHAR(128) PRIMARY KEY,
		platform VARCHAR(32) NOT NULL,
		platform_account_id VARCHAR(64) NOT NULL,
		account_id VARCHAR(64) NOT NULL,
		object_type VARCHAR(32) NOT NULL,
		next_cursor VARCHAR(255) NOT NULL,
		next_watermark VARCHAR(64) NOT NULL,
		last_collected_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		KEY idx_sync_checkpoints_lookup (platform_account_id, object_type, updated_at)
	)`)
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO sync_checkpoints (
			profile_id, platform, platform_account_id, account_id, object_type,
			next_cursor, next_watermark, last_collected_at, updated_at
		)
		SELECT
			rr.profile_id,
			rr.platform,
			rr.platform_account_id,
			rr.account_id,
			rr.object_type,
			rr.source_cursor,
			rr.source_watermark,
			rr.collected_at,
			UTC_TIMESTAMP()
		FROM raw_records rr
		INNER JOIN (
			SELECT profile_id, MAX(id) AS max_id
			FROM raw_records
			GROUP BY profile_id
		) latest ON latest.max_id = rr.id
		LEFT JOIN sync_checkpoints sc ON sc.profile_id = rr.profile_id
		WHERE sc.profile_id IS NULL
	`)
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS outbox_events (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		event_id VARCHAR(128) NOT NULL,
		aggregate_type VARCHAR(64) NOT NULL,
		aggregate_id VARCHAR(255) NOT NULL,
		event_type VARCHAR(64) NOT NULL,
		topic VARCHAR(128) NOT NULL,
		payload_json JSON NOT NULL,
		status VARCHAR(16) NOT NULL,
		attempt_count INT NOT NULL DEFAULT 0,
		last_error TEXT NULL,
		available_at DATETIME NOT NULL,
		published_at DATETIME NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		UNIQUE KEY uk_outbox_event_id (event_id),
		KEY idx_outbox_status_available (status, available_at, id)
	)`)
	return err
}

func (r *Repository) SaveBatch(ctx context.Context, input rawdomain.SaveBatchInput) ([]rawdomain.StoredRawRecord, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	collectedAt := time.Now().UTC()
	if input.CollectedAt != "" {
		if parsed, err := time.Parse(time.RFC3339, input.CollectedAt); err == nil {
			collectedAt = parsed.UTC()
		}
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO raw_records (
			job_id, trace_id, profile_id, platform, platform_account_id, account_id, object_type, resource_id,
			payload_json, source_mode, source_cursor, source_watermark, collected_at, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	outboxStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO outbox_events (
			event_id, aggregate_type, aggregate_id, event_type, topic, payload_json,
			status, attempt_count, last_error, available_at, published_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return nil, err
	}
	defer outboxStmt.Close()

	checkpointStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO sync_checkpoints (
			profile_id, platform, platform_account_id, account_id, object_type,
			next_cursor, next_watermark, last_collected_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			platform = VALUES(platform),
			platform_account_id = VALUES(platform_account_id),
			account_id = VALUES(account_id),
			object_type = VALUES(object_type),
			next_cursor = VALUES(next_cursor),
			next_watermark = VALUES(next_watermark),
			last_collected_at = VALUES(last_collected_at),
			updated_at = VALUES(updated_at)
	`)
	if err != nil {
		return nil, err
	}
	defer checkpointStmt.Close()

	items := make([]rawdomain.StoredRawRecord, 0, len(input.Records))
	now := time.Now().UTC()
	for _, record := range input.Records {
		result, err := stmt.ExecContext(
			ctx,
			input.JobID,
			input.TraceID,
			input.ProfileID,
			record.Platform,
			record.PlatformAccountID,
			input.AccountID,
			record.ObjectType,
			record.ResourceID,
			record.Payload,
			input.SourceMode,
			input.SourceCursor,
			input.SourceWatermark,
			collectedAt,
			now,
		)
		if err != nil {
			return nil, err
		}
		id, err := result.LastInsertId()
		if err != nil {
			return nil, err
		}
		items = append(items, rawdomain.StoredRawRecord{
			ID:                id,
			JobID:             input.JobID,
			TraceID:           input.TraceID,
			ProfileID:         input.ProfileID,
			Platform:          record.Platform,
			PlatformAccountID: record.PlatformAccountID,
			AccountID:         input.AccountID,
			ObjectType:        record.ObjectType,
			ResourceID:        record.ResourceID,
			Payload:           record.Payload,
			SourceMode:        input.SourceMode,
			SourceCursor:      input.SourceCursor,
			SourceWatermark:   input.SourceWatermark,
			CollectedAt:       collectedAt,
		})
	}

	rawIDs := make([]int64, 0, len(items))
	for _, item := range items {
		rawIDs = append(rawIDs, item.ID)
	}

	for _, event := range input.OutboxEvents {
		payload := event.Payload
		if event.RawEvent != nil {
			cloned := *event.RawEvent
			cloned.RawRecordIDs = append([]int64(nil), rawIDs...)
			encoded, err := json.Marshal(cloned)
			if err != nil {
				return nil, err
			}
			payload = encoded
		}
		availableAt := event.AvailableAt.UTC()
		if availableAt.IsZero() {
			availableAt = now
		}
		if _, err := outboxStmt.ExecContext(
			ctx,
			event.EventID,
			event.AggregateType,
			event.AggregateID,
			event.EventType,
			event.Topic,
			payload,
			rawdomain.OutboxStatusPending,
			0,
			nil,
			availableAt,
			nil,
			now,
			now,
		); err != nil {
			return nil, err
		}
	}

	checkpoint := input.Checkpoint
	if checkpoint.ProfileID == "" {
		checkpoint.ProfileID = input.ProfileID
	}
	if checkpoint.LastCollectedAt.IsZero() {
		checkpoint.LastCollectedAt = collectedAt
	}
	if _, err := checkpointStmt.ExecContext(
		ctx,
		checkpoint.ProfileID,
		input.Platform,
		input.PlatformAccountID,
		input.AccountID,
		input.ObjectType,
		checkpoint.NextCursor,
		checkpoint.NextWatermark,
		checkpoint.LastCollectedAt.UTC(),
		now,
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *Repository) LatestCheckpoint(ctx context.Context, profileID string) (*rawdomain.SyncCheckpoint, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT profile_id, next_cursor, next_watermark, last_collected_at
		FROM sync_checkpoints
		WHERE profile_id = ?
	`, profileID)

	var checkpoint rawdomain.SyncCheckpoint
	if err := row.Scan(
		&checkpoint.ProfileID,
		&checkpoint.NextCursor,
		&checkpoint.NextWatermark,
		&checkpoint.LastCollectedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &checkpoint, nil
}

func (r *Repository) ListPendingOutbox(ctx context.Context, limit int) ([]rawdomain.OutboxEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, event_id, aggregate_type, aggregate_id, event_type, topic, CAST(payload_json AS CHAR),
			status, attempt_count, COALESCE(last_error, ''), available_at, published_at, created_at
		FROM outbox_events
		WHERE status IN (?, ?) AND available_at <= UTC_TIMESTAMP()
		ORDER BY id
		LIMIT ?
	`, rawdomain.OutboxStatusPending, rawdomain.OutboxStatusFailed, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]rawdomain.OutboxEvent, 0, limit)
	for rows.Next() {
		var item rawdomain.OutboxEvent
		var payload string
		var status string
		var publishedAt sql.NullTime
		if err := rows.Scan(
			&item.ID,
			&item.EventID,
			&item.AggregateType,
			&item.AggregateID,
			&item.EventType,
			&item.Topic,
			&payload,
			&status,
			&item.AttemptCount,
			&item.LastError,
			&item.AvailableAt,
			&publishedAt,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		item.Payload = []byte(payload)
		item.Status = rawdomain.OutboxStatus(status)
		if publishedAt.Valid {
			value := publishedAt.Time.UTC()
			item.PublishedAt = &value
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) MarkOutboxPublished(ctx context.Context, id int64, publishedAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE outbox_events
		SET status = ?, published_at = ?, last_error = NULL, updated_at = ?
		WHERE id = ?
	`, rawdomain.OutboxStatusPublished, publishedAt.UTC(), publishedAt.UTC(), id)
	return err
}

func (r *Repository) MarkOutboxFailed(ctx context.Context, id int64, lastError string, availableAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE outbox_events
		SET status = ?, attempt_count = attempt_count + 1, last_error = ?, available_at = ?, updated_at = ?
		WHERE id = ?
	`, rawdomain.OutboxStatusFailed, lastError, availableAt.UTC(), time.Now().UTC(), id)
	return err
}

func (r *Repository) DeleteRawRecordsBefore(ctx context.Context, before time.Time, limit int) (int64, error) {
	if limit <= 0 {
		limit = 1000
	}
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM raw_records
		WHERE collected_at < ?
		ORDER BY id
		LIMIT ?
	`, before.UTC(), limit)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *Repository) DeletePublishedOutboxBefore(ctx context.Context, before time.Time, limit int) (int64, error) {
	if limit <= 0 {
		limit = 1000
	}
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM outbox_events
		WHERE status = ? AND published_at IS NOT NULL AND published_at < ?
		ORDER BY id
		LIMIT ?
	`, rawdomain.OutboxStatusPublished, before.UTC(), limit)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *Repository) GetByIDs(ctx context.Context, ids []int64) ([]rootdomain.RawRecord, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	query := `
		SELECT platform, platform_account_id, object_type, resource_id, CAST(payload_json AS CHAR)
		FROM raw_records
		WHERE id IN (` + strings.Join(placeholders, ",") + `)
		ORDER BY id
	`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]rootdomain.RawRecord, 0, len(ids))
	for rows.Next() {
		var platform string
		var accountID string
		var objectType string
		var resourceID string
		var payload string
		if err := rows.Scan(&platform, &accountID, &objectType, &resourceID, &payload); err != nil {
			return nil, err
		}
		items = append(items, rootdomain.RawRecord{
			Platform:          rootdomain.Platform(platform),
			PlatformAccountID: accountID,
			ObjectType:        rootdomain.ObjectType(objectType),
			ResourceID:        resourceID,
			Payload:           []byte(payload),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(items) != len(ids) {
		return nil, fmt.Errorf("raw records not found for all ids requested=%d found=%d", len(ids), len(items))
	}
	return items, nil
}
