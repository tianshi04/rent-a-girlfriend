package port

// GoogleUserInfo contains the user info returned from Google OAuth.
type GoogleUserInfo struct {
	GoogleID string
	Email    string
	Name     string
	Picture  string
}

// GoogleOAuthProvider is the port for interacting with Google OAuth 2.0.
type GoogleOAuthProvider interface {
	// BuildAuthURL builds the Google consent URL with PKCE code_challenge and state.
	BuildAuthURL(state, codeChallenge string) string

	// ExchangeCode exchanges an authorization code for user info using PKCE code_verifier.
	ExchangeCode(code, codeVerifier string) (*GoogleUserInfo, error)
}
