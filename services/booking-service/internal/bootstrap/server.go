package bootstrap

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/gorm"

	bookingv1 "github.com/rent-a-girlfriend/booking-service/gen/proto"
	"github.com/rent-a-girlfriend/booking-service/internal/application/command"
	"github.com/rent-a-girlfriend/booking-service/internal/application/query"
	"github.com/rent-a-girlfriend/booking-service/internal/infrastructure/client"
	"github.com/rent-a-girlfriend/booking-service/internal/infrastructure/messaging"
	"github.com/rent-a-girlfriend/booking-service/internal/infrastructure/persistence"
	grpchandler "github.com/rent-a-girlfriend/booking-service/internal/interfaces/grpc/handler"
	grpcinterceptor "github.com/rent-a-girlfriend/booking-service/internal/interfaces/grpc/interceptor"
	router "github.com/rent-a-girlfriend/booking-service/internal/interfaces/http"
	"github.com/rent-a-girlfriend/booking-service/internal/infrastructure/worker"
)

// Server holds all wired dependencies.
type Server struct {
	Router             http.Handler
	GRPCServer         *grpc.Server
	OutboxWorker       *messaging.OutboxWorker
	AutoCompleteWorker *worker.AutoCompleteWorker
	KafkaConsumer      *messaging.KafkaConsumer
	KafkaPublisher     *messaging.KafkaPublisher
	ProfileConn        *grpc.ClientConn
	FinanceConn        *grpc.ClientConn
}

// NewServer wires all dependencies and returns a configured server.
func NewServer(db *gorm.DB, cfg *Config) *Server {
	// --- Infrastructure (Adapters) ---
	bookingRepo := persistence.NewBookingRepository(db)
	sagaRepo := persistence.NewBookingSagaRepo(db)
	outboxPublisher := persistence.NewOutboxPublisher(db)

	// --- Dial gRPC Clients ---
	profileConn, err := grpc.Dial(cfg.Clients.ProfileServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to dial profile service at %s: %v", cfg.Clients.ProfileServiceAddr, err)
	}

	financeConn, err := grpc.Dial(cfg.Clients.FinanceServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to dial finance service at %s: %v", cfg.Clients.FinanceServiceAddr, err)
	}

	// --- Clients (Implements Port Interfaces) ---
	profileService := client.NewProfileClient(profileConn)
	financeService := client.NewFinanceClient(financeConn)

	// --- Messaging ---
	kafkaConnCfg := messaging.KafkaConnConfig{
		Brokers:       cfg.Kafka.Brokers,
		SASLUsername:  cfg.Kafka.SASLUsername,
		SASLPassword:  cfg.Kafka.SASLPassword,
		SASLMechanism: cfg.Kafka.SASLMechanism,
		TLSEnabled:    cfg.Kafka.TLSEnabled,
	}
	kafkaPublisher := messaging.NewKafkaPublisher(kafkaConnCfg, cfg.Kafka.TopicBooking)
	outboxWorker := messaging.NewOutboxWorker(
		db,
		kafkaPublisher,
		cfg.Outbox.PollingInterval,
		cfg.Outbox.BatchSize,
		cfg.Kafka.TopicBooking,
	)

	// --- Application (Command Handlers) ---
	requestBookingHandler := command.NewRequestBookingHandler(bookingRepo, profileService, financeService)
	acceptBookingHandler := command.NewAcceptBookingHandler(bookingRepo, sagaRepo, db, outboxPublisher)
	rejectBookingHandler := command.NewRejectBookingHandler(bookingRepo, financeService)
	cancelBookingHandler := command.NewCancelBookingHandler(bookingRepo)
	completeBookingHandler := command.NewCompleteBookingHandler(bookingRepo)
	systemCompleteBookingHandler := command.NewSystemCompleteBookingHandler(bookingRepo)
	disputeBookingHandler := command.NewDisputeBookingHandler(bookingRepo)

	// --- Auto-Complete Worker ---
	autoCompleteWorker := worker.NewAutoCompleteWorker(
		bookingRepo,
		systemCompleteBookingHandler,
		cfg.Worker.AutoCompleteInterval,
		cfg.Worker.AutoCompleteBuffer,
	)

	// --- SAGA Coordinator ---
	sagaCoordinator := command.NewSagaCoordinator(bookingRepo, sagaRepo, db, outboxPublisher)

	// --- Kafka Consumer ---
	kafkaConsumer := messaging.NewKafkaConsumer(
		kafkaConnCfg,
		"booking-service",
		[]string{cfg.Kafka.TopicFinance, cfg.Kafka.TopicInteraction, cfg.Kafka.TopicDispute},
		sagaCoordinator,
		disputeBookingHandler,
		db,
	)

	// --- Application (Query Handlers) ---
	getBookingHandler := query.NewGetBookingHandler(bookingRepo)
	listBookingsHandler := query.NewListBookingsHandler(bookingRepo)

	// --- Router (gRPC Gateway) ---
	grpcTargetAddr := "localhost:" + cfg.Server.GRPCPort
	r, err := router.NewGateway(context.Background(), grpcTargetAddr)
	if err != nil {
		log.Fatalf("failed to initialize HTTP Gateway: %v", err)
	}

	// --- Interfaces (gRPC Handlers) ---
	grpcHandler := grpchandler.NewBookingGRPCHandler(
		requestBookingHandler,
		acceptBookingHandler,
		rejectBookingHandler,
		cancelBookingHandler,
		completeBookingHandler,
		getBookingHandler,
		listBookingsHandler,
	)

	// --- gRPC Server ---
	gServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpcinterceptor.AuthInterceptor),
	)
	bookingv1.RegisterBookingServiceServer(gServer, grpcHandler)

	return &Server{
		Router:             r,
		GRPCServer:         gServer,
		OutboxWorker:       outboxWorker,
		AutoCompleteWorker: autoCompleteWorker,
		KafkaConsumer:      kafkaConsumer,
		KafkaPublisher:     kafkaPublisher,
		ProfileConn:        profileConn,
		FinanceConn:        financeConn,
	}
}

// Run starts HTTP, gRPC servers, the Outbox Worker, and Kafka Consumer concurrently.
func (s *Server) Run(ctx context.Context, httpAddr, grpcAddr string) error {
	errChan := make(chan error, 2)

	// Start Outbox Worker (background)
	go func() {
		log.Println("[OUTBOX] Starting outbox polling worker...")
		s.OutboxWorker.Start(ctx)
	}()

	// Start Auto-Complete Worker (background)
	go func() {
		log.Println("[AUTO-COMPLETE-WORKER] Starting auto-completion worker...")
		s.AutoCompleteWorker.Start(ctx)
	}()

	// Start Kafka Consumer (background)
	go func() {
		log.Println("[KAFKA-CONSUMER] Starting Kafka consumer...")
		s.KafkaConsumer.Start(ctx)
	}()

	// Start gRPC server
	go func() {
		log.Printf("[GRPC] Booking Service starting on %s", grpcAddr)
		lis, err := net.Listen("tcp", grpcAddr)
		if err != nil {
			errChan <- fmt.Errorf("failed to listen for gRPC: %w", err)
			return
		}
		if err := s.GRPCServer.Serve(lis); err != nil {
			errChan <- fmt.Errorf("failed to serve gRPC: %w", err)
		}
	}()

	// Start HTTP server
	go func() {
		log.Printf("[HTTP] Booking Service starting on %s", httpAddr)
		httpServer := &http.Server{
			Addr:    httpAddr,
			Handler: s.Router,
		}
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("failed to start HTTP server: %w", err)
		}
	}()

	return <-errChan
}
