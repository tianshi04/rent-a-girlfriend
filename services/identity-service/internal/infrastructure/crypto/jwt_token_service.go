package crypto

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rent-a-girlfriend/identity-service/internal/application/port"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/aggregate"
	domainerr "github.com/rent-a-girlfriend/identity-service/internal/domain/errors"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/vo"
	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/persistence"
)

// JWTTokenService implements TokenService using RS256 JWT.
type JWTTokenService struct {
	db               *gorm.DB
	keyProvider      *RSAKeyProvider
	accessTokenTTL   time.Duration
	refreshTokenTTL  time.Duration
	issuer           string
}

// NewJWTTokenService creates a new JWT token service.
func NewJWTTokenService(
	db *gorm.DB,
	keyProvider *RSAKeyProvider,
	accessTokenTTL, refreshTokenTTL time.Duration,
	issuer string,
) *JWTTokenService {
	return &JWTTokenService{
		db:              db,
		keyProvider:     keyProvider,
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: refreshTokenTTL,
		issuer:          issuer,
	}
}

func (s *JWTTokenService) GenerateTokenPair(account *aggregate.UserAccount) (*port.TokenPair, error) {
	privKey, kid, err := s.keyProvider.GetActiveKey()
	if err != nil {
		return nil, err
	}

	now := time.Now()

	// Access Token
	accessClaims := jwt.MapClaims{
		"sub":    account.ID().String(),
		"email":  account.Email().String(),
		"role":   string(account.Role()),
		"status": string(account.Status()),
		"iss":    s.issuer,
		"iat":    now.Unix(),
		"exp":    now.Add(s.accessTokenTTL).Unix(),
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims)
	accessToken.Header["kid"] = kid
	accessToken.Header["typ"] = "JWT"
	accessTokenStr, err := accessToken.SignedString(privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Refresh Token — signed JWT, stored in DB by TokenID
	tokenID := uuid.New()
	familyID := uuid.New()
	
	refreshClaims := jwt.MapClaims{
		"sub": account.ID().String(),
		"jti": tokenID.String(),
		"fam": familyID.String(),
		"typ": "Refresh",
		"iss": s.issuer,
		"iat": now.Unix(),
		"exp": now.Add(s.refreshTokenTTL).Unix(),
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodRS256, refreshClaims)
	refreshToken.Header["kid"] = kid
	refreshToken.Header["typ"] = "JWT"
	refreshTokenStr, err := refreshToken.SignedString(privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	tokenHash := hashToken(refreshTokenStr)

	rtModel := persistence.RefreshTokenModel{
		ID:        tokenID,
		UserID:    account.ID().UUID(),
		TokenHash: tokenHash,
		FamilyID:  familyID,
		IsRevoked: false,
		ExpiresAt: now.Add(s.refreshTokenTTL),
		CreatedAt: now,
	}
	if err := s.db.Create(&rtModel).Error; err != nil {
		return nil, fmt.Errorf("failed to save refresh token: %w", err)
	}

	return &port.TokenPair{
		AccessToken:  accessTokenStr,
		RefreshToken: refreshTokenStr,
		ExpiresIn:    int64(s.accessTokenTTL.Seconds()),
	}, nil
}

func (s *JWTTokenService) ValidateRefreshToken(token string) (*port.RefreshTokenClaims, error) {
	// Parse and validate JWT signature
	parsedToken, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		kid, ok := t.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("missing kid in token header")
		}
		
		pubKey, err := s.keyProvider.GetPublicKey(kid)
		if err != nil {
			return nil, err
		}
		return pubKey, nil
	})

	if err != nil || !parsedToken.Valid {
		return nil, domainerr.ErrInvalidRefreshToken
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return nil, domainerr.ErrInvalidRefreshToken
	}

	tokenIDStr, _ := claims["jti"].(string)
	tokenID, err := uuid.Parse(tokenIDStr)
	if err != nil {
		return nil, domainerr.ErrInvalidRefreshToken
	}

	var model persistence.RefreshTokenModel
	if err := s.db.Where("id = ?", tokenID).First(&model).Error; err != nil {
		return nil, domainerr.ErrInvalidRefreshToken
	}

	// Check if token was revoked (reuse detection)
	if model.IsRevoked {
		// Revoke entire family — token reuse attack!
		s.db.Model(&persistence.RefreshTokenModel{}).
			Where("family_id = ?", model.FamilyID).
			Update("is_revoked", true)

		userID, _ := vo.ParseUserID(model.UserID.String())
		return &port.RefreshTokenClaims{
			UserID:   userID,
			FamilyID: model.FamilyID.String(),
			TokenID:  model.ID.String(),
		}, domainerr.ErrRefreshTokenReuse
	}

	// Check expiry
	if time.Now().After(model.ExpiresAt) {
		return nil, domainerr.ErrInvalidRefreshToken
	}

	// Check hash match
	if hashToken(token) != model.TokenHash {
		return nil, domainerr.ErrInvalidRefreshToken
	}

	userID, _ := vo.ParseUserID(model.UserID.String())
	return &port.RefreshTokenClaims{
		UserID:   userID,
		FamilyID: model.FamilyID.String(),
		TokenID:  model.ID.String(),
	}, nil
}

