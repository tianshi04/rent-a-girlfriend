package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/gorm"

	bookingv1 "github.com/rent-a-girlfriend/booking-service/gen/proto"
	"github.com/rent-a-girlfriend/booking-service/internal/application/command"
	"github.com/rent-a-girlfriend/booking-service/internal/application/query"
	"github.com/rent-a-girlfriend/booking-service/internal/infrastructure/client"
	"github.com/rent-a-girlfriend/booking-service/internal/infrastructure/messaging"
	"github.com/rent-a-girlfriend/booking-service/internal/infrastructure/persistence"
	"github.com/rent-a-girlfriend/booking-service/internal/infrastructure/worker"
	grpchandler "github.com/rent-a-girlfriend/booking-service/internal/interfaces/grpc/handler"
	grpcinterceptor "github.com/rent-a-girlfriend/booking-service/internal/interfaces/grpc/interceptor"
	router "github.com/rent-a-girlfriend/booking-service/internal/interfaces/http"
)

// Server holds all wired dependencies.
type Server struct {
	Router               http.Handler
	GRPCServer           *grpc.Server
	OutboxWorker         *messaging.OutboxWorker
	AutoCompleteWorker   *worker.AutoCompleteWorker
	PendingTimeoutWorker *worker.PendingTimeoutWorker
	KafkaConsumer        *messaging.KafkaConsumer
	KafkaPublisher       *messaging.KafkaPublisher
	ProfileConn          *grpc.ClientConn
	FinanceConn          *grpc.ClientConn
}

// NewServer wires all dependencies and returns a configured server.
func NewServer(db *gorm.DB, cfg *Config) *Server {
	// --- Infrastructure (Adapters) ---
	bookingRepo := persistence.NewBookingRepository(db)
	sagaRepo := persistence.NewBookingSagaRepo(db)
	outboxPublisher := persistence.NewOutboxPublisher(db)

	// --- Dial gRPC Clients ---
	profileConn, err := grpc.NewClient(cfg.Clients.ProfileServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to dial profile service at %s: %v", cfg.Clients.ProfileServiceAddr, err)
	}

	financeConn, err := grpc.NewClient(cfg.Clients.FinanceServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
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
	resolveBookingHandler := command.NewResolveBookingHandler(bookingRepo)

	// --- Auto-Complete Worker ---
	autoCompleteWorker := worker.NewAutoCompleteWorker(
		bookingRepo,
		systemCompleteBookingHandler,
		cfg.Worker.AutoCompleteInterval,
		cfg.Worker.AutoCompleteBuffer,
	)

	// --- Pending Timeout Worker ---
	systemRejectBookingHandler := command.NewSystemRejectBookingHandler(bookingRepo, financeService)
	pendingTimeoutWorker := worker.NewPendingTimeoutWorker(
		bookingRepo,
		systemRejectBookingHandler,
		cfg.Worker.AutoCompleteInterval,
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
		resolveBookingHandler,
		db,
	)

	// --- Application (Query Handlers) ---
	getBookingHandler := query.NewGetBookingHandler(bookingRepo)
	listBookingsHandler := query.NewListBookingsHandler(bookingRepo)

	// --- Health check endpoints options ---
	var gatewayOpts []router.GatewayOption

	// 1. /health/live
	gatewayOpts = append(gatewayOpts, router.WithAdditionalHandler("/health/live", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","service":"booking-service"}`))
	})))

	// 2. /health/ready
	gatewayOpts = append(gatewayOpts, router.WithAdditionalHandler("/health/ready", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// 1. Ping DB Postgres
		sqlDB, err := db.DB()
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status": "error",
				"reason": fmt.Sprintf("database client error: %v", err),
			})
			return
		}

		pingCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := sqlDB.PingContext(pingCtx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status": "error",
				"reason": fmt.Sprintf("database connection error: %v", err),
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","service":"booking-service"}`))
	})))

	// --- Router (gRPC Gateway) ---
	grpcTargetAddr := "localhost:" + cfg.Server.GRPCPort
	r, err := router.NewGateway(context.Background(), grpcTargetAddr, gatewayOpts...)
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
		Router:               r,
		GRPCServer:           gServer,
		OutboxWorker:         outboxWorker,
		AutoCompleteWorker:   autoCompleteWorker,
		PendingTimeoutWorker: pendingTimeoutWorker,
		KafkaConsumer:        kafkaConsumer,
		KafkaPublisher:       kafkaPublisher,
		ProfileConn:          profileConn,
		FinanceConn:          financeConn,
	}
}

// Run starts HTTP, gRPC servers, the Outbox Worker, and Kafka Consumer concurrently.
func (s *Server) Run(ctx context.Context, httpAddr, grpcAddr string) error {
	errChan := make(chan error, 3)

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

	// Start Pending-Timeout Worker (background)
	go func() {
		log.Println("[PENDING-TIMEOUT-WORKER] Starting pending timeout worker...")
		s.PendingTimeoutWorker.Start(ctx)
	}()

	// Start gRPC server
	go func() {
		log.Printf("[GRPC] Booking Service starting on %s", grpcAddr)
		lis, err := net.Listen("tcp", grpcAddr)
		if err != nil {
			errChan <- fmt.Errorf("failed to listen for gRPC: %w", err)
			return
		}
		if err := s.GRPCServer.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			errChan <- fmt.Errorf("failed to serve gRPC: %w", err)
		}
	}()

	// Start HTTP server
	httpServer := &http.Server{
		Addr:    httpAddr,
		Handler: s.Router,
	}
	go func() {
		log.Printf("[HTTP] Booking Service starting on %s", httpAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("failed to start HTTP server: %w", err)
		}
	}()

	// Graceful shutdown listener
	go func() {
		<-ctx.Done()
		log.Println("[SERVER] Shutting down servers gracefully...")

		// Stop HTTP server
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)

		// Stop gRPC server
		s.GRPCServer.GracefulStop()

		// Close client connections
		_ = s.ProfileConn.Close()
		_ = s.FinanceConn.Close()

		errChan <- nil // unblock Run
	}()

	return <-errChan
}
