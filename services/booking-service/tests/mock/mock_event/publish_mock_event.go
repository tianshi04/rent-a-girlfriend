package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

type cloudEvent struct {
	SpecVersion     string          `json:"specversion"`
	ID              string          `json:"id"`
	Source          string          `json:"source"`
	Type            string          `json:"type"`
	DataContentType string          `json:"datacontenttype"`
	Time            time.Time       `json:"time"`
	Data            json.RawMessage `json:"data"`
}

type bookingPayload struct {
	BookingID string `json:"booking_id"`
}

// Payload structs for commands from Booking Service
type TransferToEscrowCmdPayload struct {
	BookingID   string `json:"bookingId"`
	ClientID    string `json:"clientId"`
	CompanionID string `json:"companionId"`
	Amount      int64  `json:"amount"`
}

type CreateChatRoomCmdPayload struct {
	BookingID   string `json:"bookingId"`
	ClientID    string `json:"clientId"`
	CompanionID string `json:"companionId"`
}

type RefundEscrowCmdPayload struct {
	BookingID   string `json:"bookingId"`
	ClientID    string `json:"clientId"`
	CompanionID string `json:"companionId"`
	Amount      int64  `json:"amount"`
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	fmt.Println("==========================================================")
	fmt.Println("       RENT-A-GIRLFRIEND SAGA INTERACTIVE TEST DRIVER      ")
	fmt.Println("==========================================================")

	brokers := getKafkaBrokers()
	fmt.Printf("[INFO] Using Kafka Brokers: %s\n\n", brokers)

	fmt.Println("Select your mode:")
	fmt.Println("  [1] Interactive SAGA Driver (Listen to Booking requests and reply on-the-fly)")
	fmt.Println("  [2] Manual Event Publisher (Directly shoot a single mock event to a topic)")

	reader := bufio.NewReader(os.Stdin)
	var mode int
	for {
		fmt.Print("\nEnter mode (1-2): ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		_, err := fmt.Sscanf(input, "%d", &mode)
		if err == nil && (mode == 1 || mode == 2) {
			break
		}
		fmt.Println("[ERROR] Invalid choice. Please enter 1 or 2.")
	}

	if mode == 1 {
		runInteractiveDriver(brokers, reader)
	} else {
		runManualPublisher(brokers, reader)
	}
}

// =========================================================================
// MODE 1: Interactive SAGA Driver
// =========================================================================
func runInteractiveDriver(brokers string, reader *bufio.Reader) {
	brokerList := strings.Split(brokers, ",")

	// Create unique group ID to ensure we only get latest events in this run
	groupID := "interactive-driver-group-" + uuid.New().String()
	topic := "booking.events"

	fmt.Printf("\n[DRIVER] Subscribing to topic '%s' using GroupID '%s'...\n", topic, groupID)
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokerList,
		GroupID:     groupID,
		Topic:       topic,
		MinBytes:    1,
		MaxBytes:    10e6,
		StartOffset: kafka.LastOffset, // Only listen to new events
	})
	defer func() { _ = r.Close() }()

	fmt.Println("[DRIVER] 🟢 Active and listening! Accept a booking in your app to kick off Saga...")
	fmt.Println("--------------------------------------------------------------------------------")

	ctx := context.Background()
	for {
		msg, err := r.ReadMessage(ctx)
		if err != nil {
			log.Printf("[ERROR] Failed to read message: %v", err)
			continue
		}

		var ce cloudEvent
		if err := json.Unmarshal(msg.Value, &ce); err != nil {
			continue // ignore malformed CloudEvents
		}

		switch ce.Type {
		case "finance.transfer-to-escrow.v1":
			var payload TransferToEscrowCmdPayload
			if err := json.Unmarshal(ce.Data, &payload); err != nil {
				log.Printf("[ERROR] Failed to parse TransferToEscrow data: %v", err)
				continue
			}
			fmt.Printf("\n📥 [RECEIVED] TransferToEscrow command (Saga Step 1)\n")
			fmt.Printf("   BookingID:   %s\n", payload.BookingID)
			fmt.Printf("   ClientID:    %s\n", payload.ClientID)
			fmt.Printf("   CompanionID: %s\n", payload.CompanionID)
			fmt.Printf("   Amount:      %d Kano-Coins\n", payload.Amount)

			fmt.Println("\nChoose how to respond to this Escrow request:")
			fmt.Println("  [1] SUCCESS (Client has sufficient funds -> Send CoinEscrowed)")
			fmt.Println("  [2] FAILURE (Client has insufficient funds -> Send EscrowFailed)")

			choice := getChoice(reader, 1, 2)
			if choice == 1 {
				publishResponseEvent(brokers, "finance.events", "finance.escrow-created.v1", payload.BookingID)
			} else {
				publishResponseEvent(brokers, "finance.events", "finance.escrow-failed.v1", payload.BookingID)
			}
			fmt.Println("\n--- Back to listening... ---")

		case "interaction.create-chat-room.v1":
			var payload CreateChatRoomCmdPayload
			if err := json.Unmarshal(ce.Data, &payload); err != nil {
				log.Printf("[ERROR] Failed to parse CreateChatRoom data: %v", err)
				continue
			}
			fmt.Printf("\n📥 [RECEIVED] CreateChatRoom command (Saga Step 2)\n")
			fmt.Printf("   BookingID:   %s\n", payload.BookingID)
			fmt.Printf("   ClientID:    %s\n", payload.ClientID)
			fmt.Printf("   CompanionID: %s\n", payload.CompanionID)

			fmt.Println("\nChoose how to respond to this Chat Room request:")
			fmt.Println("  [1] SUCCESS (Chat Room successfully initialized -> Send ChatRoomCreated)")
			fmt.Println("  [2] FAILURE (Room creation failed/companion overlap -> Send ChatRoomCreationFailed)")

			choice := getChoice(reader, 1, 2)
			if choice == 1 {
				publishResponseEvent(brokers, "interaction.events", "interaction.chat-room-created.v1", payload.BookingID)
			} else {
				publishResponseEvent(brokers, "interaction.events", "interaction.chat-room-creation-failed.v1", payload.BookingID)
			}
			fmt.Println("\n--- Back to listening... ---")

		case "finance.refund-escrow.v1":
			var payload RefundEscrowCmdPayload
			if err := json.Unmarshal(ce.Data, &payload); err != nil {
				log.Printf("[ERROR] Failed to parse RefundEscrow data: %v", err)
				continue
			}
			fmt.Printf("\n📥 [RECEIVED] RefundEscrow compensation command (Saga Compensating Action)\n")
			fmt.Printf("   BookingID:   %s\n", payload.BookingID)
			fmt.Printf("   ClientID:    %s\n", payload.ClientID)
			fmt.Printf("   CompanionID: %s\n", payload.CompanionID)
			fmt.Printf("   Amount:      %d Kano-Coins\n", payload.Amount)

			fmt.Println("\nChoose how to respond to this Refund request:")
			fmt.Println("  [1] SUCCESS (Refund successfully executed -> Send RefundSuccess)")
			fmt.Println("  [2] FAILURE (System failure on refund -> Send RefundFailed - ALERT)")

			choice := getChoice(reader, 1, 2)
			if choice == 1 {
				publishResponseEvent(brokers, "finance.events", "finance.escrow-refunded.v1", payload.BookingID)
			} else {
				publishResponseEvent(brokers, "finance.events", "finance.refund-failed.v1", payload.BookingID)
			}
			fmt.Println("\n--- Back to listening... ---")
		}
	}
}

