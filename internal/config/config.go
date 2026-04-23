package config

import (
	ads "be_ads_project/internal/shared/ads"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPPort           string
	SyncInterval       time.Duration
	OutboxTransport    string
	ShardCount         int
	WorkerID           string
	LeaseTTL           time.Duration
	HeartbeatInterval  time.Duration
	WorkerPlatforms    []ads.Platform
	CollectorRuntime   WorkerRuntimeConfig
	TransformerRuntime WorkerRuntimeConfig
	MySQL              MySQLConfig
	RawMySQL           MySQLConfig
	ServingMySQL       MySQLConfig
	ClickHouse         ClickHouseConfig
	NATS               NATSConfig
}

type MySQLConfig struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
}

type ClickHouseConfig struct {
	Host     string
	Port     int
	Database string
	Username string
	Password string
}

type NATSConfig struct {
	URL           string
	AckWait       time.Duration
	MaxDeliver    int
	MaxAckPending int
}

type WorkerRuntimeConfig struct {
	FetchBatch  int
	Concurrency int
}

func Load() Config {
	rawMySQL := MySQLConfig{
		Host:     envOrDefault("BE_RAW_MYSQL_HOST", "127.0.0.1"),
		Port:     intEnv("BE_RAW_MYSQL_PORT", 3307),
		Database: envOrDefault("BE_RAW_MYSQL_DATABASE", "be_ads_raw"),
		User:     envOrDefault("BE_RAW_MYSQL_USER", "be_ads"),
		Password: envOrDefault("BE_RAW_MYSQL_PASSWORD", "be_ads"),
	}
	servingMySQL := MySQLConfig{
		Host:     envOrDefault("BE_SERVING_MYSQL_HOST", envOrDefault("BE_MYSQL_HOST", "127.0.0.1")),
		Port:     intEnv("BE_SERVING_MYSQL_PORT", intEnv("BE_MYSQL_PORT", 3308)),
		Database: envOrDefault("BE_SERVING_MYSQL_DATABASE", envOrDefault("BE_MYSQL_DATABASE", "be_ads_serving")),
		User:     envOrDefault("BE_SERVING_MYSQL_USER", envOrDefault("BE_MYSQL_USER", "be_ads")),
		Password: envOrDefault("BE_SERVING_MYSQL_PASSWORD", envOrDefault("BE_MYSQL_PASSWORD", "be_ads")),
	}

	return Config{
		HTTPPort:          envOrDefault("BE_HTTP_PORT", "8080"),
		SyncInterval:      durationEnv("BE_SYNC_INTERVAL", 10*time.Second),
		OutboxTransport:   envOrDefault("BE_OUTBOX_TRANSPORT", "relay"),
		ShardCount:        intEnv("BE_SHARD_COUNT", 4),
		WorkerID:          envOrDefault("BE_WORKER_ID", hostNameOrDefault("worker-local")),
		LeaseTTL:          durationEnv("BE_LEASE_TTL", 30*time.Second),
		HeartbeatInterval: durationEnv("BE_HEARTBEAT_INTERVAL", 10*time.Second),
		WorkerPlatforms:   platformListEnv("BE_WORKER_PLATFORMS"),
		CollectorRuntime: WorkerRuntimeConfig{
			FetchBatch:  intEnv("BE_COLLECTOR_FETCH_BATCH", 10),
			Concurrency: intEnv("BE_COLLECTOR_CONCURRENCY", 4),
		},
		TransformerRuntime: WorkerRuntimeConfig{
			FetchBatch:  intEnv("BE_TRANSFORMER_FETCH_BATCH", 10),
			Concurrency: intEnv("BE_TRANSFORMER_CONCURRENCY", 4),
		},
		MySQL:        servingMySQL,
		RawMySQL:     rawMySQL,
		ServingMySQL: servingMySQL,
		ClickHouse: ClickHouseConfig{
			Host:     envOrDefault("BE_CLICKHOUSE_HOST", "127.0.0.1"),
			Port:     intEnv("BE_CLICKHOUSE_PORT", 9000),
			Database: envOrDefault("BE_CLICKHOUSE_DATABASE", "be_ads"),
			Username: envOrDefault("BE_CLICKHOUSE_USER", "be_ads"),
			Password: envOrDefault("BE_CLICKHOUSE_PASSWORD", "be_ads"),
		},
		NATS: NATSConfig{
			URL:           envOrDefault("BE_NATS_URL", "nats://127.0.0.1:4222"),
			AckWait:       durationEnv("BE_NATS_ACK_WAIT", 30*time.Second),
			MaxDeliver:    intEnv("BE_NATS_MAX_DELIVER", 5),
			MaxAckPending: intEnv("BE_NATS_MAX_ACK_PENDING", 128),
		},
	}
}

func (c MySQLConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&multiStatements=true&charset=utf8mb4", c.User, c.Password, c.Host, c.Port, c.Database)
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func intEnv(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func platformListEnv(key string) []ads.Platform {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	items := make([]ads.Platform, 0, len(parts))
	for _, part := range parts {
		raw := strings.TrimSpace(part)
		if raw == "" {
			continue
		}
		platform, err := ads.ParsePlatform(raw)
		if err != nil {
			continue
		}
		items = append(items, platform)
	}
	if len(items) == 0 {
		return nil
	}
	return items
}

func hostNameOrDefault(fallback string) string {
	host, err := os.Hostname()
	if err != nil {
		return fallback
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return fallback
	}
	return host
}
