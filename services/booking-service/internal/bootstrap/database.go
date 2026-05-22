package bootstrap

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/rent-a-girlfriend/booking-service/internal/infrastructure/persistence"
)

// InitDatabase initializes the GORM database connection and runs auto-migration.
func InitDatabase(cfg DatabaseConfig) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		//Logger: logger.Default.LogMode(logger.Info),
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.AutoMigrate(
		&persistence.BookingModel{},
		&persistence.OutboxModel{},
		&persistence.BookingAcceptSagaModel{},
		&persistence.ProcessedEventModel{},
	); err != nil {
		return nil, fmt.Errorf("failed to auto-migrate: %w", err)
	}

	log.Println("[DB] Connected to PostgreSQL and migrations applied successfully")
	return db, nil
}
