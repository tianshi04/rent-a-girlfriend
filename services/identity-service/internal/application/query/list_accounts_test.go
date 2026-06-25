package query_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/rent-a-girlfriend/identity-service/internal/application/query"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/vo"
)

type MockUserAccountRepository struct {
	mock.Mock
}

func (m *MockUserAccountRepository) Save(ctx context.Context, account *aggregate.UserAccount) error {
	return nil
}

func (m *MockUserAccountRepository) Update(ctx context.Context, account *aggregate.UserAccount) error {
	return nil
}

func (m *MockUserAccountRepository) FindByID(ctx context.Context, id vo.UserID) (*aggregate.UserAccount, error) {
	return nil, nil
}

func (m *MockUserAccountRepository) FindByEmail(ctx context.Context, email vo.Email) (*aggregate.UserAccount, error) {
	return nil, nil
}

func (m *MockUserAccountRepository) FindByGoogleID(ctx context.Context, googleID string) (*aggregate.UserAccount, error) {
	return nil, nil
}

func (m *MockUserAccountRepository) FindAll(ctx context.Context, page int, pageSize int) ([]*aggregate.UserAccount, int64, error) {
	args := m.Called(ctx, page, pageSize)
	if args.Get(0) != nil {
		return args.Get(0).([]*aggregate.UserAccount), args.Get(1).(int64), args.Error(2)
	}
	return nil, 0, args.Error(2)
}

func TestListAccountsHandler_Handle_Success(t *testing.T) {
	repoMock := new(MockUserAccountRepository)
	handler := query.NewListAccountsHandler(repoMock)
	ctx := context.Background()

	id1, _ := vo.ParseUserID("550e8400-e29b-41d4-a716-446655440001")
	email1, _ := vo.NewEmail("user1@example.com")
	acc1 := aggregate.Reconstitute(id1, email1, "g-1", vo.RoleClient, vo.StatusActive, 0, 1, time.Now(), time.Now())

	id2, _ := vo.ParseUserID("550e8400-e29b-41d4-a716-446655440002")
	email2, _ := vo.NewEmail("user2@example.com")
	acc2 := aggregate.Reconstitute(id2, email2, "g-2", vo.RoleClient, vo.StatusActive, 0, 1, time.Now(), time.Now())

	expectedAccounts := []*aggregate.UserAccount{acc1, acc2}
	repoMock.On("FindAll", ctx, 1, 10).Return(expectedAccounts, int64(2), nil)

	res, total, err := handler.Handle(ctx, query.ListAccountsQuery{
		Page:     1,
		PageSize: 10,
	})

	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, res, 2)
	assert.Equal(t, acc1.Email(), res[0].Email())
	assert.Equal(t, acc2.Email(), res[1].Email())

	repoMock.AssertExpectations(t)
}

func TestListAccountsHandler_Handle_DefaultPagination(t *testing.T) {
	repoMock := new(MockUserAccountRepository)
	handler := query.NewListAccountsHandler(repoMock)
	ctx := context.Background()

	// Should default page to 1 and page_size to 20 when invalid pagination is provided
	repoMock.On("FindAll", ctx, 1, 20).Return(([]*aggregate.UserAccount)(nil), int64(0), nil)

	_, _, err := handler.Handle(ctx, query.ListAccountsQuery{
		Page:     -1,
		PageSize: 0,
	})

	assert.NoError(t, err)
	repoMock.AssertExpectations(t)
}
