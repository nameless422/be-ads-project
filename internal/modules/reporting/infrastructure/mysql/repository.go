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
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS bi_game_kpis (
		platform VARCHAR(32) NOT NULL,
		platform_account_id VARCHAR(64) NOT NULL,
		platform_campaign_id VARCHAR(128) NOT NULL DEFAULT '',
		platform_ad_group_id VARCHAR(128) NOT NULL DEFAULT '',
		platform_ad_id VARCHAR(128) NOT NULL DEFAULT '',
		stat_date DATE NOT NULL,
		country VARCHAR(64) NOT NULL DEFAULT '',
		os VARCHAR(32) NOT NULL DEFAULT '',
		placement VARCHAR(128) NOT NULL DEFAULT '',
		creative_id VARCHAR(128) NOT NULL DEFAULT '',
		creative_type VARCHAR(64) NOT NULL DEFAULT '',
		optimization_goal VARCHAR(64) NOT NULL DEFAULT '',
		bid_type VARCHAR(64) NOT NULL DEFAULT '',
		targeting TEXT,
		installs BIGINT NOT NULL DEFAULT 0,
		activations BIGINT NOT NULL DEFAULT 0,
		registrations BIGINT NOT NULL DEFAULT 0,
		tutorial_completions BIGINT NOT NULL DEFAULT 0,
		role_creations BIGINT NOT NULL DEFAULT 0,
		level_x_users BIGINT NOT NULL DEFAULT 0,
		purchasers BIGINT NOT NULL DEFAULT 0,
		purchase_count BIGINT NOT NULL DEFAULT 0,
		first_purchase_amount DECIMAL(18, 4) NOT NULL DEFAULT 0,
		revenue_d1 DECIMAL(18, 4) NOT NULL DEFAULT 0,
		revenue_d7 DECIMAL(18, 4) NOT NULL DEFAULT 0,
		revenue_d30 DECIMAL(18, 4) NOT NULL DEFAULT 0,
		ad_revenue DECIMAL(18, 4) NOT NULL DEFAULT 0,
		total_revenue DECIMAL(18, 4) NOT NULL DEFAULT 0,
		retention_d1 DECIMAL(18, 4) NOT NULL DEFAULT 0,
		retention_d3 DECIMAL(18, 4) NOT NULL DEFAULT 0,
		retention_d7 DECIMAL(18, 4) NOT NULL DEFAULT 0,
		retention_d30 DECIMAL(18, 4) NOT NULL DEFAULT 0,
		ltv_d7 DECIMAL(18, 4) NOT NULL DEFAULT 0,
		ltv_d30 DECIMAL(18, 4) NOT NULL DEFAULT 0,
		avg_online_duration_seconds BIGINT NOT NULL DEFAULT 0,
		task_completion_rate DECIMAL(18, 4) NOT NULL DEFAULT 0,
		high_value_payer_ratio DECIMAL(18, 4) NOT NULL DEFAULT 0,
		raw_payload JSON NULL,
		updated_at DATETIME NOT NULL,
		PRIMARY KEY (platform, platform_account_id, platform_campaign_id, platform_ad_group_id, platform_ad_id, stat_date, country, os)
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

func (r *Repository) UpsertGameKPIs(ctx context.Context, items []biquerydomain.GameKPIRecord) error {
	for _, item := range items {
		_, err := r.db.ExecContext(ctx, `
			INSERT INTO bi_game_kpis (
				platform, platform_account_id, platform_campaign_id, platform_ad_group_id, platform_ad_id,
				stat_date, country, os, placement, creative_id, creative_type, optimization_goal, bid_type, targeting,
				installs, activations, registrations, tutorial_completions, role_creations, level_x_users, purchasers, purchase_count,
				first_purchase_amount, revenue_d1, revenue_d7, revenue_d30, ad_revenue, total_revenue,
				retention_d1, retention_d3, retention_d7, retention_d30, ltv_d7, ltv_d30,
				avg_online_duration_seconds, task_completion_rate, high_value_payer_ratio, raw_payload, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
				placement = VALUES(placement),
				creative_id = VALUES(creative_id),
				creative_type = VALUES(creative_type),
				optimization_goal = VALUES(optimization_goal),
				bid_type = VALUES(bid_type),
				targeting = VALUES(targeting),
				installs = VALUES(installs),
				activations = VALUES(activations),
				registrations = VALUES(registrations),
				tutorial_completions = VALUES(tutorial_completions),
				role_creations = VALUES(role_creations),
				level_x_users = VALUES(level_x_users),
				purchasers = VALUES(purchasers),
				purchase_count = VALUES(purchase_count),
				first_purchase_amount = VALUES(first_purchase_amount),
				revenue_d1 = VALUES(revenue_d1),
				revenue_d7 = VALUES(revenue_d7),
				revenue_d30 = VALUES(revenue_d30),
				ad_revenue = VALUES(ad_revenue),
				total_revenue = VALUES(total_revenue),
				retention_d1 = VALUES(retention_d1),
				retention_d3 = VALUES(retention_d3),
				retention_d7 = VALUES(retention_d7),
				retention_d30 = VALUES(retention_d30),
				ltv_d7 = VALUES(ltv_d7),
				ltv_d30 = VALUES(ltv_d30),
				avg_online_duration_seconds = VALUES(avg_online_duration_seconds),
				task_completion_rate = VALUES(task_completion_rate),
				high_value_payer_ratio = VALUES(high_value_payer_ratio),
				raw_payload = VALUES(raw_payload),
				updated_at = VALUES(updated_at)
		`,
			item.Platform,
			item.PlatformAccountID,
			item.PlatformCampaignID,
			item.PlatformAdGroupID,
			item.PlatformAdID,
			item.StatDate,
			item.Country,
			item.OS,
			item.Placement,
			item.CreativeID,
			item.CreativeType,
			item.OptimizationGoal,
			item.BidType,
			item.Targeting,
			item.Installs,
			item.Activations,
			item.Registrations,
			item.TutorialCompletions,
			item.RoleCreations,
			item.LevelXUsers,
			item.Purchasers,
			item.PurchaseCount,
			decimalOrZero(item.FirstPurchaseAmount),
			decimalOrZero(item.RevenueD1),
			decimalOrZero(item.RevenueD7),
			decimalOrZero(item.RevenueD30),
			decimalOrZero(item.AdRevenue),
			decimalOrZero(item.TotalRevenue),
			decimalOrZero(item.RetentionD1),
			decimalOrZero(item.RetentionD3),
			decimalOrZero(item.RetentionD7),
			decimalOrZero(item.RetentionD30),
			decimalOrZero(item.LTVD7),
			decimalOrZero(item.LTVD30),
			item.AvgOnlineDurationSeconds,
			decimalOrZero(item.TaskCompletionRate),
			decimalOrZero(item.HighValuePayerRatio),
			nullJSON(item.RawPayload),
			time.Now().UTC(),
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) ListGameKPIs(ctx context.Context, filter biquerydomain.GameKPIQueryFilter) ([]biquerydomain.GameKPIRecord, error) {
	args := make([]any, 0, 10)
	clauses := make([]string, 0, 8)
	if filter.Platform != "" {
		clauses = append(clauses, "platform = ?")
		args = append(args, filter.Platform)
	}
	if filter.AccountID != "" {
		clauses = append(clauses, "platform_account_id = ?")
		args = append(args, filter.AccountID)
	}
	if !filter.DateFrom.IsZero() {
		clauses = append(clauses, "stat_date >= ?")
		args = append(args, filter.DateFrom)
	}
	if !filter.DateTo.IsZero() {
		clauses = append(clauses, "stat_date <= ?")
		args = append(args, filter.DateTo)
	}
	if filter.Country != "" {
		clauses = append(clauses, "country = ?")
		args = append(args, filter.Country)
	}
	if filter.OS != "" {
		clauses = append(clauses, "os = ?")
		args = append(args, filter.OS)
	}
	if filter.PlatformCampaignID != "" {
		clauses = append(clauses, "platform_campaign_id = ?")
		args = append(args, filter.PlatformCampaignID)
	}
	if filter.PlatformAdGroupID != "" {
		clauses = append(clauses, "platform_ad_group_id = ?")
		args = append(args, filter.PlatformAdGroupID)
	}
	if filter.PlatformAdID != "" {
		clauses = append(clauses, "platform_ad_id = ?")
		args = append(args, filter.PlatformAdID)
	}

	query := `
		SELECT platform, platform_account_id, platform_campaign_id, platform_ad_group_id, platform_ad_id,
			stat_date, country, os, placement, creative_id, creative_type, optimization_goal, bid_type, targeting,
			installs, activations, registrations, tutorial_completions, role_creations, level_x_users, purchasers, purchase_count,
			CAST(first_purchase_amount AS CHAR), CAST(revenue_d1 AS CHAR), CAST(revenue_d7 AS CHAR), CAST(revenue_d30 AS CHAR),
			CAST(ad_revenue AS CHAR), CAST(total_revenue AS CHAR), CAST(retention_d1 AS CHAR), CAST(retention_d3 AS CHAR),
			CAST(retention_d7 AS CHAR), CAST(retention_d30 AS CHAR), CAST(ltv_d7 AS CHAR), CAST(ltv_d30 AS CHAR),
			avg_online_duration_seconds, CAST(task_completion_rate AS CHAR), CAST(high_value_payer_ratio AS CHAR), raw_payload
		FROM bi_game_kpis
	`
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY stat_date DESC, platform, platform_account_id, platform_campaign_id, platform_ad_group_id, platform_ad_id"
	limit := filter.Limit
	if limit <= 0 {
		limit = 500
	}
	query += " LIMIT ?"
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]biquerydomain.GameKPIRecord, 0, limit)
	for rows.Next() {
		var item biquerydomain.GameKPIRecord
		var platform string
		var rawPayload sql.NullString
		if err := rows.Scan(
			&platform,
			&item.PlatformAccountID,
			&item.PlatformCampaignID,
			&item.PlatformAdGroupID,
			&item.PlatformAdID,
			&item.StatDate,
			&item.Country,
			&item.OS,
			&item.Placement,
			&item.CreativeID,
			&item.CreativeType,
			&item.OptimizationGoal,
			&item.BidType,
			&item.Targeting,
			&item.Installs,
			&item.Activations,
			&item.Registrations,
			&item.TutorialCompletions,
			&item.RoleCreations,
			&item.LevelXUsers,
			&item.Purchasers,
			&item.PurchaseCount,
			&item.FirstPurchaseAmount,
			&item.RevenueD1,
			&item.RevenueD7,
			&item.RevenueD30,
			&item.AdRevenue,
			&item.TotalRevenue,
			&item.RetentionD1,
			&item.RetentionD3,
			&item.RetentionD7,
			&item.RetentionD30,
			&item.LTVD7,
			&item.LTVD30,
			&item.AvgOnlineDurationSeconds,
			&item.TaskCompletionRate,
			&item.HighValuePayerRatio,
			&rawPayload,
		); err != nil {
			return nil, err
		}
		item.Platform = rootdomain.Platform(platform)
		if rawPayload.Valid {
			item.RawPayload = []byte(rawPayload.String)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) CountRecords(ctx context.Context, table string) (int64, error) {
	row := r.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", table))
	var count int64
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func decimalOrZero(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return "0"
	}
	return raw
}

func nullJSON(raw []byte) any {
	if len(raw) == 0 {
		return nil
	}
	return string(raw)
}
