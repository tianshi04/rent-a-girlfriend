package command_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/rent-a-girlfriend/booking-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/event"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/repository"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// MockBookingRepository implements repository.BookingRepository using in-memory storage.
type MockBookingRepository struct {
	mu         sync.RWMutex
	bookings   map[string]*aggregate.Booking
	SaveError  error
	FindError  error
	CountError error
	OverlapErr error
}

func NewMockBookingRepository() *MockBookingRepository {
	return &MockBookingRepository{
		bookings: make(map[string]*aggregate.Booking),
	}
}

func (m *MockBookingRepository) Save(ctx context.Context, booking *aggregate.Booking) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.SaveError != nil {
		return m.SaveError
	}
	m.bookings[booking.ID().String()] = booking
	return nil
}

func (m *MockBookingRepository) Update(ctx context.Context, booking *aggregate.Booking) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.SaveError != nil {
		return m.SaveError
	}
	m.bookings[booking.ID().String()] = booking
	return nil
}

func (m *MockBookingRepository) FindByID(ctx context.Context, id vo.BookingID) (*aggregate.Booking, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.FindError != nil {
		return nil, m.FindError
	}
	b, ok := m.bookings[id.String()]
	if !ok {
		return nil, errors.New("booking not found")
	}
	return b, nil
}

func (m *MockBookingRepository) CountPendingByCompanion(ctx context.Context, companionID vo.CompanionID) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.CountError != nil {
		return 0, m.CountError
	}
	var count int64 = 0
	for _, b := range m.bookings {
		if b.CompanionID().Equals(companionID) && b.Status() == vo.StatusPending {
			count++
		}
	}
	return count, nil
}

func (m *MockBookingRepository) HasOverlappingBooking(ctx context.Context, actorID string, isCompanion bool, statuses []vo.BookingStatus, startTime, endTime time.Time) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.OverlapErr != nil {
		return false, m.OverlapErr
	}

	statusMap := make(map[vo.BookingStatus]bool)
	for _, s := range statuses {
		statusMap[s] = true
	}

	for _, b := range m.bookings {
		if !statusMap[b.Status()] {
			continue
		}

		match := false
		if isCompanion {
			match = b.CompanionID().String() == actorID
		} else {
			match = b.ClientID().String() == actorID
		}

		if match {
			// Check overlap: start1 < end2 AND start2 < end1
			if b.TimeRange().StartTime().Before(endTime) && startTime.Before(b.TimeRange().EndTime()) {
				return true, nil
			}
		}
	}
	return false, nil
}

func (m *MockBookingRepository) FindByFilters(ctx context.Context, filters repository.BookingFilters) ([]*aggregate.Booking, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*aggregate.Booking
	for _, b := range m.bookings {
		if filters.ClientID != nil && b.ClientID().String() != *filters.ClientID {
			continue
		}
		if filters.CompanionID != nil && b.CompanionID().String() != *filters.CompanionID {
			continue
		}
		if len(filters.Statuses) > 0 {
			matchStatus := false
			for _, st := range filters.Statuses {
				if string(b.Status()) == st {
					matchStatus = true
					break
				}
			}
			if !matchStatus {
				continue
			}
		}
		result = append(result, b)
	}
	return result, int64(len(result)), nil
}

func (m *MockBookingRepository) FindAcceptedBookingsPastEndTimeBuffer(ctx context.Context, now time.Time, buffer time.Duration) ([]*aggregate.Booking, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*aggregate.Booking
	for _, b := range m.bookings {
		if b.Status() == vo.StatusAccepted {
			deadline := b.TimeRange().EndTime().Add(buffer)
			if now.After(deadline) {
				result = append(result, b)
			}
		}
	}
	return result, nil
}

func (m *MockBookingRepository) FindPendingBookingsEligibleForTimeout(ctx context.Context, now time.Time) ([]*aggregate.Booking, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*aggregate.Booking
	for _, b := range m.bookings {
		if b.Status() == vo.StatusPending {
			if b.CreatedAt().Add(12*time.Hour).Before(now) || b.TimeRange().StartTime().Add(-1*time.Hour).Before(now) {
				result = append(result, b)
			}
		}
	}
	return result, nil
}

// MockBookingSagaRepository implements repository.BookingSagaRepository.
type MockBookingSagaRepository struct {
	mu        sync.RWMutex
	sagas     map[string]*aggregate.BookingAcceptSaga
	SaveError error
}

func NewMockBookingSagaRepository() *MockBookingSagaRepository {
	return &MockBookingSagaRepository{
		sagas: make(map[string]*aggregate.BookingAcceptSaga),
	}
}

