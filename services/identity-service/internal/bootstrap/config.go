package bootstrap

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	OAuth    OAuthConfig
	JWT      JWTConfig
	Kafka    KafkaConfig
	Outbox   OutboxConfig
	Redis    RedisConfig
}

type ServerConfig struct {
	Port     string
	GRPCPort string
	Mode     string // "debug", "release", "test"
}

type DatabaseConfig struct {
	Host        string
	Port        int
	User        string
	Password    string
	DBName      string
	SSLMode     string
	DatabaseURL string // Full connection string (e.g., from Neon)
}

// DSN returns the PostgreSQL connection string.
func (d DatabaseConfig) DSN() string {
	if d.DatabaseURL != "" {
		return d.DatabaseURL
	}
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode,
	)
}

type OAuthConfig struct {
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURI  string
}

type JWTConfig struct {
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	Issuer          string
}

type KafkaConfig struct {
	Brokers       string
	TopicIdentity string
}

type OutboxConfig struct {
	PollingInterval time.Duration
	BatchSize       int
}

type RedisConfig struct {
	URL string
}

// LoadConfig loads configuration from environment variables.
func LoadConfig() *Config {
	// Try to load .env file from current directory or parent directories
	// (Useful for tests running in subdirectories)
	err := godotenv.Load()
	if err != nil {
		err = godotenv.Load("../.env")
	}
	if err != nil {
		err = godotenv.Load("../../.env")
	}
	if err != nil {
		log.Println("[CONFIG] No .env file found in standard locations, relying on environment variables")
	}

	dbPort, _ := strconv.Atoi(getEnv("DB_PORT", "5432"))
	accessTTL, _ := strconv.Atoi(getEnv("JWT_ACCESS_TTL_MINUTES", "30"))
	refreshTTL, _ := strconv.Atoi(getEnv("JWT_REFRESH_TTL_DAYS", "7"))
	outboxInterval, _ := strconv.Atoi(getEnv("OUTBOX_POLLING_INTERVAL_MS", "500"))
	outboxBatchSize, _ := strconv.Atoi(getEnv("OUTBOX_BATCH_SIZE", "50"))

	return &Config{
		Server: ServerConfig{
			Port:     getEnv("SERVER_PORT", "8081"),
			GRPCPort: getEnv("GRPC_PORT", "50051"),
			Mode:     getEnv("GIN_MODE", "debug"),
		},
		Database: DatabaseConfig{
			Host:        getEnv("DB_HOST", "localhost"),
			Port:        dbPort,
			User:        getEnv("DB_USER", "postgres"),
			Password:    getEnv("DB_PASSWORD", "postgres"),
			DBName:      getEnv("DB_NAME", "identity_db"),
			SSLMode:     getEnv("DB_SSLMODE", "disable"),
			DatabaseURL: getEnv("DATABASE_URL", ""),
		},
		OAuth: OAuthConfig{
			GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
			GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
			GoogleRedirectURI:  getEnv("GOOGLE_REDIRECT_URI", "http://localhost:8081/api/v1/auth/google/callback"),
		},
		JWT: JWTConfig{
			AccessTokenTTL:  time.Duration(accessTTL) * time.Minute,
			RefreshTokenTTL: time.Duration(refreshTTL) * 24 * time.Hour,
			Issuer:          getEnv("JWT_ISSUER", "rent-a-girlfriend-identity"),
		},
		Kafka: KafkaConfig{
			Brokers:       getEnv("KAFKA_BROKERS", "localhost:9092"),
			TopicIdentity: getEnv("KAFKA_TOPIC_IDENTITY", "identity-events"),
		},
		Outbox: OutboxConfig{
			PollingInterval: time.Duration(outboxInterval) * time.Millisecond,
			BatchSize:       outboxBatchSize,
		},
		Redis: RedisConfig{
			URL: getEnv("REDIS_URL", ""),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}
