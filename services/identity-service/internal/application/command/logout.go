package command

import (
	"context"

	"github.com/rent-a-girlfriend/identity-service/internal/application/port"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/vo"
)

// LogoutCommand contains the token to revoke.
type LogoutCommand struct {
	RefreshToken string
}

// LogoutHandler revokes the current refresh token.
type LogoutHandler struct {
	tokenService port.TokenService
}

// NewLogoutHandler creates a new handler.
func NewLogoutHandler(tokenService port.TokenService) *LogoutHandler {
	return &LogoutHandler{tokenService: tokenService}
}

// Handle revokes the given refresh token.
func (h *LogoutHandler) Handle(ctx context.Context, cmd LogoutCommand) error {
	claims, err := h.tokenService.ValidateRefreshToken(cmd.RefreshToken)
	if err != nil {
		// Token is already invalid or expired — treat as success
		return nil
	}
	_ = vo.UserID{} // avoid unused import (claims used below)
	return h.tokenService.RevokeRefreshToken(claims.TokenID)
}
