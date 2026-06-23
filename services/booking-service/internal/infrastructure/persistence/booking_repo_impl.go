package persistence

import (
	"context"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/rent-a-girlfriend/booking-service/internal/domain/aggregate"
	domainerr "github.com/rent-a-girlfriend/booking-service/internal/domain/errors"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/event"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/repository"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
)

// BookingRepositoryImpl is the GORM implementation of BookingRepository.
type BookingRepositoryImpl struct {
	db *gorm.DB
}

func NewBookingRepository(db *gorm.DB) *BookingRepositoryImpl {
	return &BookingRepositoryImpl{db: db}
}

func (r *BookingRepositoryImpl) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(vo.TxKey).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}

func (r *BookingRepositoryImpl) Save(ctx context.Context, booking *aggregate.Booking) error {
	tx, ok := ctx.Value(vo.TxKey).(*gorm.DB)
	if ok {
		return r.save(tx.WithContext(ctx), booking)
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return r.save(tx, booking)
	})
}

func (r *BookingRepositoryImpl) save(db *gorm.DB, booking *aggregate.Booking) error {
	model := ToModel(booking)
	if err := db.Create(model).Error; err != nil {
		return err
	}

	// Write outbox events
	for _, evt := range booking.Events() {
		payload, err := protojson.Marshal(evt.ToProto())
		if err != nil {
			return err
		}
		outbox := &OutboxModel{
			ID:            uuid.New().String(),
			AggregateType: "Booking",
			AggregateID:   extractBookingID(evt),
			EventType:     evt.EventType(),
			Payload:       string(payload),
			Published:     false,
			CreatedAt:     time.Now(),
		}
		if err := db.Create(outbox).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *BookingRepositoryImpl) Update(ctx context.Context, booking *aggregate.Booking) error {
	tx, ok := ctx.Value(vo.TxKey).(*gorm.DB)
	if ok {
		return r.update(tx.WithContext(ctx), booking)
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return r.update(tx, booking)
	})
}

func (r *BookingRepositoryImpl) update(db *gorm.DB, booking *aggregate.Booking) error {
	model := ToModel(booking)
	oldVersion := model.Version
	model.Version = oldVersion + 1

	// Optimistic locking: only update if version matches
	result := db.
		Model(&BookingModel{}).
		Where("id = ? AND version = ?", model.ID, oldVersion).
		Clauses(clause.Returning{}).
		Updates(map[string]interface{}{
			"status":            model.Status,
			"cancelled_by_role": model.CancelledByRole,
			"is_late_cancel":    model.IsLateCancel,
			"version":           model.Version,
			"updated_at":        model.UpdatedAt,
		})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerr.ErrConcurrencyConflict
	}

	// Write outbox events
	for _, evt := range booking.Events() {
		payload, err := protojson.Marshal(evt.ToProto())
		if err != nil {
			return err
		}
		outbox := &OutboxModel{
			ID:            uuid.New().String(),
			AggregateType: "Booking",
			AggregateID:   extractBookingID(evt),
			EventType:     evt.EventType(),
			Payload:       string(payload),
			Published:     false,
			CreatedAt:     time.Now(),
		}
		if err := db.Create(outbox).Error; err != nil {
			return err
		}
	}
	return nil
}

func extractBookingID(evt event.DomainEvent) string {
	switch e := evt.(type) {
	case event.BookingRequested:
		return e.BookingId
	case event.BookingReserved:
		return e.BookingId
	case event.BookingAccepted:
		return e.BookingId
	case event.BookingRejected:
		return e.BookingId
	case event.BookingCancelledEarly:
		return e.BookingId
	case event.BookingCancelledLate:
		return e.BookingId
	case event.BookingTimedOut:
		return e.BookingId
	case event.BookingCompleted:
		return e.BookingId
	case event.TransferToEscrowCommand:
		return e.BookingId
	case event.CreateChatRoomCommand:
		return e.BookingId
	case event.RefundEscrowCommand:
		return e.BookingId
	case event.UnfreezeCoinCommand:
		return e.BookingId
	case *event.BookingRequested:
		return e.BookingId
	case *event.BookingReserved:
		return e.BookingId
	case *event.BookingAccepted:
		return e.BookingId
	case *event.BookingRejected:
		return e.BookingId
	case *event.BookingCancelledEarly:
		return e.BookingId
	case *event.BookingCancelledLate:
		return e.BookingId
	case *event.BookingTimedOut:
		return e.BookingId
	case *event.BookingCompleted:
		return e.BookingId
	case *event.TransferToEscrowCommand:
		return e.BookingId
	case *event.CreateChatRoomCommand:
		return e.BookingId
	case *event.RefundEscrowCommand:
		return e.BookingId
	case *event.UnfreezeCoinCommand:
		return e.BookingId

	default:
		return ""
	}
}