// =========================================================================
// MODE 2: Manual Event Publisher
// =========================================================================
func runManualPublisher(brokers string, reader *bufio.Reader) {
	fmt.Println("\nSelect the type of event you want to publish:")
	options := []struct {
		Topic     string
		EventType string
		Label     string
	}{
		{Topic: "finance.events", EventType: "finance.escrow-created.v1", Label: "EscrowCreated (Saga Step 1: Client Escrow Success)"},
		{Topic: "finance.events", EventType: "finance.escrow-failed.v1", Label: "EscrowFailed (Saga Step 1: Client Escrow Failed)"},
		{Topic: "interaction.events", EventType: "interaction.chat-room-created.v1", Label: "ChatRoomCreated (Saga Step 2: Chat Room Created Success)"},
		{Topic: "interaction.events", EventType: "interaction.chat-room-creation-failed.v1", Label: "ChatRoomCreationFailed (Saga Step 2: Chat Room Failed)"},
		{Topic: "finance.events", EventType: "finance.escrow-refunded.v1", Label: "EscrowRefunded (Saga Compensation: Escrow Refund Success)"},
		{Topic: "finance.events", EventType: "finance.refund-failed.v1", Label: "RefundFailed (Saga Compensation: Escrow Refund Failed - ALERT)"},
		{Topic: "dispute.events", EventType: "dispute.dispute-created.v1", Label: "DisputeCreated (User Flow: Client opens a Dispute)"},
	}

	for i, opt := range options {
		fmt.Printf("  [%d] %s (Topic: %s)\n", i+1, opt.Label, opt.Topic)
	}

	choice := getChoice(reader, 1, 7)
	selectedOpt := options[choice-1]

	var bookingID string
	for {
		fmt.Print("Enter Booking ID (UUID): ")
		input, _ := reader.ReadString('\n')
		bookingID = strings.TrimSpace(input)
		if _, err := uuid.Parse(bookingID); err == nil {
			break
		}
		fmt.Println("[ERROR] Invalid UUID format. Please try again.")
	}

	publishResponseEvent(brokers, selectedOpt.Topic, selectedOpt.EventType, bookingID)
}

