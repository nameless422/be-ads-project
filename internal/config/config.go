package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPPort        string
	SyncInterval    time.Duration
	OutboxTransport string
	MySQL           MySQLConfig
	RawMySQL        MySQLConfig
	ServingMySQL    MySQLConfig
	ClickHouse      ClickHouseConfig
	NATS            NATSConfig
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
	URL string
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
		HTTPPort:        envOrDefault("BE_HTTP_PORT", "8080"),
		SyncInterval:    durationEnv("BE_SYNC_INTERVAL", 10*time.Second),
		OutboxTransport: envOrDefault("BE_OUTBOX_TRANSPORT", "relay"),
		MySQL:           servingMySQL,
		RawMySQL:        rawMySQL,
		ServingMySQL:    servingMySQL,
		ClickHouse: ClickHouseConfig{
			Host:     envOrDefault("BE_CLICKHOUSE_HOST", "127.0.0.1"),
			Port:     intEnv("BE_CLICKHOUSE_PORT", 9000),
			Database: envOrDefault("BE_CLICKHOUSE_DATABASE", "be_ads"),
			Username: envOrDefault("BE_CLICKHOUSE_USER", "be_ads"),
			Password: envOrDefault("BE_CLICKHOUSE_PASSWORD", "be_ads"),
		},
		NATS: NATSConfig{
			URL: envOrDefault("BE_NATS_URL", "nats://127.0.0.1:4222"),
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
