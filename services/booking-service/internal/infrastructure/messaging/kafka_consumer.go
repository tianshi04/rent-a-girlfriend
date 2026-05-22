package messaging

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/segmentio/kafka-go"
	"gorm.io/gorm"

	"github.com/rent-a-girlfriend/booking-service/internal/application/command"
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
		msg, err := r.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return // context cancelled
			}
			log.Printf("[KAFKA-CONSUMER] Error reading from topic=%s: %v", r.Config().Topic, err)
			continue
		}

		if err := c.dispatch(ctx, msg); err != nil {
			log.Printf("[KAFKA-CONSUMER] Failed to dispatch event from topic=%s offset=%d: %v",
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

	// Idempotency check: skip if already processed
	alreadyProcessed, err := persistence.CheckAndRecordEvent(ctx, c.db, ce.ID, ce.Type)
	if err != nil {
		return err
	}
	if alreadyProcessed {
		log.Printf("[KAFKA-CONSUMER] Skipping duplicate event id=%s type=%s", ce.ID, ce.Type)
		return nil
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
	case "com.rentagf.finance.CoinEscrowed.v1":
		return c.coordinators.HandleEscrowSuccess(ctx, bookingID)
	case "com.rentagf.finance.EscrowFailed.v1":
		return c.coordinators.HandleEscrowFailed(ctx, bookingID)
	case "com.rentagf.finance.RefundSuccess.v1":
		return c.coordinators.HandleRefundSuccess(ctx, bookingID)
	case "com.rentagf.finance.RefundFailed.v1":
		return c.coordinators.HandleRefundFailed(ctx, bookingID)
	// Interaction events
	case "com.rentagf.interaction.ChatRoomCreated.v1":
		return c.coordinators.HandleChatRoomCreated(ctx, bookingID)
	case "com.rentagf.interaction.ChatRoomCreationFailed.v1":
		return c.coordinators.HandleChatRoomFailed(ctx, bookingID)
	// Dispute events
	case "com.rentagf.dispute.DisputeCreated.v1":
		_, err := c.disputeHandler.Handle(ctx, command.DisputeBookingCmd{BookingID: bookingID})
		return err
	default:
		log.Printf("[KAFKA-CONSUMER] Unrecognised event type=%s, ignoring", ce.Type)
	}
	return nil
}
