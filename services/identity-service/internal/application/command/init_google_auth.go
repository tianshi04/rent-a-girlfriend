package command

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"

	"github.com/rent-a-girlfriend/identity-service/internal/application/port"
)

// InitGoogleAuthResult contains the data needed by the client to start OAuth.
type InitGoogleAuthResult struct {
	AuthURL       string `json:"authUrl"`
	State         string `json:"state"`
	CodeChallenge string `json:"codeChallenge"`
}

// InitGoogleAuthHandler generates PKCE params and returns the Google auth URL.
type InitGoogleAuthHandler struct {
	oauthProvider port.GoogleOAuthProvider
	pkceStore     port.PKCEStore
}

// NewInitGoogleAuthHandler creates a new handler.
func NewInitGoogleAuthHandler(
	oauthProvider port.GoogleOAuthProvider,
	pkceStore port.PKCEStore,
) *InitGoogleAuthHandler {
	return &InitGoogleAuthHandler{
		oauthProvider: oauthProvider,
		pkceStore:     pkceStore,
	}
}

// Handle generates PKCE code_verifier/code_challenge, stores verifier, returns auth URL.
func (h *InitGoogleAuthHandler) Handle(ctx context.Context) (*InitGoogleAuthResult, error) {
	// Generate cryptographically random state
	state, err := generateRandomString(32)
	if err != nil {
		return nil, err
	}

	// Generate PKCE code_verifier (43-128 chars, RFC 7636)
	codeVerifier, err := generateRandomString(64)
	if err != nil {
		return nil, err
	}

	// code_challenge = BASE64URL(SHA256(code_verifier))
	codeChallenge := computeCodeChallenge(codeVerifier)

	// Store code_verifier keyed by state (TTL handled by store impl)
	if err := h.pkceStore.Store(ctx, state, codeVerifier); err != nil {
		return nil, err
	}

	// Build Google auth URL with PKCE
	authURL := h.oauthProvider.BuildAuthURL(state, codeChallenge)

	return &InitGoogleAuthResult{
		AuthURL:       authURL,
		State:         state,
		CodeChallenge: codeChallenge,
	}, nil
}

func generateRandomString(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func computeCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}
