package port

import (
	"context"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/event"
)

// EventPublisher is the port for publishing domain and integration events.
type EventPublisher interface {
	Publish(ctx context.Context, evt event.DomainEvent) error
}
