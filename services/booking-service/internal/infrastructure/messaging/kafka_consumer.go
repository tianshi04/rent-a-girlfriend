package messaging

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
	"gorm.io/gorm"

	"github.com/rent-a-girlfriend/booking-service/internal/application/command"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
	"github.com/rent-a-girlfriend/booking-service/internal/infrastructure/persistence"
)

// inboundEvent is the envelope expected from finance-events and interaction-events topics.
type inboundEvent struct {
	SpecVersion string          `json:"specversion"`
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	Data        json.RawMessage `json:"data"`
}

type bookingIDPayload struct {
	BookingID string `json:"bookingId"`
}

// KafkaConsumer listens to finance-events and interaction-events topics and
// routes received CloudEvents to the SagaCoordinator.
type KafkaConsumer struct {
	coordinators   *command.SagaCoordinator
	disputeHandler *command.DisputeBookingHandler
	resolveHandler *command.ResolveBookingHandler
	db             *gorm.DB
	readers        []*kafka.Reader
}

// NewKafkaConsumer creates readers for the given topics and wires to the coordinator.
func NewKafkaConsumer(
	cfg KafkaConnConfig,
	groupID string,
	topics []string,
	coordinator *command.SagaCoordinator,
	disputeHandler *command.DisputeBookingHandler,
	resolveHandler *command.ResolveBookingHandler,
	db *gorm.DB,
) *KafkaConsumer {
	brokerList := strings.Split(cfg.Brokers, ",")
	dialer := cfg.GetDialer()

	readers := make([]*kafka.Reader, 0, len(topics))
	for _, topic := range topics {
		r := kafka.NewReader(kafka.ReaderConfig{
			Brokers:     brokerList,
			GroupID:     groupID + "-" + topic,
			Topic:       topic,
			MinBytes:    1,
			MaxBytes:    10e6, // 10 MB
			Dialer:      dialer,
			StartOffset: kafka.FirstOffset,
		})
		readers = append(readers, r)
	}
	return &KafkaConsumer{
		coordinators:   coordinator,
		disputeHandler: disputeHandler,
		resolveHandler: resolveHandler,
		db:             db,
		readers:        readers,
	}
}

// Start begins consuming all registered topics concurrently. Blocks until ctx is cancelled.
func (c *KafkaConsumer) Start(ctx context.Context) {
	for _, r := range c.readers {
		go c.consume(ctx, r)
	}
	<-ctx.Done()
	log.Println("[KAFKA-CONSUMER] Stopping all readers...")
	for _, r := range c.readers {
		_ = r.Close()
	}
}

func (c *KafkaConsumer) consume(ctx context.Context, r *kafka.Reader) {
	log.Printf("[KAFKA-CONSUMER] Started consuming topic=%s group=%s", r.Config().Topic, r.Config().GroupID)
	for {
		// Fetch message without committing offset immediately
		msg, err := r.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return // context cancelled
			}
			log.Printf("[KAFKA-CONSUMER] Error fetching from topic=%s: %v", r.Config().Topic, err)
			continue
		}

		// Dispatch and process event with a retry loop for transient database/locking errors
		maxRetries := 3
		backoff := 1 * time.Second
		var dispatchErr error

		for attempt := 1; attempt <= maxRetries; attempt++ {
			dispatchErr = c.dispatch(ctx, msg)
			if dispatchErr == nil {
				break
			}

			log.Printf("[KAFKA-CONSUMER] Dispatch failed (attempt %d/%d) on topic=%s offset=%d: %v",
				attempt, maxRetries, r.Config().Topic, msg.Offset, dispatchErr)

			if attempt < maxRetries {
				select {
				case <-ctx.Done():
					return
				case <-time.After(backoff):
					backoff *= 2 // Exponential backoff: 1s, 2s, 4s
				}
			}
		}

		if dispatchErr != nil {
			// CRITICAL: We failed to process the message even after retries.
			// To enforce at-least-once processing, we DO NOT commit the offset!
			// This will halt consumer offset progress or allow reprocessing upon consumer restart.
			log.Printf("[KAFKA-CONSUMER] CRITICAL: Dispatch failed after all retries on topic=%s offset=%d: %v. Hiding offset commit to force reprocessing later.",
				r.Config().Topic, msg.Offset, dispatchErr)
			continue
		}

		// Commit offset only after successful processing
		if err := r.CommitMessages(ctx, msg); err != nil {
			log.Printf("[KAFKA-CONSUMER] Failed to commit message offset on topic=%s offset=%d: %v",
				r.Config().Topic, msg.Offset, err)
		}
	}
}

func (c *KafkaConsumer) dispatch(ctx context.Context, msg kafka.Message) error {
	var ce inboundEvent
	if err := json.Unmarshal(msg.Value, &ce); err != nil {
		log.Printf("[KAFKA-CONSUMER] Failed to parse CloudEvent: %v", err)
		return nil // skip malformed messages
	}

	var payload bookingIDPayload
	if err := json.Unmarshal(ce.Data, &payload); err != nil {
		log.Printf("[KAFKA-CONSUMER] Failed to parse bookingId from event type=%s: %v", ce.Type, err)
		return nil
	}

	bookingID := payload.BookingID
	log.Printf("[KAFKA-CONSUMER] Routing event type=%s bookingId=%s", ce.Type, bookingID)

	switch ce.Type {
	// Finance events
	case "finance.coin-escrowed.v1":
		return c.coordinators.HandleEscrowSuccess(ctx, bookingID, ce.ID)
	case "finance.escrow-failed.v1":
		return c.coordinators.HandleEscrowFailed(ctx, bookingID, ce.ID)
	case "finance.refund-success.v1":
		return c.coordinators.HandleRefundSuccess(ctx, bookingID, ce.ID)
	case "finance.refund-failed.v1":
		return c.coordinators.HandleRefundFailed(ctx, bookingID, ce.ID)
	// Interaction events
	case "interaction.chat-room-created.v1":
		return c.coordinators.HandleChatRoomCreated(ctx, bookingID, ce.ID)
	case "interaction.chat-room-creation-failed.v1":
		return c.coordinators.HandleChatRoomFailed(ctx, bookingID, ce.ID)
	// Dispute events
	case "dispute.dispute-created.v1":
		err := c.db.Transaction(func(tx *gorm.DB) error {
			txCtx := context.WithValue(ctx, vo.TxKey, tx)
			alreadyProcessed, err := persistence.CheckAndRecordEvent(txCtx, tx, ce.ID, ce.Type)
			if err != nil {
				return err
			}
			if alreadyProcessed {
				log.Printf("[KAFKA-CONSUMER] Skipping duplicate event id=%s type=%s", ce.ID, ce.Type)
				return nil
			}
			_, err = c.disputeHandler.Handle(txCtx, command.DisputeBookingCmd{BookingID: bookingID})
			return err
		})
		return err
	case "dispute.dispute-resolved.v1":
		err := c.db.Transaction(func(tx *gorm.DB) error {
			txCtx := context.WithValue(ctx, vo.TxKey, tx)
			alreadyProcessed, err := persistence.CheckAndRecordEvent(txCtx, tx, ce.ID, ce.Type)
			if err != nil {
				return err
			}
			if alreadyProcessed {
				log.Printf("[KAFKA-CONSUMER] Skipping duplicate event id=%s type=%s", ce.ID, ce.Type)
				return nil
			}
			_, err = c.resolveHandler.Handle(txCtx, command.ResolveBookingCmd{BookingID: bookingID})
			return err
		})
		return err
	default:
		log.Printf("[KAFKA-CONSUMER] Unrecognised event type=%s, ignoring", ce.Type)
	}
	return nil
}
