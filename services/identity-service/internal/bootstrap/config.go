package bootstrap

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
	DatabaseURL string // Full connection string (e.g., from Neon)
}

// DSN returns the PostgreSQL connection string.
func (d DatabaseConfig) DSN() string {
	return d.DatabaseURL
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

	configDir := getEnv("CONFIG_DIR", "/etc/config")
	secretDir := getEnv("SECRETS_DIR", "/etc/secrets")
	configFiles := loadConfigDir(configDir)
	secretFiles := loadConfigDir(secretDir)

	accessTTL, _ := strconv.Atoi(getConfigValue(configFiles, secretFiles, "JWT_ACCESS_TTL_MINUTES", "30"))
	refreshTTL, _ := strconv.Atoi(getConfigValue(configFiles, secretFiles, "JWT_REFRESH_TTL_DAYS", "7"))
	outboxInterval, _ := strconv.Atoi(getConfigValue(configFiles, secretFiles, "OUTBOX_POLLING_INTERVAL_MS", "500"))
	outboxBatchSize, _ := strconv.Atoi(getConfigValue(configFiles, secretFiles, "OUTBOX_BATCH_SIZE", "50"))

	return &Config{
		Server: ServerConfig{
			Port:     getConfigValue(configFiles, secretFiles, "SERVER_PORT", "8081"),
			GRPCPort: getConfigValue(configFiles, secretFiles, "GRPC_PORT", "50051"),
			Mode:     getConfigValue(configFiles, secretFiles, "GIN_MODE", "debug"),
		},
		Database: DatabaseConfig{
			DatabaseURL: getConfigValue(configFiles, secretFiles, "DB_URL", getConfigValue(configFiles, secretFiles, "DATABASE_URL", "")),
		},
		OAuth: OAuthConfig{
			GoogleClientID:     getConfigValue(configFiles, secretFiles, "GOOGLE_CLIENT_ID", ""),
			GoogleClientSecret: getConfigValue(configFiles, secretFiles, "GOOGLE_CLIENT_SECRET", ""),
			GoogleRedirectURI:  getConfigValue(configFiles, secretFiles, "GOOGLE_REDIRECT_URI", "http://localhost:8081/api/v1/auth/google/callback"),
		},
		JWT: JWTConfig{
			AccessTokenTTL:  time.Duration(accessTTL) * time.Minute,
			RefreshTokenTTL: time.Duration(refreshTTL) * 24 * time.Hour,
			Issuer:          getConfigValue(configFiles, secretFiles, "JWT_ISSUER", "rent-a-girlfriend-identity"),
		},
		Kafka: KafkaConfig{
			Brokers:       getConfigValue(configFiles, secretFiles, "KAFKA_BROKERS", "localhost:9092"),
			TopicIdentity: getConfigValue(configFiles, secretFiles, "KAFKA_TOPIC_IDENTITY", "identity.events"),
		},
		Outbox: OutboxConfig{
			PollingInterval: time.Duration(outboxInterval) * time.Millisecond,
			BatchSize:       outboxBatchSize,
		},
		Redis: RedisConfig{
			URL: getConfigValue(configFiles, secretFiles, "REDIS_URL", ""),
		},
	}
}

func getConfigValue(configFiles, secretFiles map[string]string, key, defaultValue string) string {
	if value, ok := configFiles[key]; ok && value != "" {
		return value
	}
	if value, ok := secretFiles[key]; ok && value != "" {
		return value
	}
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return defaultValue
}

func loadConfigDir(dir string) map[string]string {
	configs := make(map[string]string)
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return configs
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return configs
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		configs[entry.Name()] = strings.TrimSpace(string(content))
	}

	return configs
}

func getEnv(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}
