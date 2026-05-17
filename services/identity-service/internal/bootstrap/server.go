package bootstrap

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"gorm.io/gorm"

	"github.com/rent-a-girlfriend/identity-service/internal/application/command"
	"github.com/rent-a-girlfriend/identity-service/internal/application/query"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/service"
	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/client"
	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/cache"
	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/crypto"
	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/persistence"
	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/store"
	grpchandler "github.com/rent-a-girlfriend/identity-service/internal/interfaces/grpc/handler"
	grpcinterceptor "github.com/rent-a-girlfriend/identity-service/internal/interfaces/grpc/interceptor"
	httphandler "github.com/rent-a-girlfriend/identity-service/internal/interfaces/http/handler"
	router "github.com/rent-a-girlfriend/identity-service/internal/interfaces/http"
	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/messaging"
	identityv1 "github.com/rent-a-girlfriend/identity-service/api/proto"
)

// Server holds all wired dependencies.
type Server struct {
	Router       *gin.Engine
	GRPCServer   *grpc.Server
	outboxWorker *messaging.OutboxWorker
	kafkaAdapter *messaging.KafkaAdapter
}

// NewServer wires all dependencies and returns a configured server.
func NewServer(db *gorm.DB, cfg *Config) *Server {
	gin.SetMode(cfg.Server.Mode)

	// --- Infrastructure: Cache ---
	redisAdapter, err := cache.NewRedisAdapter(cfg.Redis.URL)
	if err != nil {
		log.Fatalf("[CACHE] Failed to initialize Redis: %v", err)
	}

	// ... (Infrastructure and Domain logic remains same)
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

	// --- Interfaces (HTTP Handlers) ---
	authHandler := httphandler.NewAuthHandler(
		initGoogleAuthHandler,
		loginGoogleHandler,
		refreshTokenHandler,
		logoutHandler,
		getJWKSHandler,
		requestUpgradeHandler,
		mockLoginHandler,
	)

	adminHandler := httphandler.NewAdminHandler(
		getAccountHandler,
		lockAccountHandler,
		unlockAccountHandler,
		approveUpgradeHandler,
		rejectUpgradeHandler,
		listUpgradeReqsHandler,
	)

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

	// --- Router ---
	r := router.NewRouter(authHandler, adminHandler)

	return &Server{
		Router:       r,
		GRPCServer:   gServer,
		outboxWorker: outboxWorker,
		kafkaAdapter: kafkaAdapter,
	}
}

// Run starts both HTTP and gRPC servers, and the background workers.
func (s *Server) Run(httpAddr, grpcAddr string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errChan := make(chan error, 3)

	// Start Outbox Worker
	if s.outboxWorker != nil {
		go func() {
			s.outboxWorker.Start(ctx)
		}()
	} else {
		log.Println("[BOOTSTRAP] Outbox Worker is disabled")
	}

	// Start gRPC server
	go func() {
		log.Printf("[GRPC] Identity Service starting on %s", grpcAddr)
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
		log.Printf("[HTTP] Identity Service starting on %s", httpAddr)
		if err := s.Router.Run(httpAddr); err != nil {
			errChan <- fmt.Errorf("failed to start HTTP server: %w", err)
		}
	}()

	return <-errChan
}

