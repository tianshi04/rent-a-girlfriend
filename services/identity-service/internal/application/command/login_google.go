package command

import (
	"context"
	"time"

	domainerr "github.com/rent-a-girlfriend/identity-service/internal/domain/errors"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/repository"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/vo"
	"github.com/rent-a-girlfriend/identity-service/internal/application/port"
)

// LoginGoogleCommand contains the OAuth callback parameters.
type LoginGoogleCommand struct {
	Code  string
	State string
}

// LoginGoogleHandler exchanges the Google auth code for tokens and creates/finds the user.
type LoginGoogleHandler struct {
	oauthProvider port.GoogleOAuthProvider
	pkceStore     port.PKCEStore
	accountRepo   repository.UserAccountRepository
	tokenService  port.TokenService
	publisher     port.EventPublisher
}

// NewLoginGoogleHandler creates a new handler.
func NewLoginGoogleHandler(
	oauthProvider port.GoogleOAuthProvider,
	pkceStore port.PKCEStore,
	accountRepo repository.UserAccountRepository,
	tokenService port.TokenService,
	publisher port.EventPublisher,
) *LoginGoogleHandler {
	return &LoginGoogleHandler{
		oauthProvider: oauthProvider,
		pkceStore:     pkceStore,
		accountRepo:   accountRepo,
		tokenService:  tokenService,
		publisher:     publisher,
	}
}

// Handle processes the Google OAuth callback with PKCE verification.
func (h *LoginGoogleHandler) Handle(ctx context.Context, cmd LoginGoogleCommand) (*port.TokenPair, error) {
	// 1. Retrieve PKCE code_verifier by state
	codeVerifier, err := h.pkceStore.Retrieve(ctx, cmd.State)
	if err != nil {
		return nil, domainerr.ErrPKCEStateNotFound
	}

	// 2. Exchange auth code + code_verifier with Google
	userInfo, err := h.oauthProvider.ExchangeCode(cmd.Code, codeVerifier)
	if err != nil {
		return nil, domainerr.ErrInvalidOAuthToken
	}

	// 3. Find or create UserAccount
	account, err := h.accountRepo.FindByGoogleID(ctx, userInfo.GoogleID)
	if err != nil {
		// New user — create account with default role CLIENT
		email, emailErr := vo.NewEmail(userInfo.Email)
		if emailErr != nil {
			return nil, emailErr
		}

		account = aggregate.NewUserAccount(email, userInfo.GoogleID, time.Now())
		if saveErr := h.accountRepo.Save(ctx, account); saveErr != nil {
			return nil, saveErr
		}

		// Publish UserRegistered event
		for _, evt := range account.Events() {
			if pubErr := h.publisher.Publish(ctx, evt); pubErr != nil {
				return nil, pubErr
			}
		}
	}

	// 4. Check [INV-ID01]: account must not be LOCKED
	if err := account.CheckLoginAllowed(); err != nil {
		return nil, err
	}

	// 5. Generate JWT token pair
	tokenPair, err := h.tokenService.GenerateTokenPair(account)
	if err != nil {
		return nil, err
	}

	return tokenPair, nil
}
