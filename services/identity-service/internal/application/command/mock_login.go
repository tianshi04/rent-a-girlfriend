package command

import (
	"context"
	"time"

	"github.com/rent-a-girlfriend/identity-service/internal/application/port"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/repository"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/vo"
)

type MockLoginCommand struct {
	Email    string
	GoogleID string
	Role     string
}

type MockLoginHandler struct {
	accountRepo  repository.UserAccountRepository
	tokenService port.TokenService
	publisher    port.EventPublisher
}

func NewMockLoginHandler(
	accountRepo repository.UserAccountRepository,
	tokenService port.TokenService,
	publisher port.EventPublisher,
) *MockLoginHandler {
	return &MockLoginHandler{
		accountRepo:  accountRepo,
		tokenService: tokenService,
		publisher:    publisher,
	}
}

func (h *MockLoginHandler) Handle(ctx context.Context, cmd MockLoginCommand) (*port.TokenPair, error) {
	// Find or create UserAccount
	account, err := h.accountRepo.FindByGoogleID(ctx, cmd.GoogleID)
	if err != nil {
		// New user
		email, err := vo.NewEmail(cmd.Email)
		if err != nil {
			return nil, err
		}

		account = aggregate.NewUserAccount(email, cmd.GoogleID, time.Now())

		// Override role for testing if provided
		if cmd.Role != "" {
			if cmd.Role == string(vo.RoleAdmin) {
				account.PromoteToAdmin()
			}
		}

		if saveErr := h.accountRepo.Save(ctx, account); saveErr != nil {
			return nil, saveErr
		}

		// Publish UserRegistered event
		for _, evt := range account.Events() {
			if pubErr := h.publisher.Publish(ctx, evt); pubErr != nil {
				return nil, pubErr
			}
		}
	} else {
		// If user exists and role is requested to be upgraded to Admin for testing
		if cmd.Role == string(vo.RoleAdmin) && account.Role() != vo.RoleAdmin {
			account.PromoteToAdmin()
			if updateErr := h.accountRepo.Update(ctx, account); updateErr != nil {
				return nil, updateErr
			}
		}
	}

	// Check [INV-ID01]
	if err := account.CheckLoginAllowed(); err != nil {
		return nil, err
	}

	// Generate JWT
	tokenPair, err := h.tokenService.GenerateTokenPair(account)
	if err != nil {
		return nil, err
	}

	return tokenPair, nil
}
