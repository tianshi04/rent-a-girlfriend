package bootstrap

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Kafka    KafkaConfig
	Outbox   OutboxConfig
	Worker   WorkerConfig
	Clients  ClientsConfig
}

type ClientsConfig struct {
	ProfileServiceAddr string
	FinanceServiceAddr string
}

type ServerConfig struct {
	Port     string
	GRPCPort string
	Mode     string // "debug", "release", "test"
}

type DatabaseConfig struct {
	URL      string
	Host     string
	Port     int64
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type KafkaConfig struct {
	Brokers          string
	TopicBooking     string
	TopicFinance     string
	TopicInteraction string
	TopicDispute     string
	SASLUsername     string
	SASLPassword     string
	SASLMechanism    string
	TLSEnabled       bool
}

type OutboxConfig struct {
	PollingInterval time.Duration
	BatchSize       int64
}

type WorkerConfig struct {
	AutoCompleteInterval time.Duration
	AutoCompleteBuffer   time.Duration
}

// DSN returns the PostgreSQL connection string.
func (d DatabaseConfig) DSN() string {
	if d.URL != "" {
		return d.URL
	}
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode,
	)
}

// LoadConfig loads configuration from environment variables.
func LoadConfig() *Config {
	dbPort, _ := strconv.ParseInt(getEnv("DB_PORT", "5432"), 10, 64)
	outboxInterval, _ := strconv.ParseInt(getEnv("OUTBOX_POLLING_INTERVAL_MS", "500"), 10, 64)
	outboxBatchSize, _ := strconv.ParseInt(getEnv("OUTBOX_BATCH_SIZE", "50"), 10, 64)
	autoCompleteIntervalMs, _ := strconv.ParseInt(getEnv("AUTO_COMPLETE_INTERVAL_MS", "10000"), 10, 64) // default 10s
	autoCompleteBufferHours, _ := strconv.ParseInt(getEnv("AUTO_COMPLETE_BUFFER_HOURS", "12"), 10, 64)  // default 12h

	return &Config{
		Server: ServerConfig{
			Port:     getEnv("SERVER_PORT", "8080"),
			GRPCPort: getEnv("GRPC_PORT", "50051"),
			Mode:     getEnv("GIN_MODE", "debug"),
		},
		Database: DatabaseConfig{
			URL:      getEnv("DATABASE_URL", ""),
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     dbPort,
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			DBName:   getEnv("DB_NAME", "booking_db"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Kafka: KafkaConfig{
			Brokers:          getEnv("KAFKA_BROKERS", "localhost:29091,localhost:29092,localhost:29093"),
			TopicBooking:     getEnv("KAFKA_TOPIC_BOOKING", "booking-events"),
			TopicFinance:     getEnv("KAFKA_TOPIC_FINANCE", "finance-events"),
			TopicInteraction: getEnv("KAFKA_TOPIC_INTERACTION", "interaction-events"),
			TopicDispute:     getEnv("KAFKA_TOPIC_DISPUTE", "dispute-events"),
			SASLUsername:     getEnv("KAFKA_SASL_USERNAME", ""),
			SASLPassword:     getEnv("KAFKA_SASL_PASSWORD", ""),
			SASLMechanism:    getEnv("KAFKA_SASL_MECHANISM", "PLAIN"),
			TLSEnabled:       getEnv("KAFKA_TLS_ENABLED", "false") == "true",
		},
		Outbox: OutboxConfig{
			PollingInterval: time.Duration(outboxInterval) * time.Millisecond,
			BatchSize:       outboxBatchSize,
		},
		Worker: WorkerConfig{
			AutoCompleteInterval: time.Duration(autoCompleteIntervalMs) * time.Millisecond,
			AutoCompleteBuffer:   time.Duration(autoCompleteBufferHours) * time.Hour,
		},
		Clients: ClientsConfig{
			ProfileServiceAddr: getEnv("PROFILE_SERVICE_ADDR", "localhost:50052"),
			FinanceServiceAddr: getEnv("FINANCE_SERVICE_ADDR", "localhost:50053"),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}
