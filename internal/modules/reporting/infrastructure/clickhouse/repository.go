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
		SELECT platform, platform_account_id, stat_date,
			sum(impressions) AS impressions,
			sum(clicks) AS clicks,
			sum(toDecimal64(spend, 4)) AS spend,
			sum(toDecimal64(conversions, 4)) AS conversions,
			sum(reach) AS reach
		FROM ` + r.database + `.olap_insights FINAL
	`
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " GROUP BY platform, platform_account_id, stat_date ORDER BY stat_date, platform, platform_account_id"

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
		if err := rows.Scan(&platform, &accountID, &statDate, &item.Impressions, &item.Clicks, &spend, &conversions, &item.Reach); err != nil {
			return nil, err
		}
		item.Platform = rootdomain.Platform(platform)
		item.PlatformAccountID = accountID
		item.StatDate = statDate
		item.Spend = spend
		item.Conversions = conversions
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
