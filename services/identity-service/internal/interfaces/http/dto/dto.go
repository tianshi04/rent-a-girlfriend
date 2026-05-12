package dto

// --- Auth DTOs ---

// LoginGoogleRequest is the request body for the Google OAuth callback.
type LoginGoogleRequest struct {
	Code  string `json:"code" binding:"required"`
	State string `json:"state" binding:"required"`
}

// TokenResponse is the response containing JWT tokens.
type TokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"`
}

// RefreshTokenRequest is the request body for token refresh.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// InitAuthResponse is the response for initializing Google OAuth.
type InitAuthResponse struct {
	AuthURL       string `json:"authUrl"`
	State         string `json:"state"`
	CodeChallenge string `json:"codeChallenge"`
}

// --- Upgrade DTOs ---

// RequestUpgradeRequest is the request body for companion upgrade.
type RequestUpgradeRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// UpgradeRequestResponse is the response for an upgrade request.
type UpgradeRequestResponse struct {
	ID           string  `json:"id"`
	UserID       string  `json:"userId"`
	Status       string  `json:"status"`
	Reason       string  `json:"reason"`
	RejectReason string  `json:"rejectReason,omitempty"`
	ReviewedBy   string  `json:"reviewedBy,omitempty"`
	ReviewedAt   *string `json:"reviewedAt,omitempty"`
	CreatedAt    string  `json:"createdAt"`
}

// --- Admin DTOs ---

// LockAccountRequest is the request body for locking an account.
type LockAccountRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// RejectUpgradeRequestBody is the request body for rejecting an upgrade.
type RejectUpgradeRequestBody struct {
	Reason string `json:"reason" binding:"required"`
}

// --- Account DTOs ---

// AccountResponse is the response for an account.
type AccountResponse struct {
	ID             string `json:"id"`
	Email          string `json:"email"`
	Role           string `json:"role"`
	Status         string `json:"status"`
	ViolationCount int    `json:"violationCount"`
	CreatedAt      string `json:"createdAt"`
}

// PaginatedResponse wraps paginated results.
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"pageSize"`
}
