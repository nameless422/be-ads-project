package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	transformationdomain "be_ads_project/internal/modules/transformation/domain"
	rootdomain "be_ads_project/internal/shared/ads"
)

type Projector struct {
	db *sql.DB
}

func NewProjector(db *sql.DB) *Projector {
	return &Projector{db: db}
}

func (p *Projector) Migrate(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS oltp_accounts (
			platform_account_id VARCHAR(64) PRIMARY KEY,
			platform VARCHAR(32) NOT NULL,
			account_id VARCHAR(64) NOT NULL,
			account_name VARCHAR(255) NOT NULL,
			status VARCHAR(64) NOT NULL,
			timezone VARCHAR(64) NOT NULL,
			currency VARCHAR(16) NOT NULL,
			raw_payload LONGBLOB NULL,
			ingested_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			UNIQUE KEY uk_accounts_platform_account_id (platform, account_id)
		)`,
		`CREATE TABLE IF NOT EXISTS oltp_campaigns (
			platform_account_id VARCHAR(64) NOT NULL,
			platform_campaign_id VARCHAR(128) NOT NULL,
			platform VARCHAR(32) NOT NULL,
			account_id VARCHAR(64) NOT NULL,
			campaign_name VARCHAR(255) NOT NULL,
			status VARCHAR(64) NOT NULL,
			objective VARCHAR(128) NOT NULL,
			buying_type VARCHAR(128) NOT NULL,
			bidding_strategy VARCHAR(128) NOT NULL,
			daily_budget VARCHAR(64) NOT NULL,
			lifetime_budget VARCHAR(64) NOT NULL,
			currency VARCHAR(16) NOT NULL,
			start_time DATETIME NULL,
			end_time DATETIME NULL,
			source_updated_at DATETIME NULL,
			raw_payload LONGBLOB NULL,
			ingested_at DATETIME NOT NULL,
			PRIMARY KEY (platform_account_id, platform_campaign_id)
		)`,
		`CREATE TABLE IF NOT EXISTS oltp_ad_groups (
			platform_account_id VARCHAR(64) NOT NULL,
			platform_ad_group_id VARCHAR(128) NOT NULL,
			platform VARCHAR(32) NOT NULL,
			account_id VARCHAR(64) NOT NULL,
			platform_parent_id VARCHAR(128) NOT NULL,
			ad_group_name VARCHAR(255) NOT NULL,
			status VARCHAR(64) NOT NULL,
			bid_strategy VARCHAR(128) NOT NULL,
			daily_budget VARCHAR(64) NOT NULL,
			start_time DATETIME NULL,
			end_time DATETIME NULL,
			source_updated_at DATETIME NULL,
			raw_payload LONGBLOB NULL,
			ingested_at DATETIME NOT NULL,
			PRIMARY KEY (platform_account_id, platform_ad_group_id)
		)`,
		`CREATE TABLE IF NOT EXISTS oltp_ads (
			platform_account_id VARCHAR(64) NOT NULL,
			platform_ad_id VARCHAR(128) NOT NULL,
			platform VARCHAR(32) NOT NULL,
			account_id VARCHAR(64) NOT NULL,
			platform_parent_id VARCHAR(128) NOT NULL,
			ad_name VARCHAR(255) NOT NULL,
			status VARCHAR(64) NOT NULL,
			creative_id VARCHAR(128) NOT NULL,
			creative_name VARCHAR(255) NOT NULL,
			source_updated_at DATETIME NULL,
			raw_payload LONGBLOB NULL,
			ingested_at DATETIME NOT NULL,
			PRIMARY KEY (platform_account_id, platform_ad_id)
		)`,
	}

	for _, stmt := range statements {
		if _, err := p.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	if err := p.addColumnIfMissing(ctx, "oltp_campaigns", "bidding_strategy", "ALTER TABLE oltp_campaigns ADD COLUMN bidding_strategy VARCHAR(128) NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	return nil
}

func (p *Projector) addColumnIfMissing(ctx context.Context, tableName, columnName, alterSQL string) error {
	var count int
	if err := p.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM information_schema.columns
		WHERE table_schema = DATABASE()
		  AND table_name = ?
		  AND column_name = ?
	`, tableName, columnName).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	_, err := p.db.ExecContext(ctx, alterSQL)
	return err
}

func (p *Projector) Project(ctx context.Context, batch *transformationdomain.NormalizedBatch) error {
	now := batch.NormalizedAt.UTC()
	account := batch.Collected.Target.Bundle.Account

	if err := p.upsertAccountDimension(ctx, account, now); err != nil {
		return err
	}
	if err := p.upsertCampaigns(ctx, account.AccountID, batch.Payload.Campaigns, now); err != nil {
		return err
	}
	if err := p.upsertAdGroups(ctx, account.AccountID, batch.Payload.AdGroups, now); err != nil {
		return err
	}
	if err := p.upsertAds(ctx, account.AccountID, batch.Payload.Ads, now); err != nil {
		return err
	}
	return nil
}

