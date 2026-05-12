package port

import (
	"github.com/rent-a-girlfriend/identity-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/vo"
)

// TokenPair contains the access and refresh tokens issued to a user.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64 // seconds until access token expires
}

// RefreshTokenClaims contains the claims extracted from a valid refresh token.
type RefreshTokenClaims struct {
	UserID   vo.UserID
	FamilyID string
	TokenID  string
}

// TokenService is the port for JWT token generation and validation.
type TokenService interface {
	// GenerateTokenPair creates a new access/refresh token pair for the given account.
	GenerateTokenPair(account *aggregate.UserAccount) (*TokenPair, error)

	// ValidateRefreshToken validates a refresh token and returns its claims.
	ValidateRefreshToken(token string) (*RefreshTokenClaims, error)

	// RevokeRefreshToken revokes a specific refresh token.
	RevokeRefreshToken(tokenID string) error

	// RevokeAllUserTokens revokes all refresh tokens for a user (reuse detection).
	RevokeAllUserTokens(userID vo.UserID) error

	// RotateRefreshToken revokes the old token and generates a new pair.
	RotateRefreshToken(claims *RefreshTokenClaims, account *aggregate.UserAccount) (*TokenPair, error)
}