func (s *JWTTokenService) RevokeRefreshToken(tokenID string) error {
	return s.db.Model(&persistence.RefreshTokenModel{}).
		Where("id = ?", tokenID).
		Update("is_revoked", true).Error
}

func (s *JWTTokenService) RevokeAllUserTokens(userID vo.UserID) error {
	return s.db.Model(&persistence.RefreshTokenModel{}).
		Where("user_id = ?", userID.UUID()).
		Update("is_revoked", true).Error
}

func (s *JWTTokenService) RotateRefreshToken(claims *port.RefreshTokenClaims, account *aggregate.UserAccount) (*port.TokenPair, error) {
	// Revoke old token
	if err := s.RevokeRefreshToken(claims.TokenID); err != nil {
		return nil, err
	}

	privKey, kid, err := s.keyProvider.GetActiveKey()
	if err != nil {
		return nil, err
	}

	now := time.Now()

	// New Access Token
	accessClaims := jwt.MapClaims{
		"sub":    account.ID().String(),
		"email":  account.Email().String(),
		"role":   string(account.Role()),
		"status": string(account.Status()),
		"iss":    s.issuer,
		"iat":    now.Unix(),
		"exp":    now.Add(s.accessTokenTTL).Unix(),
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims)
	accessToken.Header["kid"] = kid
	accessTokenStr, err := accessToken.SignedString(privKey)
	if err != nil {
		return nil, err
	}

	// New Refresh Token in same family
	familyID, _ := uuid.Parse(claims.FamilyID)
	tokenID := uuid.New()

	refreshClaims := jwt.MapClaims{
		"sub": account.ID().String(),
		"jti": tokenID.String(),
		"fam": familyID.String(),
		"typ": "Refresh",
		"iss": s.issuer,
		"iat": now.Unix(),
		"exp": now.Add(s.refreshTokenTTL).Unix(),
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodRS256, refreshClaims)
	refreshToken.Header["kid"] = kid
	refreshToken.Header["typ"] = "JWT"
	refreshTokenStr, err := refreshToken.SignedString(privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	rtModel := persistence.RefreshTokenModel{
		ID:        tokenID,
		UserID:    account.ID().UUID(),
		TokenHash: hashToken(refreshTokenStr),
		FamilyID:  familyID,
		IsRevoked: false,
		ExpiresAt: now.Add(s.refreshTokenTTL),
		CreatedAt: now,
	}
	if err := s.db.Create(&rtModel).Error; err != nil {
		return nil, err
	}

	return &port.TokenPair{
		AccessToken:  accessTokenStr,
		RefreshToken: refreshTokenStr,
		ExpiresIn:    int64(s.accessTokenTTL.Seconds()),
	}, nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
