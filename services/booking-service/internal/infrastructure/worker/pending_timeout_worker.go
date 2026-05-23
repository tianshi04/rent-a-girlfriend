package worker

import (
	"context"
	"log"
	"time"

	"github.com/rent-a-girlfriend/booking-service/internal/application/command"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/repository"
)

// PendingTimeoutWorker automatically rejects pending bookings that have timed out.
type PendingTimeoutWorker struct {
	repo            repository.BookingRepository
	rejectHandler   *command.SystemRejectBookingHandler
	pollingInterval time.Duration
}

// NewPendingTimeoutWorker creates a new PendingTimeoutWorker instance.
func NewPendingTimeoutWorker(
	repo repository.BookingRepository,
	rejectHandler *command.SystemRejectBookingHandler,
	pollingInterval time.Duration,
) *PendingTimeoutWorker {
	return &PendingTimeoutWorker{
		repo:            repo,
		rejectHandler:   rejectHandler,
		pollingInterval: pollingInterval,
	}
}

// Start begins the auto-rejection loop. Blocks until ctx is cancelled.
func (w *PendingTimeoutWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.pollingInterval)
	defer ticker.Stop()

	log.Printf("[PENDING-TIMEOUT-WORKER] Started, polling every %v", w.pollingInterval)

	for {
		select {
		case <-ctx.Done():
			log.Println("[PENDING-TIMEOUT-WORKER] Stopping...")
			return
		case <-ticker.C:
			w.runProcess(ctx)
		}
	}
}

func (w *PendingTimeoutWorker) runProcess(ctx context.Context) {
	now := time.Now()
	// Find all pending bookings eligible for timeout
	bookings, err := w.repo.FindPendingBookingsEligibleForTimeout(ctx, now)
	if err != nil {
		log.Printf("[PENDING-TIMEOUT-WORKER] Failed to query bookings for timeout: %v", err)
		return
	}

	if len(bookings) == 0 {
		return
	}

	log.Printf("[PENDING-TIMEOUT-WORKER] Found %d bookings eligible for timeout rejection", len(bookings))

	for _, b := range bookings {
		cmd := command.SystemRejectBookingCmd{
			BookingID: b.ID().String(),
		}
		_, err := w.rejectHandler.Handle(ctx, cmd)
		if err != nil {
			log.Printf("[PENDING-TIMEOUT-WORKER] Failed to timeout booking id=%s: %v", b.ID().String(), err)
			continue
		}
		log.Printf("[PENDING-TIMEOUT-WORKER] Successfully timed out booking id=%s", b.ID().String())
	}
}
