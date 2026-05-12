package port

import (
	"context"

	"github.com/rent-a-girlfriend/identity-service/internal/domain/event"
)

// EventPublisher is the port for publishing domain events via Transactional Outbox.
type EventPublisher interface {
	// Publish writes a domain event to the outbox table within the current transaction.
	Publish(ctx context.Context, evt event.DomainEvent) error
}
