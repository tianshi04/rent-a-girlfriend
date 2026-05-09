package repository

import (
	"context"

	"github.com/rent-a-girlfriend/identity-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/vo"
)

// UserAccountRepository is the port interface for persisting UserAccount aggregates.
// Implementations reside in the infrastructure layer.
type UserAccountRepository interface {
	// Save persists a new UserAccount aggregate.
	Save(ctx context.Context, account *aggregate.UserAccount) error

	// Update persists changes to an existing UserAccount with optimistic locking.
	Update(ctx context.Context, account *aggregate.UserAccount) error

	// FindByID retrieves a UserAccount by its ID.
	FindByID(ctx context.Context, id vo.UserID) (*aggregate.UserAccount, error)

	// FindByEmail retrieves a UserAccount by email.
	FindByEmail(ctx context.Context, email vo.Email) (*aggregate.UserAccount, error)

	// FindByGoogleID retrieves a UserAccount by Google OAuth ID.
	FindByGoogleID(ctx context.Context, googleID string) (*aggregate.UserAccount, error)
}
