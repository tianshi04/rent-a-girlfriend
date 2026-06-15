package bootstrap

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"

	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/persistence"
)

// InitDatabase initializes the GORM database connection and runs GORM AutoMigrate.
func InitDatabase(cfg DatabaseConfig) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Run GORM AutoMigrate
	if err := db.AutoMigrate(
		&persistence.UserAccountModel{},
		&persistence.RefreshTokenModel{},
		&persistence.SigningKeyModel{},
		&persistence.UpgradeRequestModel{},
		&persistence.SystemConfigModel{},
		&persistence.OutboxModel{},
		&persistence.PKCEVerifierModel{},
	); err != nil {
		return nil, fmt.Errorf("failed to run AutoMigrate: %w", err)
	}

	// Seed Initial Configuration Data
	seedConfigs := []persistence.SystemConfigModel{
		{
			Key:       "violation_lock_threshold",
			Value:     "3",
			UpdatedAt: time.Now(),
		},
		{
			Key:       "companion_upgrade_manual_approval",
			Value:     "true",
			UpdatedAt: time.Now(),
		},
	}

	for _, seed := range seedConfigs {
		err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}},
			DoNothing: true,
		}).Create(&seed).Error
		if err != nil {
			return nil, fmt.Errorf("failed to seed config key %s: %w", seed.Key, err)
		}
	}

	log.Println("[DB] Connected to PostgreSQL, GORM AutoMigrate and seeding applied successfully")
	return db, nil
}