func (m *MockBookingSagaRepository) Save(ctx context.Context, saga *aggregate.BookingAcceptSaga) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.SaveError != nil {
		return m.SaveError
	}
	m.sagas[saga.ID] = saga
	return nil
}

func (m *MockBookingSagaRepository) Update(ctx context.Context, saga *aggregate.BookingAcceptSaga) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.SaveError != nil {
		return m.SaveError
	}
	m.sagas[saga.ID] = saga
	return nil
}

func (m *MockBookingSagaRepository) FindByID(ctx context.Context, id string) (*aggregate.BookingAcceptSaga, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sagas[id]
	if !ok {
		return nil, errors.New("saga not found")
	}
	return s, nil
}

func (m *MockBookingSagaRepository) FindByBookingID(ctx context.Context, bookingID string) (*aggregate.BookingAcceptSaga, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, s := range m.sagas {
		if s.BookingID == bookingID {
			return s, nil
		}
	}
	return nil, errors.New("saga not found")
}

// MockProfileService implements port.ProfileService.
type MockProfileService struct {
	Snapshot *vo.ScenarioSnapshot
	Err      error
}

func (m *MockProfileService) GetScenarioSnapshot(ctx context.Context, scenarioID string) (*vo.ScenarioSnapshot, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Snapshot, nil
}

// MockFinanceService implements port.FinanceService.
type MockFinanceService struct {
	mu          sync.Mutex
	FrozenCalls []string
	RefundCalls []string
	FreezeErr   error
	UnfreezeErr error
}

func (m *MockFinanceService) FreezeCoin(ctx context.Context, clientID vo.ClientID, amount vo.Money) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.FreezeErr != nil {
		return m.FreezeErr
	}
	m.FrozenCalls = append(m.FrozenCalls, clientID.String())
	return nil
}

func (m *MockFinanceService) UnfreezeCoin(ctx context.Context, clientID vo.ClientID, amount vo.Money) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.UnfreezeErr != nil {
		return m.UnfreezeErr
	}
	m.RefundCalls = append(m.RefundCalls, clientID.String())
	return nil
}

// --- Mock GORM Database using custom dummy SQL driver ---

type mockDriver struct{}

func (d mockDriver) Open(name string) (driver.Conn, error) {
	return &mockConn{}, nil
}

type mockConn struct{}

func (c *mockConn) Prepare(query string) (driver.Stmt, error) {
	return &mockStmt{}, nil
}

func (c *mockConn) Close() error { return nil }

func (c *mockConn) Begin() (driver.Tx, error) {
	return &mockTx{}, nil
}

type mockStmt struct{}

func (s *mockStmt) Close() error  { return nil }
func (s *mockStmt) NumInput() int { return -1 }
func (s *mockStmt) Exec(args []driver.Value) (driver.Result, error) {
	return &mockResult{}, nil
}
func (s *mockStmt) Query(args []driver.Value) (driver.Rows, error) {
	return &mockRows{}, nil
}

type mockTx struct{}

func (t *mockTx) Commit() error   { return nil }
func (t *mockTx) Rollback() error { return nil }

type mockResult struct{}

func (r *mockResult) LastInsertId() (int64, error) { return 0, nil }
func (r *mockResult) RowsAffected() (int64, error) { return 0, nil }

type mockRows struct{}

func (r *mockRows) Columns() []string              { return nil }
func (r *mockRows) Close() error                   { return nil }
func (r *mockRows) Next(dest []driver.Value) error { return io.EOF }

func init() {
	sql.Register("mock_gorm_driver", mockDriver{})
}

// NewMockGormDB returns a real *gorm.DB that executes commands against our dummy driver.
func NewMockGormDB() *gorm.DB {
	sqlDB, err := sql.Open("mock_gorm_driver", "mock_dsn")
	if err != nil {
		panic(err)
	}
	db, err := gorm.Open(postgres.New(postgres.Config{
		Conn: sqlDB,
	}), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	if err != nil {
		panic(err)
	}
	return db
}

// MockEventPublisher implements port.EventPublisher.
type MockEventPublisher struct {
	mu           sync.Mutex
	Events       []event.DomainEvent
	PublishError error
}

func NewMockEventPublisher() *MockEventPublisher {
	return &MockEventPublisher{}
}

func (m *MockEventPublisher) Publish(ctx context.Context, evt event.DomainEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.PublishError != nil {
		return m.PublishError
	}
	m.Events = append(m.Events, evt)
	return nil
}
