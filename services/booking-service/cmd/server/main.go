package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/rent-a-girlfriend/booking-service/internal/bootstrap"
)

func main() {
	// 1. Load configuration
	cfg := bootstrap.LoadConfig()

	// 2. Initialize database
	db, err := bootstrap.InitDatabase(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// 3. Wire dependencies and create server
	server := bootstrap.NewServer(db, cfg)

	// 4. Handle OS signals for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 5. Start HTTP, gRPC servers + background workers
	httpAddr := ":" + cfg.Server.Port
	grpcAddr := ":" + cfg.Server.GRPCPort
	log.Printf("[MAIN] Booking Service starting (HTTP: %s, gRPC: %s)", httpAddr, grpcAddr)
	if err := server.Run(ctx, httpAddr, grpcAddr); err != nil {
		log.Fatalf("[MAIN] Server failed: %v", err)
	}
}
