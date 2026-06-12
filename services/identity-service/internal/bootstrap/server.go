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
	"gorm.io/gorm"

	identityv1 "github.com/rent-a-girlfriend/identity-service/gen/proto"
	"github.com/rent-a-girlfriend/identity-service/internal/application/command"
	"github.com/rent-a-girlfriend/identity-service/internal/application/query"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/service"
	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/cache"
	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/client"
	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/crypto"
	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/messaging"
	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/persistence"
	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/store"
	grpchandler "github.com/rent-a-girlfriend/identity-service/internal/interfaces/grpc/handler"
	grpcinterceptor "github.com/rent-a-girlfriend/identity-service/internal/interfaces/grpc/interceptor"
	gateway "github.com/rent-a-girlfriend/identity-service/internal/interfaces/http"
	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/worker"
)

// Server holds all wired dependencies.
type Server struct {
	GRPCServer       *grpc.Server
	outboxWorker     *messaging.OutboxWorker
	dbCleanupWorker  *worker.DbCleanupWorker
	kafkaAdapter     *messaging.KafkaAdapter
	getJWKSHandler   *query.GetJWKSHandler
	mockLoginHandler *command.MockLoginHandler
	db               *gorm.DB
	redisAdapter     *cache.RedisAdapter
}

// NewServer wires all dependencies and returns a configured server.
func NewServer(db *gorm.DB, cfg *Config) *Server {
	// --- Infrastructure: Cache ---
	redisAdapter, err := cache.NewRedisAdapter(cfg.Redis.URL)
	if err != nil {
		log.Fatalf("[CACHE] Failed to initialize Redis: %v", err)
	}

	accountRepo := persistence.NewUserAccountRepoImpl(db, redisAdapter)
	upgradeRepo := persistence.NewUpgradeRequestRepoImpl(db)
	configRepo := persistence.NewSystemConfigRepoImpl(db, redisAdapter)
	pkceStore := store.NewPKCEStoreDB(db)

	keyProvider := crypto.NewRSAKeyProvider(db)
	if err := keyProvider.EnsureSigningKey(); err != nil {
		log.Fatalf("[CRYPTO] Failed to ensure signing key: %v", err)
	}

	tokenService := crypto.NewJWTTokenService(
		db, keyProvider,
		cfg.JWT.AccessTokenTTL,
		cfg.JWT.RefreshTokenTTL,
		cfg.JWT.Issuer,
	)

	googleOAuth := client.NewGoogleOAuthClient(
		cfg.OAuth.GoogleClientID,
		cfg.OAuth.GoogleClientSecret,
		cfg.OAuth.GoogleRedirectURI,
	)

	// --- Messaging & Outbox ---
	kafkaAdapter := messaging.NewKafkaAdapter(cfg.Kafka.Brokers)
	outboxPublisher := persistence.NewOutboxPublisher(db)

	var outboxWorker *messaging.OutboxWorker
	if cfg.Kafka.Brokers != "" && cfg.Kafka.Brokers != "disabled" && cfg.Kafka.Brokers != "none" {
		outboxWorker = messaging.NewOutboxWorker(
			db,
			kafkaAdapter,
			cfg.Outbox.PollingInterval,
			cfg.Outbox.BatchSize,
			cfg.Kafka.TopicIdentity,
		)
	}

	// --- DB Cleanup Worker ---
	dbCleanupWorker := worker.NewDbCleanupWorker(
		db,
		cfg.Worker.CleanupInterval,
		cfg.Worker.CleanupRetentionDays,
	)

	lockPolicy := service.NewAccountLockPolicyService(configRepo)

	initGoogleAuthHandler := command.NewInitGoogleAuthHandler(googleOAuth, pkceStore)
	loginGoogleHandler := command.NewLoginGoogleHandler(googleOAuth, pkceStore, accountRepo, tokenService, outboxPublisher)
	mockLoginHandler := command.NewMockLoginHandler(accountRepo, tokenService, outboxPublisher)
	refreshTokenHandler := command.NewRefreshTokenHandler(tokenService, accountRepo)
	logoutHandler := command.NewLogoutHandler(tokenService)
	requestUpgradeHandler := command.NewRequestCompanionUpgradeHandler(accountRepo, upgradeRepo, outboxPublisher)
	approveUpgradeHandler := command.NewApproveUpgradeHandler(upgradeRepo, accountRepo, outboxPublisher)
	rejectUpgradeHandler := command.NewRejectUpgradeHandler(upgradeRepo, outboxPublisher)
	recordViolationHandler := command.NewRecordViolationHandler(accountRepo, lockPolicy, outboxPublisher)
	lockAccountHandler := command.NewLockAccountHandler(accountRepo, tokenService, outboxPublisher)
	unlockAccountHandler := command.NewUnlockAccountHandler(accountRepo, outboxPublisher)

	_ = recordViolationHandler

	getAccountHandler := query.NewGetAccountHandler(accountRepo)
	getJWKSHandler := query.NewGetJWKSHandler(keyProvider)
	listUpgradeReqsHandler := query.NewListUpgradeRequestsHandler(upgradeRepo)

	// --- Interfaces (gRPC Handlers) ---
	grpcHandler := grpchandler.NewIdentityGRPCHandler(
		getAccountHandler,
		lockAccountHandler,
		unlockAccountHandler,
		approveUpgradeHandler,
		rejectUpgradeHandler,
		requestUpgradeHandler,
		listUpgradeReqsHandler,
		initGoogleAuthHandler,
		loginGoogleHandler,
		refreshTokenHandler,
		logoutHandler,
	)

	// --- gRPC Server ---
	gServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcinterceptor.AuthInterceptor,
			grpcinterceptor.AdminInterceptor,
		),
	)
	identityv1.RegisterIdentityServiceServer(gServer, grpcHandler)

	return &Server{
		GRPCServer:       gServer,
		outboxWorker:     outboxWorker,
		dbCleanupWorker:  dbCleanupWorker,
		kafkaAdapter:     kafkaAdapter,
		getJWKSHandler:   getJWKSHandler,
		mockLoginHandler: mockLoginHandler,
		db:               db,
		redisAdapter:     redisAdapter,
	}
}

