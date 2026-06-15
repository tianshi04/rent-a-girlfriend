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

// MessagePublisher is the interface for publishing CloudEvents to a topic.
type MessagePublisher interface {
	PublishEvent(ctx context.Context, topic string, key string, event cloudevents.Event) error
}

// KafkaPublisher implements port.EventPublisher and MessagePublisher via kafka-go.
type KafkaPublisher struct {
	writer *kafka.Writer
	topic  string
}

func NewKafkaPublisher(cfg KafkaConnConfig, topic string) *KafkaPublisher {
	brokerList := strings.Split(cfg.Brokers, ",")
	dialer := cfg.GetDialer()

	transport := &kafka.Transport{
		Dial: dialer.DialFunc,
		TLS:  dialer.TLS,
		SASL: dialer.SASLMechanism,
	}

	writer := &kafka.Writer{
		Addr:                   kafka.TCP(brokerList...),
		Balancer:               &kafka.LeastBytes{},
		AllowAutoTopicCreation: true,
		Async:                  false,
		Transport:              transport,
	}
	return &KafkaPublisher{writer: writer, topic: topic}
}

func (p *KafkaPublisher) PublishEvent(ctx context.Context, topic string, key string, event cloudevents.Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal cloud event: %w", err)
	}

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: payload,
	})
	if err != nil {
		return fmt.Errorf("failed to publish to kafka topic %s: %w", topic, err)
	}

	log.Printf("[KAFKA] Published event type=%s id=%s topic=%s key=%s", event.Type(), event.ID(), topic, key)
	return nil
}

func (p *KafkaPublisher) Close() error {
	return p.writer.Close()
}
