package command_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/rent-a-girlfriend/identity-service/internal/application/command"
	"github.com/rent-a-girlfriend/identity-service/internal/application/port"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/vo"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/event"
)

// --- Mocks ---

type MockGoogleOAuthProvider struct {
	mock.Mock
}

func (m *MockGoogleOAuthProvider) BuildAuthURL(state, codeChallenge string) string {
	args := m.Called(state, codeChallenge)
	return args.String(0)
}

func (m *MockGoogleOAuthProvider) ExchangeCode(code, codeVerifier string) (*port.GoogleUserInfo, error) {
	args := m.Called(code, codeVerifier)
	if args.Get(0) != nil {
		return args.Get(0).(*port.GoogleUserInfo), args.Error(1)
	}
	return nil, args.Error(1)
}

type MockPKCEStore struct {
	mock.Mock
}

func (m *MockPKCEStore) Store(ctx context.Context, state, codeVerifier string) error {
	args := m.Called(ctx, state, codeVerifier)
	return args.Error(0)
}

func (m *MockPKCEStore) Retrieve(ctx context.Context, state string) (string, error) {
	args := m.Called(ctx, state)
	return args.String(0), args.Error(1)
}

type MockUserAccountRepository struct {
	mock.Mock
}

func (m *MockUserAccountRepository) Save(ctx context.Context, account *aggregate.UserAccount) error {
	args := m.Called(ctx, account)
	return args.Error(0)
}

func (m *MockUserAccountRepository) Update(ctx context.Context, account *aggregate.UserAccount) error {
	args := m.Called(ctx, account)
	return args.Error(0)
}

func (m *MockUserAccountRepository) FindByID(ctx context.Context, id vo.UserID) (*aggregate.UserAccount, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*aggregate.UserAccount), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserAccountRepository) FindByEmail(ctx context.Context, email vo.Email) (*aggregate.UserAccount, error) {
	args := m.Called(ctx, email)
	if args.Get(0) != nil {
		return args.Get(0).(*aggregate.UserAccount), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserAccountRepository) FindByGoogleID(ctx context.Context, googleID string) (*aggregate.UserAccount, error) {
	args := m.Called(ctx, googleID)
	if args.Get(0) != nil {
		return args.Get(0).(*aggregate.UserAccount), args.Error(1)
	}
	return nil, args.Error(1)
}

type MockTokenService struct {
	mock.Mock
}

func (m *MockTokenService) GenerateTokenPair(account *aggregate.UserAccount) (*port.TokenPair, error) {
	args := m.Called(account)
	if args.Get(0) != nil {
		return args.Get(0).(*port.TokenPair), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockTokenService) ValidateRefreshToken(token string) (*port.RefreshTokenClaims, error) {
	args := m.Called(token)
	if args.Get(0) != nil {
		return args.Get(0).(*port.RefreshTokenClaims), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockTokenService) RevokeRefreshToken(tokenID string) error {
	args := m.Called(tokenID)
	return args.Error(0)
}

func (m *MockTokenService) RevokeAllUserTokens(userID vo.UserID) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockTokenService) RotateRefreshToken(claims *port.RefreshTokenClaims, account *aggregate.UserAccount) (*port.TokenPair, error) {
	args := m.Called(claims, account)
	if args.Get(0) != nil {
		return args.Get(0).(*port.TokenPair), args.Error(1)
	}
	return nil, args.Error(1)
}

type MockEventPublisher struct {
	mock.Mock
}

func (m *MockEventPublisher) Publish(ctx context.Context, evt event.DomainEvent) error {
	args := m.Called(ctx, evt)
	return args.Error(0)
}

// --- Tests ---

func TestLoginGoogleHandler_Success_NewUser(t *testing.T) {
	oauthMock := new(MockGoogleOAuthProvider)
	pkceMock := new(MockPKCEStore)
	repoMock := new(MockUserAccountRepository)
	tokenMock := new(MockTokenService)
	pubMock := new(MockEventPublisher)

	handler := command.NewLoginGoogleHandler(oauthMock, pkceMock, repoMock, tokenMock, pubMock)
	ctx := context.Background()

	cmd := command.LoginGoogleCommand{
		Code:  "auth-code",
		State: "auth-state",
	}

	// 1. Retrieve PKCE
	pkceMock.On("Retrieve", ctx, "auth-state").Return("code-verifier", nil)

	// 2. Exchange Code
	userInfo := &port.GoogleUserInfo{
		GoogleID: "google-123",
		Email:    "test@example.com",
		Name:     "Test User",
	}
	oauthMock.On("ExchangeCode", "auth-code", "code-verifier").Return(userInfo, nil)

	// 3. FindByGoogleID -> Not Found (New User)
	repoMock.On("FindByGoogleID", ctx, "google-123").Return((*aggregate.UserAccount)(nil), assert.AnError)

	// 4. Save new account
	repoMock.On("Save", ctx, mock.AnythingOfType("*aggregate.UserAccount")).Return(nil)

	// 5. Publish Event
	pubMock.On("Publish", ctx, mock.Anything).Return(nil)

	// 6. Generate Token
	tokenPair := &port.TokenPair{
		AccessToken:  "access",
		RefreshToken: "refresh",
	}
	tokenMock.On("GenerateTokenPair", mock.AnythingOfType("*aggregate.UserAccount")).Return(tokenPair, nil)

	// Execute
	res, err := handler.Handle(ctx, cmd)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, "access", res.AccessToken)

	pkceMock.AssertExpectations(t)
	oauthMock.AssertExpectations(t)
	repoMock.AssertExpectations(t)
	pubMock.AssertExpectations(t)
	tokenMock.AssertExpectations(t)
}
