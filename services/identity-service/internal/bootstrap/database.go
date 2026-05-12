package bootstrap

import (
	"embed"
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// InitDatabase initializes the GORM database connection and runs SQL migrations.
func InitDatabase(cfg DatabaseConfig) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Run SQL migrations automatically
	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("[DB] Connected to PostgreSQL and migrations applied successfully")
	return db, nil
}

func runMigrations(db *gorm.DB) error {
	migrationPath := "migrations/000001_init_schema.up.sql"
	log.Printf("[DB] Running migrations from embedded FS: %s", migrationPath)

	sqlContent, err := migrationsFS.ReadFile(migrationPath)
	if err != nil {
		return fmt.Errorf("could not read embedded migration file: %w", err)
	}

	if err := db.Exec(string(sqlContent)).Error; err != nil {
		return fmt.Errorf("could not execute migration SQL: %w", err)
	}

	return nil
}
