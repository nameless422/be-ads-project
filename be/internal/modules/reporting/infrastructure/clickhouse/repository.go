package clickhouse

import (
	"context"
	"database/sql"
	"strings"
	"time"

	biquerydomain "be_ads_project/internal/modules/reporting/domain"
	rootdomain "be_ads_project/internal/shared/ads"
)

type Repository struct {
	db       *sql.DB
	database string
}

func NewRepository(db *sql.DB, database string) *Repository {
	return &Repository{db: db, database: database}
}

func (r *Repository) QueryInsightSummary(ctx context.Context, filter biquerydomain.InsightSummaryFilter) ([]biquerydomain.InsightSummaryRow, error) {
	args := make([]any, 0, 8)
	clauses := make([]string, 0, 4)

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

	query := `
		SELECT
			platform,
			platform_account_id,
			stat_date,
			impressions,
			clicks,
			spend,
			conversions,
			all_conversions,
			conversions_value,
			if(conversions = toDecimal64(0, 4), toDecimal64(0, 4), spend / conversions) AS cost_per_conversion,
			if(all_conversions = toDecimal64(0, 4), toDecimal64(0, 4), spend / all_conversions) AS cost_per_all_conversions,
			reach
		FROM (
			SELECT
				platform,
				platform_account_id,
				stat_date,
				sum(impressions) AS impressions,
				sum(clicks) AS clicks,
				sum(spend) AS spend,
				sum(conversions) AS conversions,
				sum(all_conversions) AS all_conversions,
				sum(conversions_value) AS conversions_value,
				sum(reach) AS reach
			FROM ` + r.database + `.olap_insights FINAL
	`
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += `
			GROUP BY platform, platform_account_id, stat_date
		)
		ORDER BY stat_date, platform, platform_account_id
	`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]biquerydomain.InsightSummaryRow, 0)
	for rows.Next() {
		var item biquerydomain.InsightSummaryRow
		var platform string
		var accountID string
		var statDate time.Time
		var spend string
		var conversions string
		var allConversions string
		var conversionsValue string
		var costPerConversion string
		var costPerAllConversions string
		if err := rows.Scan(
			&platform,
			&accountID,
			&statDate,
			&item.Impressions,
			&item.Clicks,
			&spend,
			&conversions,
			&allConversions,
			&conversionsValue,
			&costPerConversion,
			&costPerAllConversions,
			&item.Reach,
		); err != nil {
			return nil, err
		}
		item.Platform = rootdomain.Platform(platform)
		item.PlatformAccountID = accountID
		item.StatDate = statDate
		item.Spend = spend
		item.Conversions = conversions
		item.AllConversions = allConversions
		item.ConversionsValue = conversionsValue
		item.CostPerConversion = costPerConversion
		item.CostPerAllConversions = costPerAllConversions
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) QueryInsightDetails(ctx context.Context, filter biquerydomain.InsightDetailFilter) ([]biquerydomain.InsightDetailRow, error) {
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
	if filter.EntityLevel != "" {
		clauses = append(clauses, "entity_level = ?")
		args = append(args, filter.EntityLevel)
	}
	if filter.Device != "" {
		clauses = append(clauses, "device = ?")
		args = append(args, filter.Device)
	}
	if filter.Network != "" {
		clauses = append(clauses, "network = ?")
		args = append(args, filter.Network)
	}

	query := `
		SELECT platform, platform_account_id, platform_campaign_id, entity_level, entity_id, platform_ad_group_id, platform_ad_id,
			stat_date, device, network, impressions, clicks, toString(spend), toString(ctr), toString(cpc), toString(cpm),
			toString(conversions), toString(all_conversions), toString(conversions_value), toString(cost_per_conversion),
			toString(cost_per_all_conversions), reach
		FROM ` + r.database + `.olap_insights FINAL
	`
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY stat_date DESC, platform, platform_account_id, entity_level, entity_id"

	limit := filter.Limit
	if limit <= 0 {
		limit = 200
	}
	query += " LIMIT ?"
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]biquerydomain.InsightDetailRow, 0, limit)
	for rows.Next() {
		var item biquerydomain.InsightDetailRow
		var platform string
		var entityLevel string
		if err := rows.Scan(
			&platform,
			&item.PlatformAccountID,
			&item.PlatformCampaignID,
			&entityLevel,
			&item.EntityID,
			&item.PlatformAdGroupID,
			&item.PlatformAdID,
			&item.StatDate,
			&item.Device,
			&item.Network,
			&item.Impressions,
			&item.Clicks,
			&item.Spend,
			&item.CTR,
			&item.CPC,
			&item.CPM,
			&item.Conversions,
			&item.AllConversions,
			&item.ConversionsValue,
			&item.CostPerConversion,
			&item.CostPerAllConversions,
			&item.Reach,
		); err != nil {
			return nil, err
		}
		item.Platform = rootdomain.Platform(platform)
		item.EntityLevel = rootdomain.ObjectType(entityLevel)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) QueryCampaignDiagnostics(ctx context.Context, filter biquerydomain.CampaignDiagnosticFilter) ([]biquerydomain.CampaignDiagnosticRow, error) {
	args := make([]any, 0, 6)
	clauses := make([]string, 0, 4)

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

	query := `
		SELECT platform, platform_account_id, platform_campaign_id, stat_date,
			toString(search_impression_share), toString(search_top_impression_share), toString(search_absolute_top_impression_share)
		FROM ` + r.database + `.olap_campaign_diagnostics FINAL
	`
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY stat_date DESC, platform, platform_account_id, platform_campaign_id"
	limit := filter.Limit
	if limit <= 0 {
		limit = 200
	}
	query += " LIMIT ?"
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]biquerydomain.CampaignDiagnosticRow, 0, limit)
	for rows.Next() {
		var item biquerydomain.CampaignDiagnosticRow
		var platform string
		if err := rows.Scan(
			&platform,
			&item.PlatformAccountID,
			&item.PlatformCampaignID,
			&item.StatDate,
			&item.SearchImpressionShare,
			&item.SearchTopImpressionShare,
			&item.SearchAbsoluteTopImpressionShare,
		); err != nil {
			return nil, err
		}
		item.Platform = rootdomain.Platform(platform)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) QueryUAReportRows(ctx context.Context, filter biquerydomain.UAReportFilter) ([]biquerydomain.UAAdReportRow, error) {
	args := make([]any, 0, 16)
	clauses := make([]string, 0, 12)

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
	if filter.EntityLevel != "" {
		clauses = append(clauses, "entity_level = ?")
		args = append(args, filter.EntityLevel)
	}
	if filter.Device != "" {
		clauses = append(clauses, "device = ?")
		args = append(args, filter.Device)
	}
	if filter.Network != "" {
		clauses = append(clauses, "network = ?")
		args = append(args, filter.Network)
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
		SELECT
			platform,
			platform_account_id,
			platform_campaign_id,
			entity_level,
			entity_id,
			platform_ad_group_id,
			platform_ad_id,
			stat_date,
			device,
			network,
			impressions,
			clicks,
			toString(spend_num) AS spend,
			toString(if(impressions = 0, 0.0, toFloat64(clicks) / toFloat64(impressions) * 100.0)) AS ctr,
			toString(if(clicks = 0, 0.0, spend_num / toFloat64(clicks))) AS cpc,
			toString(if(impressions = 0, 0.0, spend_num / toFloat64(impressions) * 1000.0)) AS cpm,
			toString(conversions_num) AS conversions,
			toString(all_conversions_num) AS all_conversions,
			toString(conversions_value_num) AS conversions_value,
			toString(if(conversions_num = 0, 0.0, spend_num / conversions_num)) AS cost_per_conversion,
			toString(if(all_conversions_num = 0, 0.0, spend_num / all_conversions_num)) AS cost_per_all_conversions,
			reach,
			toString(if(reach = 0, 0.0, toFloat64(impressions) / toFloat64(reach))) AS frequency,
			toString(if(spend_num = 0, 0.0, conversions_value_num / spend_num)) AS roas
		FROM (
			SELECT
				platform,
				platform_account_id,
				platform_campaign_id,
				entity_level,
				entity_id,
				platform_ad_group_id,
				platform_ad_id,
				stat_date,
				device,
				network,
				sum(impressions) AS impressions,
				sum(clicks) AS clicks,
				sum(toFloat64(spend)) AS spend_num,
				sum(toFloat64(conversions)) AS conversions_num,
				sum(toFloat64(all_conversions)) AS all_conversions_num,
				sum(toFloat64(conversions_value)) AS conversions_value_num,
				sum(toInt64(reach)) AS reach
			FROM ` + r.database + `.olap_insights FINAL
	`
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += `
			GROUP BY platform, platform_account_id, platform_campaign_id, entity_level, entity_id, platform_ad_group_id, platform_ad_id, stat_date, device, network
		)
		ORDER BY stat_date DESC, platform, platform_account_id, platform_campaign_id, entity_level, entity_id
	`
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

	items := make([]biquerydomain.UAAdReportRow, 0, limit)
	for rows.Next() {
		var item biquerydomain.UAAdReportRow
		var platform string
		var entityLevel string
		if err := rows.Scan(
			&platform,
			&item.PlatformAccountID,
			&item.PlatformCampaignID,
			&entityLevel,
			&item.EntityID,
			&item.PlatformAdGroupID,
			&item.PlatformAdID,
			&item.StatDate,
			&item.Device,
			&item.Network,
			&item.Impressions,
			&item.Clicks,
			&item.Spend,
			&item.CTR,
			&item.CPC,
			&item.CPM,
			&item.Conversions,
			&item.AllConversions,
			&item.ConversionsValue,
			&item.CostPerConversion,
			&item.CostPerAllConversions,
			&item.Reach,
			&item.Frequency,
			&item.ROAS,
		); err != nil {
			return nil, err
		}
		item.Platform = rootdomain.Platform(platform)
		item.EntityLevel = rootdomain.ObjectType(entityLevel)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) QuerySearchTerms(ctx context.Context, filter biquerydomain.SearchTermFilter) ([]biquerydomain.SearchTermRow, error) {
	args := make([]any, 0, 8)
	clauses := make([]string, 0, 6)

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
	if filter.MatchType != "" {
		clauses = append(clauses, "search_term_match_type = ?")
		args = append(args, filter.MatchType)
	}
	if filter.SearchTermQuery != "" {
		clauses = append(clauses, "positionCaseInsensitiveUTF8(search_term, ?) > 0")
		args = append(args, filter.SearchTermQuery)
	}

	query := `
		SELECT platform, platform_account_id, platform_campaign_id, platform_ad_group_id, search_term, search_term_match_type,
			stat_date, impressions, clicks, toString(spend), toString(conversions), toString(conversions_value)
		FROM ` + r.database + `.olap_search_terms FINAL
	`
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY stat_date DESC, conversions_value DESC, clicks DESC, search_term"
	limit := filter.Limit
	if limit <= 0 {
		limit = 200
	}
	query += " LIMIT ?"
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]biquerydomain.SearchTermRow, 0, limit)
	for rows.Next() {
		var item biquerydomain.SearchTermRow
		var platform string
		if err := rows.Scan(
			&platform,
			&item.PlatformAccountID,
			&item.PlatformCampaignID,
			&item.PlatformAdGroupID,
			&item.SearchTerm,
			&item.SearchTermMatchType,
			&item.StatDate,
			&item.Impressions,
			&item.Clicks,
			&item.Spend,
			&item.Conversions,
			&item.ConversionsValue,
		); err != nil {
			return nil, err
		}
		item.Platform = rootdomain.Platform(platform)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

func (r *Repository) CountInsights(ctx context.Context) (int64, error) {
	row := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+r.database+".olap_insights")
	var count int64
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}