func (p *Projector) upsertAccountDimension(ctx context.Context, account rootdomain.PlatformAccount, now time.Time) error {
	_, err := p.db.ExecContext(ctx, `
		INSERT INTO oltp_accounts (
			platform_account_id, platform, account_id, account_name, status, timezone, currency, ingested_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			platform = VALUES(platform),
			account_id = VALUES(account_id),
			account_name = VALUES(account_name),
			status = VALUES(status),
			timezone = VALUES(timezone),
			currency = VALUES(currency),
			ingested_at = VALUES(ingested_at),
			updated_at = VALUES(updated_at)
	`, account.ID, account.Platform, account.AccountID, account.AccountName, account.Status, account.Timezone, account.Currency, now, now)
	return err
}

func (p *Projector) upsertCampaigns(ctx context.Context, accountID string, items []rootdomain.StandardCampaign, now time.Time) error {
	for _, item := range items {
		_, err := p.db.ExecContext(ctx, `
			INSERT INTO oltp_campaigns (
				platform_account_id, platform_campaign_id, platform, account_id, campaign_name, status, objective, buying_type,
				bidding_strategy, daily_budget, lifetime_budget, currency, start_time, end_time, source_updated_at, raw_payload, ingested_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
				platform = VALUES(platform),
				account_id = VALUES(account_id),
				campaign_name = VALUES(campaign_name),
				status = VALUES(status),
				objective = VALUES(objective),
				buying_type = VALUES(buying_type),
				bidding_strategy = VALUES(bidding_strategy),
				daily_budget = VALUES(daily_budget),
				lifetime_budget = VALUES(lifetime_budget),
				currency = VALUES(currency),
				start_time = VALUES(start_time),
				end_time = VALUES(end_time),
				source_updated_at = VALUES(source_updated_at),
				raw_payload = VALUES(raw_payload),
				ingested_at = VALUES(ingested_at)
		`, item.PlatformAccountID, item.PlatformCampaignID, item.Platform, accountID, item.CampaignName, item.Status, item.Objective, item.BuyingType, item.BiddingStrategy, item.DailyBudget, item.LifetimeBudget, item.Currency, item.StartTime, item.EndTime, item.UpdatedAt, item.RawPayload, now)
		if err != nil {
			return fmt.Errorf("upsert campaign %s: %w", item.PlatformCampaignID, err)
		}
	}
	return nil
}

func (p *Projector) upsertAdGroups(ctx context.Context, accountID string, items []rootdomain.StandardAdGroup, now time.Time) error {
	for _, item := range items {
		_, err := p.db.ExecContext(ctx, `
			INSERT INTO oltp_ad_groups (
				platform_account_id, platform_ad_group_id, platform, account_id, platform_parent_id, ad_group_name, status, bid_strategy,
				daily_budget, start_time, end_time, source_updated_at, raw_payload, ingested_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
				platform = VALUES(platform),
				account_id = VALUES(account_id),
				platform_parent_id = VALUES(platform_parent_id),
				ad_group_name = VALUES(ad_group_name),
				status = VALUES(status),
				bid_strategy = VALUES(bid_strategy),
				daily_budget = VALUES(daily_budget),
				start_time = VALUES(start_time),
				end_time = VALUES(end_time),
				source_updated_at = VALUES(source_updated_at),
				raw_payload = VALUES(raw_payload),
				ingested_at = VALUES(ingested_at)
		`, item.PlatformAccountID, item.PlatformAdGroupID, item.Platform, accountID, item.PlatformParentID, item.AdGroupName, item.Status, item.BidStrategy, item.DailyBudget, item.StartTime, item.EndTime, item.UpdatedAt, item.RawPayload, now)
		if err != nil {
			return fmt.Errorf("upsert ad_group %s: %w", item.PlatformAdGroupID, err)
		}
	}
	return nil
}

func (p *Projector) upsertAds(ctx context.Context, accountID string, items []rootdomain.StandardAd, now time.Time) error {
	for _, item := range items {
		_, err := p.db.ExecContext(ctx, `
			INSERT INTO oltp_ads (
				platform_account_id, platform_ad_id, platform, account_id, platform_parent_id, ad_name, status, creative_id,
				creative_name, source_updated_at, raw_payload, ingested_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
				platform = VALUES(platform),
				account_id = VALUES(account_id),
				platform_parent_id = VALUES(platform_parent_id),
				ad_name = VALUES(ad_name),
				status = VALUES(status),
				creative_id = VALUES(creative_id),
				creative_name = VALUES(creative_name),
				source_updated_at = VALUES(source_updated_at),
				raw_payload = VALUES(raw_payload),
				ingested_at = VALUES(ingested_at)
		`, item.PlatformAccountID, item.PlatformAdID, item.Platform, accountID, item.PlatformParentID, item.AdName, item.Status, item.CreativeID, item.CreativeName, item.UpdatedAt, item.RawPayload, now)
		if err != nil {
			return fmt.Errorf("upsert ad %s: %w", item.PlatformAdID, err)
		}
	}
	return nil
}
