package persistence

import (
	"time"

	"github.com/rent-a-girlfriend/booking-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
)

// BookingModel is the GORM database model for the bookings table.
type BookingModel struct {
	ID               string    `gorm:"column:id;type:uuid;primaryKey"`
	ClientID         string    `gorm:"column:client_id;type:uuid;not null;index:idx_bookings_client_companion_status;index:idx_bookings_client_status"`
	CompanionID      string    `gorm:"column:companion_id;type:uuid;not null;index:idx_bookings_client_companion_status;index:idx_bookings_companion_status"`
	ScenarioPrice    int64     `gorm:"column:scenario_price;not null"`
	ScenarioDuration int64     `gorm:"column:scenario_duration;not null"`
	StartTime        time.Time `gorm:"column:start_time;type:timestamptz;not null"`
	EndTime          time.Time `gorm:"column:end_time;type:timestamptz;not null;index:idx_bookings_status_end_time"`
	Status           string    `gorm:"column:status;type:varchar(20);not null;default:'PENDING';index:idx_bookings_client_companion_status;index:idx_bookings_status_end_time;index:idx_bookings_companion_status;index:idx_bookings_client_status"`
	CancelledByRole  string    `gorm:"column:cancelled_by_role;type:varchar(20)"`
	IsLateCancel     bool      `gorm:"column:is_late_cancel;default:false"`
	Version          int64     `gorm:"column:version;not null;default:1"`
	CreatedAt        time.Time `gorm:"column:created_at;type:timestamptz;not null;autoCreateTime"`
	UpdatedAt        time.Time `gorm:"column:updated_at;type:timestamptz;not null;autoUpdateTime"`
}

// TableName specifies the table name for GORM.
func (BookingModel) TableName() string {
	return "bookings"
}

// OutboxModel is the GORM database model for the outbox table (prepared for Phase 2).
type OutboxModel struct {
	ID            string    `gorm:"column:id;type:uuid;primaryKey"`
	AggregateType string    `gorm:"column:aggregate_type;type:varchar(50);not null"`
	AggregateID   string    `gorm:"column:aggregate_id;type:uuid;not null"`
	EventType     string    `gorm:"column:event_type;type:varchar(100);not null"`
	Payload       string    `gorm:"column:payload;type:jsonb;not null"`
	CreatedAt     time.Time `gorm:"column:created_at;type:timestamptz;not null;autoCreateTime"`
	Published     bool      `gorm:"column:published;default:false;index:idx_outbox_unpublished"`
	PublishedAt   *time.Time `gorm:"column:published_at;type:timestamptz"`
}

// TableName specifies the table name for GORM.
func (OutboxModel) TableName() string {
	return "outbox"
}

// --- Mapping functions ---

// ToDomain converts a BookingModel to a domain Booking aggregate.
func (m *BookingModel) ToDomain() (*aggregate.Booking, error) {
	bookingID, err := vo.BookingIDFromString(m.ID)
	if err != nil {
		return nil, err
	}
	clientID, err := vo.NewClientID(m.ClientID)
	if err != nil {
		return nil, err
	}
	companionID, err := vo.NewCompanionID(m.CompanionID)
	if err != nil {
		return nil, err
	}
	price, err := vo.NewMoney(m.ScenarioPrice)
	if err != nil {
		return nil, err
	}
	scenario, err := vo.NewScenarioSnapshot(price, m.ScenarioDuration)
	if err != nil {
		return nil, err
	}
	timeRange, err := vo.NewTimeRange(m.StartTime, m.EndTime)
	if err != nil {
		return nil, err
	}

	return aggregate.Reconstitute(
		bookingID,
		clientID,
		companionID,
		scenario,
		timeRange,
		vo.BookingStatus(m.Status),
		vo.ActorRole(m.CancelledByRole),
		m.IsLateCancel,
		m.Version,
		m.CreatedAt,
		m.UpdatedAt,
	), nil
}

// ToModel converts a domain Booking aggregate to a BookingModel for persistence.
func ToModel(b *aggregate.Booking) *BookingModel {
	return &BookingModel{
		ID:               b.ID().String(),
		ClientID:         b.ClientID().String(),
		CompanionID:      b.CompanionID().String(),
		ScenarioPrice:    b.Scenario().Price().Amount(),
		ScenarioDuration: b.Scenario().DurationMinutes(),
		StartTime:        b.TimeRange().StartTime(),
		EndTime:          b.TimeRange().EndTime(),
		Status:           string(b.Status()),
		CancelledByRole:  string(b.CancelledByRole()),
		IsLateCancel:     b.IsLateCancel(),
		Version:          b.Version(),
		CreatedAt:        b.CreatedAt(),
		UpdatedAt:        b.UpdatedAt(),
	}
}

// BookingAcceptSagaModel is the GORM database model for the Saga.
type BookingAcceptSagaModel struct {
	ID        string    `gorm:"column:id;type:uuid;primaryKey"`
	BookingID string    `gorm:"column:booking_id;type:uuid;not null"`
	State     string    `gorm:"column:state;type:varchar(50);not null"`
	CreatedAt time.Time `gorm:"column:created_at;type:timestamptz;not null;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:timestamptz;not null;autoUpdateTime"`
}

func (BookingAcceptSagaModel) TableName() string {
	return "booking_accept_sagas"
}

func (m *BookingAcceptSagaModel) ToDomain() *aggregate.BookingAcceptSaga {
	saga := aggregate.NewBookingAcceptSaga(m.ID, m.BookingID, m.CreatedAt)
	saga.UpdateState(vo.SagaState(m.State), m.UpdatedAt)
	return saga
}

func ToSagaModel(s *aggregate.BookingAcceptSaga) *BookingAcceptSagaModel {
	return &BookingAcceptSagaModel{
		ID:        s.ID,
		BookingID: s.BookingID,
		State:     string(s.State),
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}