func (r *BookingRepositoryImpl) FindByID(ctx context.Context, id vo.BookingID) (*aggregate.Booking, error) {
	var model BookingModel
	result := r.getDB(ctx).Where("id = ?", id.String()).First(&model)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, domainerr.ErrBookingNotFound
		}
		return nil, result.Error
	}
	return model.ToDomain()
}

func (r *BookingRepositoryImpl) CountPendingByCompanion(
	ctx context.Context,
	companionID vo.CompanionID,
) (int64, error) {
	var count int64
	result := r.getDB(ctx).
		Model(&BookingModel{}).
		Where("companion_id = ? AND status = ?",
			companionID.String(), string(vo.StatusPending)).
		Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}

func (r *BookingRepositoryImpl) HasOverlappingBooking(
	ctx context.Context,
	actorID string,
	isCompanion bool,
	statuses []vo.BookingStatus,
	startTime, endTime time.Time,
) (bool, error) {
	var count int64
	query := r.getDB(ctx).Model(&BookingModel{})

	if isCompanion {
		query = query.Where("companion_id = ?", actorID)
	} else {
		query = query.Where("client_id = ?", actorID)
	}

	statusStrings := make([]string, len(statuses))
	for i, s := range statuses {
		statusStrings[i] = string(s)
	}

	query = query.Where("status IN (?)", statusStrings)
	query = query.Where("start_time < ? AND end_time > ?", endTime, startTime)

	err := query.Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *BookingRepositoryImpl) FindByFilters(
	ctx context.Context,
	filters repository.BookingFilters,
) ([]*aggregate.Booking, int64, error) {
	query := r.getDB(ctx).Model(&BookingModel{})

	if filters.ClientID != nil {
		query = query.Where("client_id = ?", *filters.ClientID)
	}
	if filters.CompanionID != nil {
		query = query.Where("companion_id = ?", *filters.CompanionID)
	}
	if len(filters.Statuses) > 0 {
		query = query.Where("status IN (?)", filters.Statuses)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (filters.Page - 1) * filters.PageSize
	var models []BookingModel
	if err := query.Order("created_at DESC").Offset(int(offset)).Limit(int(filters.PageSize)).Find(&models).Error; err != nil {
		return nil, 0, err
	}

	bookings := make([]*aggregate.Booking, 0, len(models))
	for i := range models {
		b, err := models[i].ToDomain()
		if err != nil {
			return nil, 0, err
		}
		bookings = append(bookings, b)
	}

	return bookings, total, nil
}

// FindAcceptedBookingsPastEndTimeBuffer finds all ACCEPTED bookings past end_time + buffer.
func (r *BookingRepositoryImpl) FindAcceptedBookingsPastEndTimeBuffer(
	ctx context.Context,
	now time.Time,
	buffer time.Duration,
) ([]*aggregate.Booking, error) {
	threshold := now.Add(-buffer)
	var models []BookingModel
	err := r.getDB(ctx).
		Where("status = ? AND end_time <= ?", string(vo.StatusAccepted), threshold).
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	bookings := make([]*aggregate.Booking, 0, len(models))
	for i := range models {
		b, err := models[i].ToDomain()
		if err != nil {
			return nil, err
		}
		bookings = append(bookings, b)
	}

	return bookings, nil
}

// FindPendingBookingsEligibleForTimeout finds all PENDING bookings that have timed out.
// Timed out means either created > 12 hours ago OR start_time <= now + 1 hour (equivalent to start_time - 1h <= now).
func (r *BookingRepositoryImpl) FindPendingBookingsEligibleForTimeout(
	ctx context.Context,
	now time.Time,
) ([]*aggregate.Booking, error) {
	createdAtThreshold := now.Add(-12 * time.Hour)
	startTimeThreshold := now.Add(1 * time.Hour)

	var models []BookingModel
	err := r.getDB(ctx).
		Where("status = ? AND (created_at <= ? OR start_time <= ?)",
			string(vo.StatusPending), createdAtThreshold, startTimeThreshold).
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	bookings := make([]*aggregate.Booking, 0, len(models))
	for i := range models {
		b, err := models[i].ToDomain()
		if err != nil {
			return nil, err
		}
		bookings = append(bookings, b)
	}

	return bookings, nil
}
