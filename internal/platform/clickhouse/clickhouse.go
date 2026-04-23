package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"be_ads_project/internal/config"

	_ "github.com/ClickHouse/clickhouse-go/v2"
)

func Open(cfg config.ClickHouseConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf("clickhouse://%s:%d/%s", cfg.Host, cfg.Port, "default")
	if cfg.Username != "" {
		dsn = fmt.Sprintf("clickhouse://%s:%s@%s:%d/%s", cfg.Username, cfg.Password, cfg.Host, cfg.Port, "default")
	}

	db, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return nil, err
	}
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping clickhouse: %w", err)
	}
	return db, nil
}
