package messaging

import (
	"crypto/tls"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
)

// KafkaConnConfig holds the connection details required to connect to local or cloud Kafka.
type KafkaConnConfig struct {
	Brokers       string
	SASLUsername  string
	SASLPassword  string
	SASLMechanism string
	TLSEnabled    bool
}

// GetDialer builds a kafka.Dialer configured with SASL and TLS if provided.
func (c KafkaConnConfig) GetDialer() *kafka.Dialer {
	dialer := &kafka.Dialer{
		Timeout:   10 * time.Second,
		DualStack: true,
	}

	if c.TLSEnabled {
		dialer.TLS = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	if c.SASLUsername != "" && c.SASLPassword != "" {
		if c.SASLMechanism == "SCRAM-SHA-256" || c.SASLMechanism == "SCRAM-SHA-512" {
			algo := scram.SHA256
			if c.SASLMechanism == "SCRAM-SHA-512" {
				algo = scram.SHA512
			}
			mechanism, err := scram.Mechanism(algo, c.SASLUsername, c.SASLPassword)
			if err == nil {
				dialer.SASLMechanism = mechanism
			}
		} else {
			// Default to PLAIN
			dialer.SASLMechanism = plain.Mechanism{
				Username: c.SASLUsername,
				Password: c.SASLPassword,
			}
		}
	}

	return dialer
}
