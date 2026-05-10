package command

import (
	"context"
	"errors"
	"strconv"

	"github.com/rent-a-girlfriend/identity-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/event"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/vo"
	"github.com/rent-a-girlfriend/identity-service/internal/application/port"
)

// Mock objects for testing
type mockRepo struct {
	accounts map[string]*aggregate.UserAccount
	saveErr  error
}

func (m *mockRepo) Save(ctx context.Context, a *aggregate.UserAccount) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.accounts[a.ID().String()] = a
	return nil
}

func (m *mockRepo) Update(ctx context.Context, a *aggregate.UserAccount) error {
	m.accounts[a.ID().String()] = a
	return nil
}

func (m *mockRepo) FindByID(ctx context.Context, id vo.UserID) (*aggregate.UserAccount, error) {
	if a, ok := m.accounts[id.String()]; ok {
		return a, nil
	}
	return nil, errors.New("not found")
}

func (m *mockRepo) FindByEmail(ctx context.Context, email vo.Email) (*aggregate.UserAccount, error) {
	for _, a := range m.accounts {
		if a.Email() == email {
			return a, nil
		}
	}
	return nil, nil
}

func (m *mockRepo) FindByGoogleID(ctx context.Context, gid string) (*aggregate.UserAccount, error) {
	for _, a := range m.accounts {
		if a.GoogleID() == gid {
			return a, nil
		}
	}
	return nil, nil
}

type mockConfigRepo struct {
	configs map[string]string
}

func (m *mockConfigRepo) Get(ctx context.Context, key, defaultValue string) (string, error) {
	if v, ok := m.configs[key]; ok {
		return v, nil
	}
	return defaultValue, nil
}

func (m *mockConfigRepo) GetInt(ctx context.Context, key string, defaultValue int) (int, error) {
	if v, ok := m.configs[key]; ok {
		return strconv.Atoi(v)
	}
	return defaultValue, nil
}

func (m *mockConfigRepo) Set(ctx context.Context, key, value string) error {
	m.configs[key] = value
	return nil
}

type mockPublisher struct {
	events []event.DomainEvent
}

func (m *mockPublisher) Publish(ctx context.Context, e event.DomainEvent) error {
	m.events = append(m.events, e)
	return nil
}

type mockTokenService struct{}

func (m *mockTokenService) GenerateTokenPair(a *aggregate.UserAccount) (*port.TokenPair, error) {
	return &port.TokenPair{AccessToken: "access", RefreshToken: "refresh", ExpiresIn: 3600}, nil
}
func (m *mockTokenService) ValidateRefreshToken(t string) (*port.RefreshTokenClaims, error) {
	return &port.RefreshTokenClaims{}, nil
}
func (m *mockTokenService) RevokeRefreshToken(tid string) error { return nil }
func (m *mockTokenService) RevokeAllUserTokens(uid vo.UserID) error { return nil }
func (m *mockTokenService) RotateRefreshToken(c *port.RefreshTokenClaims, a *aggregate.UserAccount) (*port.TokenPair, error) {
	return &port.TokenPair{AccessToken: "access-new", RefreshToken: "refresh-new", ExpiresIn: 3600}, nil
}
