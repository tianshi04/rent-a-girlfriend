package worker

import (
	"context"
	"log"
	"time"

	"github.com/rent-a-girlfriend/booking-service/internal/application/command"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/repository"
)

// AutoCompleteWorker automatically completes accepted bookings that are past their end_time + buffer (12 hours).
type AutoCompleteWorker struct {
	repo            repository.BookingRepository
	completeHandler *command.SystemCompleteBookingHandler
	pollingInterval time.Duration
	bufferTime      time.Duration
}

// NewAutoCompleteWorker creates a new AutoCompleteWorker instance.
func NewAutoCompleteWorker(
	repo repository.BookingRepository,
	completeHandler *command.SystemCompleteBookingHandler,
	pollingInterval time.Duration,
	bufferTime time.Duration,
) *AutoCompleteWorker {
	return &AutoCompleteWorker{
		repo:            repo,
		completeHandler: completeHandler,
		pollingInterval: pollingInterval,
		bufferTime:      bufferTime,
	}
}

// Start begins the auto-completion loop. Blocks until ctx is cancelled.
func (w *AutoCompleteWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.pollingInterval)
	defer ticker.Stop()

	log.Printf("[AUTO-COMPLETE-WORKER] Started, polling every %v, bufferTime=%v", w.pollingInterval, w.bufferTime)

	for {
		select {
		case <-ctx.Done():
			log.Println("[AUTO-COMPLETE-WORKER] Stopping...")
			return
		case <-ticker.C:
			w.runProcess(ctx)
		}
	}
}

func (w *AutoCompleteWorker) runProcess(ctx context.Context) {
	now := time.Now()
	// Find all accepted bookings past their end_time + buffer (12 hours)
	bookings, err := w.repo.FindAcceptedBookingsPastEndTimeBuffer(ctx, now, w.bufferTime)
	if err != nil {
		log.Printf("[AUTO-COMPLETE-WORKER] Failed to query bookings for auto-completion: %v", err)
		return
	}

	if len(bookings) == 0 {
		return
	}

	log.Printf("[AUTO-COMPLETE-WORKER] Found %d bookings eligible for auto-completion", len(bookings))

	for _, b := range bookings {
		cmd := command.SystemCompleteBookingCmd{
			BookingID: b.ID().String(),
		}
		_, err := w.completeHandler.Handle(ctx, cmd)
		if err != nil {
			log.Printf("[AUTO-COMPLETE-WORKER] Failed to auto-complete booking id=%s: %v", b.ID().String(), err)
			continue
		}
		log.Printf("[AUTO-COMPLETE-WORKER] Successfully auto-completed booking id=%s", b.ID().String())
	}
}
