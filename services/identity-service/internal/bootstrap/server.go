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
	"github.com/rent-a-girlfriend/identity-service/internal/domain/event"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/service"
	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/client"
	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/crypto"
	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/persistence"
	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/store"
	grpchandler "github.com/rent-a-girlfriend/identity-service/internal/interfaces/grpc/handler"
	grpcinterceptor "github.com/rent-a-girlfriend/identity-service/internal/interfaces/grpc/interceptor"
	httphandler "github.com/rent-a-girlfriend/identity-service/internal/interfaces/http/handler"
	router "github.com/rent-a-girlfriend/identity-service/internal/interfaces/http"
	identityv1 "github.com/rent-a-girlfriend/identity-service/api/proto"
)

// Server holds all wired dependencies.
type Server struct {
	Router     *gin.Engine
	GRPCServer *grpc.Server
}

// NewServer wires all dependencies and returns a configured server.
func NewServer(db *gorm.DB, cfg *Config) *Server {
	gin.SetMode(cfg.Server.Mode)

	// ... (Infrastructure and Domain logic remains same)
	accountRepo := persistence.NewUserAccountRepoImpl(db)
	upgradeRepo := persistence.NewUpgradeRequestRepoImpl(db)
	configRepo := persistence.NewSystemConfigRepoImpl(db)
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

	var publisher noopPublisher

	lockPolicy := service.NewAccountLockPolicyService(configRepo)

	initGoogleAuthHandler := command.NewInitGoogleAuthHandler(googleOAuth, pkceStore)
	loginGoogleHandler := command.NewLoginGoogleHandler(googleOAuth, pkceStore, accountRepo, tokenService, &publisher)
	refreshTokenHandler := command.NewRefreshTokenHandler(tokenService, accountRepo)
	logoutHandler := command.NewLogoutHandler(tokenService)
	requestUpgradeHandler := command.NewRequestCompanionUpgradeHandler(accountRepo, upgradeRepo, &publisher)
	approveUpgradeHandler := command.NewApproveUpgradeHandler(upgradeRepo, accountRepo, &publisher)
	rejectUpgradeHandler := command.NewRejectUpgradeHandler(upgradeRepo, &publisher)
	recordViolationHandler := command.NewRecordViolationHandler(accountRepo, lockPolicy, &publisher)
	lockAccountHandler := command.NewLockAccountHandler(accountRepo, &publisher)
	unlockAccountHandler := command.NewUnlockAccountHandler(accountRepo, &publisher)

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
		Router:     r,
		GRPCServer: gServer,
	}
}

// Run starts both HTTP and gRPC servers.
func (s *Server) Run(httpAddr, grpcAddr string) error {
	errChan := make(chan error, 2)

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

// noopPublisher is a no-op event publisher stub for Phase 1.
// Will be replaced with Kafka publisher in integration phase.
type noopPublisher struct{}

func (p *noopPublisher) Publish(ctx context.Context, evt event.DomainEvent) error {
	return nil
}
