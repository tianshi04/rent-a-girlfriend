package port

// JWKSResponse represents a JSON Web Key Set.
type JWKSResponse struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a single JSON Web Key.
type JWK struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// KeyProvider is the port for managing RSA signing keys and exposing JWKS.
type KeyProvider interface {
	// GetJWKS returns all active public keys in JWKS format.
	GetJWKS() (*JWKSResponse, error)
}
