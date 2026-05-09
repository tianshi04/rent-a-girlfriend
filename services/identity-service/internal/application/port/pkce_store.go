package port

import "context"

// PKCEData contains the PKCE challenge pair.
type PKCEData struct {
	State         string
	CodeVerifier  string
	CodeChallenge string
}

// PKCEStore is the port for storing PKCE code verifiers keyed by state.
type PKCEStore interface {
	// Store saves a code_verifier associated with the given state (with TTL).
	Store(ctx context.Context, state, codeVerifier string) error

	// Retrieve gets and deletes the code_verifier for the given state.
	// Returns ErrPKCEStateNotFound if not found or expired.
	Retrieve(ctx context.Context, state string) (string, error)
}
