package messaging

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/segmentio/kafka-go"
)

func TestKafkaConsumer_Dispatch_ValidBookingID(t *testing.T) {
	consumer := &KafkaConsumer{}

	dataBytes, _ := json.Marshal(map[string]string{
		"booking_id": "550e8400-e29b-41d4-a716-446655440000",
	})
	ce := map[string]interface{}{
		"specversion": "1.0",
		"id":          "event-123",
		"source":      "test",
		"type":        "unknown-type", // hits default case
		"data":        json.RawMessage(dataBytes),
	}
	ceBytes, _ := json.Marshal(ce)

	msg := kafka.Message{
		Value: ceBytes,
	}

	err := consumer.dispatch(context.Background(), msg)
	if err != nil {
		t.Errorf("expected nil error for unrecognized event type with valid booking_id, got: %v", err)
	}
}

func TestKafkaConsumer_Dispatch_MissingBookingID(t *testing.T) {
	consumer := &KafkaConsumer{}

	dataBytes, _ := json.Marshal(map[string]string{
		"some_other_field": "some-value",
	})
	ce := map[string]interface{}{
		"specversion": "1.0",
		"id":          "event-123",
		"source":      "test",
		"type":        "finance.escrow-created.v1",
		"data":        json.RawMessage(dataBytes),
	}
	ceBytes, _ := json.Marshal(ce)

	msg := kafka.Message{
		Value: ceBytes,
	}

	err := consumer.dispatch(context.Background(), msg)
	if err != nil {
		t.Errorf("expected nil error for missing booking_id (should be skipped), got: %v", err)
	}
}

func TestKafkaConsumer_Dispatch_InvalidBookingID(t *testing.T) {
	consumer := &KafkaConsumer{}

	dataBytes, _ := json.Marshal(map[string]string{
		"booking_id": "not-a-uuid",
	})
	ce := map[string]interface{}{
		"specversion": "1.0",
		"id":          "event-123",
		"source":      "test",
		"type":        "finance.escrow-created.v1",
		"data":        json.RawMessage(dataBytes),
	}
	ceBytes, _ := json.Marshal(ce)

	msg := kafka.Message{
		Value: ceBytes,
	}

	err := consumer.dispatch(context.Background(), msg)
	if err != nil {
		t.Errorf("expected nil error for invalid booking_id format (should be skipped), got: %v", err)
	}
}
