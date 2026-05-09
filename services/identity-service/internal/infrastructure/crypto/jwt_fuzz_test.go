package crypto

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/persistence"
)

func FuzzValidateRefreshToken(f *testing.F) {
	// Setup in-memory DB
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&persistence.RefreshTokenModel{})

	svc := NewJWTTokenService(db, nil, time.Minute, time.Hour, "test")

	f.Add("550e8400-e29b-41d4-a716-446655440000")
	f.Add("invalid-token")
	f.Add("")

	f.Fuzz(func(t *testing.T, token string) {
		// Just ensure it doesn't panic
		_, _ = svc.ValidateRefreshToken(token)
	})
}
