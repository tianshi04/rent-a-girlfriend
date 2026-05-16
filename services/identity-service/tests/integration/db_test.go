package integration

import (
	"testing"

	"github.com/rent-a-girlfriend/identity-service/tests/testhelper"
)

func TestDatabaseConnection(t *testing.T) {
	db := testhelper.StartPostgresContainer(t)

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get sql.DB: %v", err)
	}

	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("failed to ping database: %v", err)
	}
}
