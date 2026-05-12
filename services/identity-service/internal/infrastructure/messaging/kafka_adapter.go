package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

// CloudEvent represents the standard envelope for all domain events.
type CloudEvent struct {
	SpecVersion     string      `json:"specversion"`
	ID              string      `json:"id"`
	Source          string      `json:"source"`
	Type            string      `json:"type"`
	DataContentType string      `json:"datacontenttype"`
	Time            time.Time   `json:"time"`
	Data            interface{} `json:"data"`
	Extensions      interface{} `json:"extensions,omitempty"`
}

type KafkaAdapter struct {
	writer *kafka.Writer
}

func NewKafkaAdapter(brokers string) *KafkaAdapter {
	brokerList := strings.Split(brokers, ",")
	
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(brokerList...),
		Balancer:               &kafka.LeastBytes{},
		AllowAutoTopicCreation: true,
		Async:                  false, // We want reliability in the worker
	}

	return &KafkaAdapter{writer: writer}
}

func (a *KafkaAdapter) PublishEvent(ctx context.Context, topic string, event CloudEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal cloud event: %w", err)
	}

	err = a.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   []byte(event.ID),
		Value: payload,
	})

	if err != nil {
		return fmt.Errorf("failed to write message to kafka: %w", err)
	}

	log.Printf("[KAFKA] Published event %s to topic %s", event.ID, topic)
	return nil
}

func (a *KafkaAdapter) Close() error {
	return a.writer.Close()
}
