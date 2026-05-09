package command

import (
	"context"

	domainerr "github.com/rent-a-girlfriend/identity-service/internal/domain/errors"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/repository"
	"github.com/rent-a-girlfriend/identity-service/internal/application/port"
)

// RefreshTokenCommand contains the refresh token to rotate.
type RefreshTokenCommand struct {
	RefreshToken string
}

// RefreshTokenHandler validates and rotates a refresh token.
type RefreshTokenHandler struct {
	tokenService port.TokenService
	accountRepo  repository.UserAccountRepository
}

// NewRefreshTokenHandler creates a new handler.
func NewRefreshTokenHandler(
	tokenService port.TokenService,
	accountRepo repository.UserAccountRepository,
) *RefreshTokenHandler {
	return &RefreshTokenHandler{
		tokenService: tokenService,
		accountRepo:  accountRepo,
	}
}

// Handle validates the refresh token, detects reuse, and rotates.
func (h *RefreshTokenHandler) Handle(ctx context.Context, cmd RefreshTokenCommand) (*port.TokenPair, error) {
	// 1. Validate refresh token
	claims, err := h.tokenService.ValidateRefreshToken(cmd.RefreshToken)
	if err != nil {
		// If this is a reuse of a revoked token, revoke entire family
		if err == domainerr.ErrRefreshTokenReuse {
			_ = h.tokenService.RevokeAllUserTokens(claims.UserID)
		}
		return nil, err
	}

	// 2. Find user account and check [INV-ID01]
	account, err := h.accountRepo.FindByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}
	if err := account.CheckLoginAllowed(); err != nil {
		return nil, err
	}

	// 3. Rotate: revoke old token, issue new pair
	tokenPair, err := h.tokenService.RotateRefreshToken(claims, account)
	if err != nil {
		return nil, err
	}

	return tokenPair, nil
}
