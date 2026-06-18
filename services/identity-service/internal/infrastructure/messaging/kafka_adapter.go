package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/segmentio/kafka-go"
)

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

func (a *KafkaAdapter) PublishEvent(ctx context.Context, topic string, key string, event cloudevents.Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal cloud event: %w", err)
	}

	err = a.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: payload,
	})

	if err != nil {
		return fmt.Errorf("failed to write message to kafka: %w", err)
	}

	log.Printf("[KAFKA] Published event %s to topic %s with key %s", event.ID(), topic, key)
	return nil
}

func (a *KafkaAdapter) Close() error {
	return a.writer.Close()
}
