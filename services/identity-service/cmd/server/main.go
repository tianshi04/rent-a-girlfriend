package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/rent-a-girlfriend/identity-service/internal/bootstrap"
)

func main() {
	// Load configuration
	cfg := bootstrap.LoadConfig()

	// Initialize database
	db, err := bootstrap.InitDatabase(cfg.Database)
	if err != nil {
		log.Fatalf("[MAIN] Failed to initialize database: %v", err)
	}

	// Wire dependencies and create server
	server := bootstrap.NewServer(db, cfg)

	// Handle OS signals for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start servers
	httpAddr := ":" + cfg.Server.Port
	grpcAddr := ":" + cfg.Server.GRPCPort
	log.Printf("[MAIN] Identity Service starting (HTTP: %s, gRPC: %s)", httpAddr, grpcAddr)

	if err := server.Run(ctx, httpAddr, grpcAddr); err != nil {
		log.Fatalf("[MAIN] Server failed: %v", err)
	}
}
