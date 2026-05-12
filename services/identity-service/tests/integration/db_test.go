package integration

import (
	"testing"

	"github.com/rent-a-girlfriend/identity-service/internal/bootstrap"
)

func TestDatabaseConnection(t *testing.T) {
	// Skip if not in integration test mode or no DB available
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cfg := bootstrap.LoadConfig()
	db, err := bootstrap.InitDatabase(cfg.Database)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get sql.DB: %v", err)
	}
	defer sqlDB.Close()

	err = sqlDB.Ping()
	if err != nil {
		t.Fatalf("failed to ping database: %v", err)
	}
}
