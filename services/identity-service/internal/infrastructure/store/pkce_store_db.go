package store

import (
	"context"
	"time"

	"gorm.io/gorm"

	domainerr "github.com/rent-a-girlfriend/identity-service/internal/domain/errors"
	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/persistence"
)

const pkceTTL = 10 * time.Minute

// PKCEStoreDB implements PKCEStore using the database.
type PKCEStoreDB struct {
	db *gorm.DB
}

// NewPKCEStoreDB creates a new DB-backed PKCE store.
func NewPKCEStoreDB(db *gorm.DB) *PKCEStoreDB {
	return &PKCEStoreDB{db: db}
}

func (s *PKCEStoreDB) Store(ctx context.Context, state, codeVerifier string) error {
	model := persistence.PKCEVerifierModel{
		State:        state,
		CodeVerifier: codeVerifier,
		ExpiresAt:    time.Now().Add(pkceTTL),
	}
	return s.db.WithContext(ctx).Create(&model).Error
}

func (s *PKCEStoreDB) Retrieve(ctx context.Context, state string) (string, error) {
	var model persistence.PKCEVerifierModel
	if err := s.db.WithContext(ctx).Where("state = ?", state).First(&model).Error; err != nil {
		return "", domainerr.ErrPKCEStateNotFound
	}

	// Delete after retrieval (one-time use)
	s.db.WithContext(ctx).Delete(&model)

	if time.Now().After(model.ExpiresAt) {
		return "", domainerr.ErrPKCEStateNotFound
	}

	return model.CodeVerifier, nil
}