// Run starts both HTTP and gRPC servers, and the background workers.
func (s *Server) Run(ctx context.Context, httpAddr, grpcAddr string) error {
	errChan := make(chan error, 3)

	// Start Outbox Worker
	if s.outboxWorker != nil {
		go func() {
			s.outboxWorker.Start(ctx)
		}()
	} else {
		log.Println("[BOOTSTRAP] Outbox Worker is disabled")
	}

	// Start DB Cleanup Worker
	if s.dbCleanupWorker != nil {
		go func() {
			s.dbCleanupWorker.Start(ctx)
		}()
	}

	// Start gRPC server
	go func() {
		log.Printf("[GRPC] Identity Service starting on %s", grpcAddr)
		lis, err := net.Listen("tcp", grpcAddr)
		if err != nil {
			errChan <- fmt.Errorf("failed to listen for gRPC: %w", err)
			return
		}
		if err := s.GRPCServer.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			errChan <- fmt.Errorf("failed to serve gRPC: %w", err)
		}
	}()

	// Initialize HTTP Gateway
	gatewayOpts := s.getTestGatewayOptions()

	// Register health endpoints (/health/live and /health/ready) via GatewayOptions
	gatewayOpts = append(gatewayOpts, gateway.WithAdditionalHandler("/health/live", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","service":"identity-service"}`))
	})))

	gatewayOpts = append(gatewayOpts, gateway.WithAdditionalHandler("/health/ready", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// 1. Ping DB
		sqlDB, err := s.db.DB()
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

		// 2. Ping Redis
		if err := s.redisAdapter.Ping(pingCtx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status": "error",
				"reason": fmt.Sprintf("redis connection error: %v", err),
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","service":"identity-service"}`))
	})))

	gwHandler, err := gateway.NewGateway(ctx, grpcAddr, s.getJWKSHandler, gatewayOpts...)
	if err != nil {
		return fmt.Errorf("failed to initialize HTTP Gateway: %w", err)
	}

	httpServer := &http.Server{
		Addr:    httpAddr,
		Handler: gwHandler,
	}

	// Start HTTP server (gRPC Gateway)
	go func() {
		log.Printf("[HTTP] Identity Service starting on %s (grpc-gateway)", httpAddr)
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

		// Close Kafka connection if exists
		if s.kafkaAdapter != nil {
			_ = s.kafkaAdapter.Close()
		}

		errChan <- nil // unblock Run
	}()

	return <-errChan
}