// =========================================================================
// Helpers
// =========================================================================
func getChoice(reader *bufio.Reader, min, max int) int {
	var val int
	for {
		fmt.Printf("Enter choice (%d-%d): ", min, max)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		_, err := fmt.Sscanf(input, "%d", &val)
		if err == nil && val >= min && val <= max {
			return val
		}
		fmt.Printf("[ERROR] Invalid choice. Please enter a number between %d and %d.\n", min, max)
	}
}

func publishResponseEvent(brokers string, topic string, eventType string, bookingID string) {
	fmt.Printf("\n[PUBLISHER] Preparing CloudEvent...\n")

	payload := bookingPayload{
		BookingID: bookingID,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Fatalf("[FATAL] Failed to marshal payload: %v", err)
	}

	eventID := uuid.New().String()
	ce := cloudEvent{
		SpecVersion:     "1.0",
		ID:              eventID,
		Source:          "/mock-services/saga-driver",
		Type:            eventType,
		DataContentType: "application/json",
		Time:            time.Now(),
		Data:            payloadBytes,
	}

	eventBytes, err := json.Marshal(ce)
	if err != nil {
		log.Fatalf("[FATAL] Failed to marshal CloudEvent: %v", err)
	}

	brokerList := strings.Split(brokers, ",")
	w := &kafka.Writer{
		Addr:     kafka.TCP(brokerList...),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}
	defer func() { _ = w.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = w.WriteMessages(ctx, kafka.Message{
		Key:   []byte(bookingID),
		Value: eventBytes,
	})
	if err != nil {
		log.Fatalf("[FATAL] Failed to write message to Kafka: %v", err)
	}

	fmt.Printf("🚀 [SUCCESS] Event successfully published!\n")
	fmt.Printf("   Topic: %s\n   Type:  %s\n   ID:    %s\n", topic, eventType, eventID)
}

func getKafkaBrokers() string {
	// 1. Check system environment variable first
	if envBrokers := os.Getenv("KAFKA_BROKERS"); envBrokers != "" {
		return envBrokers
	}

	brokers := "localhost:29091,localhost:29092,localhost:29093"

	// 2. Try to read from .env in the current working directory
	file, err := os.Open(".env")
	if err == nil {
		defer func() { _ = file.Close() }()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "KAFKA_BROKERS=") {
				val := strings.TrimPrefix(line, "KAFKA_BROKERS=")
				val = strings.Trim(val, `"'`)
				brokers = val
				break
			}
		}
	}

	// 3. Verify if the brokers are resolvable. If not (e.g. running on host machine),
	// fallback to localhost mapped ports
	brokerList := strings.Split(brokers, ",")
	if len(brokerList) > 0 {
		firstBroker := brokerList[0]
		// Perform a real TCP dial to ensure the broker is actually reachable on the specified port.
		// On Windows/host machines, local DNS or wildcards might resolve 'kafka-1', but port 9092
		// is only reachable within the Docker network. Testing connection ensures proper fallback.
		conn, err := net.DialTimeout("tcp", firstBroker, 1*time.Second)
		if err != nil {
			fmt.Printf("[INFO] Broker '%s' is not reachable (%v). Falling back to localhost Kafka brokers...\n", firstBroker, err)
			return "localhost:29091,localhost:29092,localhost:29093"
		}
		_ = conn.Close()
	}

	return brokers
}
