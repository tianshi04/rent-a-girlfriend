package query

import (
	"github.com/rent-a-girlfriend/identity-service/internal/application/port"
)

// GetJWKSHandler returns the public keys in JWKS format.
type GetJWKSHandler struct {
	keyProvider port.KeyProvider
}

// NewGetJWKSHandler creates a new handler.
func NewGetJWKSHandler(keyProvider port.KeyProvider) *GetJWKSHandler {
	return &GetJWKSHandler{keyProvider: keyProvider}
}

// Handle returns the JWKS response.
func (h *GetJWKSHandler) Handle() (*port.JWKSResponse, error) {
	return h.keyProvider.GetJWKS()
}
