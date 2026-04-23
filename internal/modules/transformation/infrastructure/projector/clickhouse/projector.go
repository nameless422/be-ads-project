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
			entity_level String,
			entity_id String,
			stat_date Date,
			impressions Int64,
			clicks Int64,
			spend Decimal(18, 4),
			ctr Decimal(18, 4),
			cpc Decimal(18, 4),
			cpm Decimal(18, 4),
			conversions Decimal(18, 4),
			reach Int64,
			raw_payload String,
			ingested_at DateTime
		) ENGINE = ReplacingMergeTree(ingested_at)
		ORDER BY (platform, platform_account_id, stat_date, entity_level, entity_id)`, p.database),
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
		return nil
	}
	for _, item := range batch.Payload.Insights {
		_, err := p.db.ExecContext(ctx, `
			INSERT INTO `+p.database+`.olap_insights (
				platform, platform_account_id, entity_level, entity_id, stat_date,
				impressions, clicks, spend, ctr, cpc, cpm, conversions, reach, raw_payload, ingested_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			item.Platform,
			item.PlatformAccountID,
			item.EntityLevel,
			item.EntityID,
			item.StatDate,
			item.Impressions,
			item.Clicks,
			decimalString(item.Spend),
			decimalString(item.CTR),
			decimalString(item.CPC),
			decimalString(item.CPM),
			decimalString(item.Conversions),
			item.Reach,
			string(item.RawPayload),
			batch.NormalizedAt.UTC(),
		)
		if err != nil {
			return fmt.Errorf("insert insight %s/%s: %w", item.EntityLevel, item.EntityID, err)
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
