package crypto

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/rent-a-girlfriend/identity-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/vo"
	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/persistence"
)

func TestJWTTokenService_Lifecycle(t *testing.T) {
	// Setup in-memory DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	db.AutoMigrate(&persistence.RefreshTokenModel{}, &persistence.SigningKeyModel{})

	// Setup Key Provider
	kp := NewRSAKeyProvider(db)
	_ = kp.EnsureSigningKey()

	svc := NewJWTTokenService(db, kp, time.Minute, time.Hour, "test-issuer")

	email, _ := vo.NewEmail("test@example.com")
	acc := aggregate.NewUserAccount(email, "google-1", time.Now())

	// 1. Generate Token Pair
	pair, err := svc.GenerateTokenPair(acc)
	if err != nil {
		t.Fatalf("GenerateTokenPair failed: %v", err)
	}

	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Error("expected non-empty tokens")
	}

	// 2. Validate Refresh Token
	claims, err := svc.ValidateRefreshToken(pair.RefreshToken)
	if err != nil {
		t.Fatalf("ValidateRefreshToken failed: %v", err)
	}
	if claims.UserID.String() != acc.ID().String() {
		t.Errorf("expected UserID %s, got %s", acc.ID(), claims.UserID)
	}

	// 3. Rotate Token
	newPair, err := svc.RotateRefreshToken(claims, acc)
	if err != nil {
		t.Fatalf("RotateRefreshToken failed: %v", err)
	}

	if newPair.RefreshToken == pair.RefreshToken {
		t.Error("expected different refresh token after rotation")
	}

	// 4. Reuse Detection - Old token should be revoked
	_, err = svc.ValidateRefreshToken(pair.RefreshToken)
	if err == nil {
		t.Error("expected error when validating old token after rotation")
	}
}
