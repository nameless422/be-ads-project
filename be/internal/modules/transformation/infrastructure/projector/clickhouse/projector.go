package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	transformationdomain "be_ads_project/internal/modules/transformation/domain"
)

type Projector struct {
	db       *sql.DB
	database string
}

func NewProjector(db *sql.DB, database string) *Projector {
	return &Projector{db: db, database: database}
}

func (p *Projector) Migrate(ctx context.Context) error {
	statements := []string{
		fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", p.database),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.olap_insights (
			platform String,
			platform_account_id String,
			platform_campaign_id String,
			entity_level String,
			entity_id String,
			platform_ad_group_id String,
			platform_ad_id String,
			stat_date Date,
			device String,
			network String,
			impressions Int64,
			clicks Int64,
			spend Decimal(18, 4),
			ctr Decimal(18, 4),
			cpc Decimal(18, 4),
			cpm Decimal(18, 4),
			conversions Decimal(18, 4),
			all_conversions Decimal(18, 4),
			conversions_value Decimal(18, 4),
			cost_per_conversion Decimal(18, 4),
			cost_per_all_conversions Decimal(18, 4),
			reach Int64,
			raw_payload String,
			ingested_at DateTime
		) ENGINE = ReplacingMergeTree(ingested_at)
		ORDER BY (platform, platform_account_id, stat_date, platform_campaign_id, entity_level, entity_id, device, network)`, p.database),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.olap_campaign_diagnostics (
			platform String,
			platform_account_id String,
			platform_campaign_id String,
			stat_date Date,
			search_impression_share Decimal(18, 4),
			search_top_impression_share Decimal(18, 4),
			search_absolute_top_impression_share Decimal(18, 4),
			raw_payload String,
			ingested_at DateTime
		) ENGINE = ReplacingMergeTree(ingested_at)
		ORDER BY (platform, platform_account_id, stat_date, platform_campaign_id)`, p.database),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.olap_search_terms (
			platform String,
			platform_account_id String,
			platform_campaign_id String,
			platform_ad_group_id String,
			search_term String,
			search_term_match_type String,
			stat_date Date,
			impressions Int64,
			clicks Int64,
			spend Decimal(18, 4),
			conversions Decimal(18, 4),
			conversions_value Decimal(18, 4),
			raw_payload String,
			ingested_at DateTime
		) ENGINE = ReplacingMergeTree(ingested_at)
		ORDER BY (platform, platform_account_id, stat_date, platform_campaign_id, platform_ad_group_id, search_term)`, p.database),
		fmt.Sprintf("ALTER TABLE %s.olap_insights ADD COLUMN IF NOT EXISTS platform_campaign_id String", p.database),
		fmt.Sprintf("ALTER TABLE %s.olap_insights ADD COLUMN IF NOT EXISTS platform_ad_group_id String", p.database),
		fmt.Sprintf("ALTER TABLE %s.olap_insights ADD COLUMN IF NOT EXISTS platform_ad_id String", p.database),
		fmt.Sprintf("ALTER TABLE %s.olap_insights ADD COLUMN IF NOT EXISTS device String", p.database),
		fmt.Sprintf("ALTER TABLE %s.olap_insights ADD COLUMN IF NOT EXISTS network String", p.database),
		fmt.Sprintf("ALTER TABLE %s.olap_insights ADD COLUMN IF NOT EXISTS all_conversions Decimal(18, 4)", p.database),
		fmt.Sprintf("ALTER TABLE %s.olap_insights ADD COLUMN IF NOT EXISTS conversions_value Decimal(18, 4)", p.database),
		fmt.Sprintf("ALTER TABLE %s.olap_insights ADD COLUMN IF NOT EXISTS cost_per_conversion Decimal(18, 4)", p.database),
		fmt.Sprintf("ALTER TABLE %s.olap_insights ADD COLUMN IF NOT EXISTS cost_per_all_conversions Decimal(18, 4)", p.database),
	}
	for _, stmt := range statements {
		if _, err := p.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func (p *Projector) Project(ctx context.Context, batch *transformationdomain.NormalizedBatch) error {
	if len(batch.Payload.Insights) == 0 {
		goto diagnostics
	}
	for _, item := range batch.Payload.Insights {
		_, err := p.db.ExecContext(ctx, `
			INSERT INTO `+p.database+`.olap_insights (
				platform, platform_account_id, platform_campaign_id, entity_level, entity_id, platform_ad_group_id, platform_ad_id, stat_date, device, network,
				impressions, clicks, spend, ctr, cpc, cpm, conversions, all_conversions, conversions_value,
				cost_per_conversion, cost_per_all_conversions, reach, raw_payload, ingested_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			item.Platform,
			item.PlatformAccountID,
			item.PlatformCampaignID,
			item.EntityLevel,
			item.EntityID,
			item.PlatformAdGroupID,
			item.PlatformAdID,
			item.StatDate,
			item.Device,
			item.Network,
			item.Impressions,
			item.Clicks,
			decimalString(item.Spend),
			decimalString(item.CTR),
			decimalString(item.CPC),
			decimalString(item.CPM),
			decimalString(item.Conversions),
			decimalString(item.AllConversions),
			decimalString(item.ConversionsValue),
			decimalString(item.CostPerConversion),
			decimalString(item.CostPerAllConversions),
			item.Reach,
			string(item.RawPayload),
			batch.NormalizedAt.UTC(),
		)
		if err != nil {
			return fmt.Errorf("insert insight %s/%s: %w", item.EntityLevel, item.EntityID, err)
		}
	}
diagnostics:
	for _, item := range batch.Payload.CampaignDiagnostics {
		_, err := p.db.ExecContext(ctx, `
			INSERT INTO `+p.database+`.olap_campaign_diagnostics (
				platform, platform_account_id, platform_campaign_id, stat_date,
				search_impression_share, search_top_impression_share, search_absolute_top_impression_share,
				raw_payload, ingested_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			item.Platform,
			item.PlatformAccountID,
			item.PlatformCampaignID,
			item.StatDate,
			decimalString(item.SearchImpressionShare),
			decimalString(item.SearchTopImpressionShare),
			decimalString(item.SearchAbsoluteTopImpressionShare),
			string(item.RawPayload),
			batch.NormalizedAt.UTC(),
		)
		if err != nil {
			return fmt.Errorf("insert campaign diagnostic %s: %w", item.PlatformCampaignID, err)
		}
	}
	for _, item := range batch.Payload.SearchTerms {
		_, err := p.db.ExecContext(ctx, `
			INSERT INTO `+p.database+`.olap_search_terms (
				platform, platform_account_id, platform_campaign_id, platform_ad_group_id,
				search_term, search_term_match_type, stat_date, impressions, clicks,
				spend, conversions, conversions_value, raw_payload, ingested_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			item.Platform,
			item.PlatformAccountID,
			item.PlatformCampaignID,
			item.PlatformAdGroupID,
			item.SearchTerm,
			item.SearchTermMatchType,
			item.StatDate,
			item.Impressions,
			item.Clicks,
			decimalString(item.Spend),
			decimalString(item.Conversions),
			decimalString(item.ConversionsValue),
			string(item.RawPayload),
			batch.NormalizedAt.UTC(),
		)
		if err != nil {
			return fmt.Errorf("insert search term %s: %w", item.SearchTerm, err)
		}
	}
	return nil
}

func decimalString(raw string) string {
	if raw == "" {
		return "0"
	}
	if _, err := strconv.ParseFloat(raw, 64); err != nil {
		return "0"
	}
	return raw
}
