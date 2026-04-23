package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	biquerydomain "be_ads_project/internal/modules/reporting/domain"
	rootdomain "be_ads_project/internal/shared/ads"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Migrate(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS bi_account_snapshots (
		platform_account_id VARCHAR(64) PRIMARY KEY,
		platform VARCHAR(32) NOT NULL,
		account_id VARCHAR(64) NOT NULL,
		account_name VARCHAR(255) NOT NULL,
		last_source_mode VARCHAR(32) NOT NULL,
		last_object_type VARCHAR(32) NOT NULL,
		last_collected_at DATETIME NOT NULL,
		campaign_count INT NOT NULL,
		ad_group_count INT NOT NULL,
		ad_count INT NOT NULL,
		insight_count INT NOT NULL,
		updated_at DATETIME NOT NULL,
		UNIQUE KEY uk_snapshot_platform_account (platform, account_id)
	)`)
	return err
}

func (r *Repository) UpsertAccountSnapshot(ctx context.Context, snapshot biquerydomain.AccountSnapshot) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO bi_account_snapshots (
			platform_account_id, platform, account_id, account_name, last_source_mode, last_object_type,
			last_collected_at, campaign_count, ad_group_count, ad_count, insight_count, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			platform = VALUES(platform),
			account_id = VALUES(account_id),
			account_name = VALUES(account_name),
			last_source_mode = VALUES(last_source_mode),
			last_object_type = VALUES(last_object_type),
			last_collected_at = VALUES(last_collected_at),
			campaign_count = VALUES(campaign_count),
			ad_group_count = VALUES(ad_group_count),
			ad_count = VALUES(ad_count),
			insight_count = VALUES(insight_count),
			updated_at = VALUES(updated_at)
	`, snapshot.PlatformAccountID, snapshot.Platform, snapshot.AccountID, snapshot.AccountName, snapshot.LastSourceMode, snapshot.LastObjectType, snapshot.LastCollectedAt, snapshot.CampaignCount, snapshot.AdGroupCount, snapshot.AdCount, snapshot.InsightCount, time.Now().UTC())
	return err
}

func (r *Repository) GetAccountSnapshot(ctx context.Context, platformAccountID string) (*biquerydomain.AccountSnapshot, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT platform, platform_account_id, account_id, account_name, last_source_mode, last_object_type, last_collected_at,
			campaign_count, ad_group_count, ad_count, insight_count
		FROM bi_account_snapshots
		WHERE platform_account_id = ?
	`, platformAccountID)

	var item biquerydomain.AccountSnapshot
	var platform string
	var objectType string
	if err := row.Scan(&platform, &item.PlatformAccountID, &item.AccountID, &item.AccountName, &item.LastSourceMode, &objectType, &item.LastCollectedAt, &item.CampaignCount, &item.AdGroupCount, &item.AdCount, &item.InsightCount); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	item.Platform = rootdomain.Platform(platform)
	item.LastObjectType = rootdomain.ObjectType(objectType)
	return &item, nil
}

func (r *Repository) ListAccountSnapshots(ctx context.Context) ([]biquerydomain.AccountSnapshot, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT platform, platform_account_id, account_id, account_name, last_source_mode, last_object_type, last_collected_at,
			campaign_count, ad_group_count, ad_count, insight_count
		FROM bi_account_snapshots
		ORDER BY platform, account_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]biquerydomain.AccountSnapshot, 0)
	for rows.Next() {
		var item biquerydomain.AccountSnapshot
		var platform string
		var objectType string
		if err := rows.Scan(&platform, &item.PlatformAccountID, &item.AccountID, &item.AccountName, &item.LastSourceMode, &objectType, &item.LastCollectedAt, &item.CampaignCount, &item.AdGroupCount, &item.AdCount, &item.InsightCount); err != nil {
			return nil, err
		}
		item.Platform = rootdomain.Platform(platform)
		item.LastObjectType = rootdomain.ObjectType(objectType)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) ListCampaigns(ctx context.Context, filter biquerydomain.CampaignFilter) ([]biquerydomain.CampaignView, error) {
	args := make([]any, 0, 2)
	clauses := make([]string, 0, 2)
	if filter.Platform != "" {
		clauses = append(clauses, "platform = ?")
		args = append(args, filter.Platform)
	}
	if filter.AccountID != "" {
		clauses = append(clauses, "account_id = ?")
		args = append(args, filter.AccountID)
	}

	query := `
		SELECT platform, platform_account_id, account_id, platform_campaign_id, campaign_name, status, objective, buying_type,
			bidding_strategy, daily_budget, lifetime_budget, currency, start_time, end_time, source_updated_at, ingested_at
		FROM oltp_campaigns
	`
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY platform, account_id, platform_campaign_id"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]biquerydomain.CampaignView, 0)
	for rows.Next() {
		var item biquerydomain.CampaignView
		var platform string
		var startTime sql.NullTime
		var endTime sql.NullTime
		var sourceUpdatedAt sql.NullTime
		if err := rows.Scan(&platform, &item.PlatformAccountID, &item.AccountID, &item.PlatformCampaignID, &item.CampaignName, &item.Status, &item.Objective, &item.BuyingType, &item.BiddingStrategy, &item.DailyBudget, &item.LifetimeBudget, &item.Currency, &startTime, &endTime, &sourceUpdatedAt, &item.IngestedAt); err != nil {
			return nil, err
		}
		item.Platform = rootdomain.Platform(platform)
		if startTime.Valid {
			item.StartTime = startTime.Time
		}
		if endTime.Valid {
			item.EndTime = endTime.Time
		}
		if sourceUpdatedAt.Valid {
			item.SourceUpdatedAt = sourceUpdatedAt.Time
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

func (r *Repository) CountRecords(ctx context.Context, table string) (int64, error) {
	row := r.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", table))
	var count int64
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}
