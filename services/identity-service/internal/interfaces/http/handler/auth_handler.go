package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rent-a-girlfriend/identity-service/internal/application/command"
	"github.com/rent-a-girlfriend/identity-service/internal/application/query"
	"github.com/rent-a-girlfriend/identity-service/internal/interfaces/http/dto"
)

// AuthHandler handles authentication-related HTTP requests.
type AuthHandler struct {
	initGoogleAuth *command.InitGoogleAuthHandler
	loginGoogle    *command.LoginGoogleHandler
	refreshToken   *command.RefreshTokenHandler
	logout         *command.LogoutHandler
	getJWKS        *query.GetJWKSHandler
	requestUpgrade *command.RequestCompanionUpgradeHandler
	mockLogin      *command.MockLoginHandler
}

// NewAuthHandler creates a new auth handler.
func NewAuthHandler(
	initGoogleAuth *command.InitGoogleAuthHandler,
	loginGoogle *command.LoginGoogleHandler,
	refreshToken *command.RefreshTokenHandler,
	logout *command.LogoutHandler,
	getJWKS *query.GetJWKSHandler,
	requestUpgrade *command.RequestCompanionUpgradeHandler,
	mockLogin *command.MockLoginHandler,
) *AuthHandler {
	return &AuthHandler{
		initGoogleAuth: initGoogleAuth,
		loginGoogle:    loginGoogle,
		refreshToken:   refreshToken,
		logout:         logout,
		getJWKS:        getJWKS,
		requestUpgrade: requestUpgrade,
		mockLogin:      mockLogin,
	}
}

// InitGoogleAuth generates PKCE params and returns Google auth URL.
func (h *AuthHandler) InitGoogleAuth(c *gin.Context) {
	result, err := h.initGoogleAuth.Handle(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, dto.InitAuthResponse{
		AuthURL:       result.AuthURL,
		State:         result.State,
		CodeChallenge: result.CodeChallenge,
	})
}

// LoginGoogle handles the Google OAuth callback.
func (h *AuthHandler) LoginGoogle(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing code or state"})
		return
	}

	tokenPair, err := h.loginGoogle.Handle(c.Request.Context(), command.LoginGoogleCommand{
		Code:  code,
		State: state,
	})
	if err != nil {
		c.JSON(mapDomainErrorToHTTP(err), gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.TokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
	})
}

// RefreshToken handles token refresh with rotation.
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tokenPair, err := h.refreshToken.Handle(c.Request.Context(), command.RefreshTokenCommand{
		RefreshToken: req.RefreshToken,
	})
	if err != nil {
		c.JSON(mapDomainErrorToHTTP(err), gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.TokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
	})
}

// Logout revokes the refresh token.
func (h *AuthHandler) Logout(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_ = h.logout.Handle(c.Request.Context(), command.LogoutCommand{
		RefreshToken: req.RefreshToken,
	})

	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

// GetJWKS returns the public keys in JWKS format.
func (h *AuthHandler) GetJWKS(c *gin.Context) {
	jwks, err := h.getJWKS.Handle()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, jwks)
}

// RequestUpgrade creates a companion upgrade request.
func (h *AuthHandler) RequestUpgrade(c *gin.Context) {
	var req dto.RequestUpgradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// UserID from JWT claims (set by Istio/middleware)
	userID := c.GetHeader("X-User-Id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user identity"})
		return
	}

	err := h.requestUpgrade.Handle(c.Request.Context(), command.RequestCompanionUpgradeCommand{
		UserID: userID,
		Reason: req.Reason,
	})
	if err != nil {
		c.JSON(mapDomainErrorToHTTP(err), gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "upgrade request submitted"})
}

// MockLogin bypasses Google OAuth and generates a token pair directly for testing.
// This is extremely dangerous and MUST NOT be exposed in production.
func (h *AuthHandler) MockLogin(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		GoogleID string `json:"google_id" binding:"required"`
		Role     string `json:"role"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tokenPair, err := h.mockLogin.Handle(c.Request.Context(), command.MockLoginCommand{
		Email:    req.Email,
		GoogleID: req.GoogleID,
		Role:     req.Role,
	})
	if err != nil {
		c.JSON(mapDomainErrorToHTTP(err), gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.TokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
	})
}
