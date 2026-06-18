package command

import (
	"context"
	"time"

	"github.com/rent-a-girlfriend/booking-service/internal/application/port"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/aggregate"
	domainerr "github.com/rent-a-girlfriend/booking-service/internal/domain/errors"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/repository"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
)

// MaxPendingPerCompanion is the cap for INV-B04.
const MaxPendingPerCompanion = 10

// RequestBookingCmd holds the data for creating a new booking.
type RequestBookingCmd struct {
	ClientID    string
	CompanionID string
	ScenarioID  string
	StartTime   time.Time
}

// RequestBookingHandler handles the RequestBooking command.
type RequestBookingHandler struct {
	repo           repository.BookingRepository
	profileService port.ProfileService
	financeService port.FinanceService
}

func NewRequestBookingHandler(
	repo repository.BookingRepository,
	profileService port.ProfileService,
	financeService port.FinanceService,
) *RequestBookingHandler {
	return &RequestBookingHandler{
		repo:           repo,
		profileService: profileService,
		financeService: financeService,
	}
}

func (h *RequestBookingHandler) Handle(ctx context.Context, cmd RequestBookingCmd) (*aggregate.Booking, error) {
	now := time.Now()

	clientID, err := vo.NewClientID(cmd.ClientID)
	if err != nil {
		return nil, err
	}
	companionID, err := vo.NewCompanionID(cmd.CompanionID)
	if err != nil {
		return nil, err
	}

	// 1. Check Pending Cap [INV-B04]
	pendingCount, err := h.repo.CountPendingByCompanion(ctx, companionID)
	if err != nil {
		return nil, err
	}
	if pendingCount >= MaxPendingPerCompanion {
		return nil, domainerr.ErrPendingCapExceeded
	}

	// 2. Fetch scenario snapshot from Profile Service (sync call)
	snapshot, err := h.profileService.GetScenarioSnapshot(ctx, cmd.ScenarioID)
	if err != nil {
		return nil, err
	}

	// 3. Build TimeRange (INV-B02: EndTime = StartTime + Duration)
	endTime := cmd.StartTime.Add(time.Duration(snapshot.DurationMinutes()) * time.Minute)
	timeRange, err := vo.NewTimeRange(cmd.StartTime, endTime)
	if err != nil {
		return nil, err
	}

	// 3.5. Check for overlapping bookings
	// Client cannot have any booking in PENDING or ACCEPTED status that overlaps in this time range
	clientOverlap, err := h.repo.HasOverlappingBooking(ctx, clientID.String(), false, []vo.BookingStatus{vo.StatusPending, vo.StatusAccepted}, timeRange.StartTime(), timeRange.EndTime())
	if err != nil {
		return nil, err
	}
	if clientOverlap {
		return nil, domainerr.ErrClientBookingOverlap
	}

	// Companion cannot have any booking in ACCEPTED status that overlaps in this time range
	companionOverlap, err := h.repo.HasOverlappingBooking(ctx, companionID.String(), true, []vo.BookingStatus{vo.StatusAccepted}, timeRange.StartTime(), timeRange.EndTime())
	if err != nil {
		return nil, err
	}
	if companionOverlap {
		return nil, domainerr.ErrCompanionBookingOverlap
	}

	// 5. Create Booking aggregate (validates INV-B01, INV-B02)
	booking, err := aggregate.NewBooking(clientID, companionID, *snapshot, timeRange, now)
	if err != nil {
		return nil, err
	}

	// 6. Persist
	if err := h.repo.Save(ctx, booking); err != nil {
		return nil, err
	}

	return booking, nil
}
