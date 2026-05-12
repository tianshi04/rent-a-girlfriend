package main

import (
	"log"

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

	// Start servers
	httpAddr := ":" + cfg.Server.Port
	grpcAddr := ":" + cfg.Server.GRPCPort
	log.Printf("[MAIN] Identity Service starting (HTTP: %s, gRPC: %s)", httpAddr, grpcAddr)

	if err := server.Run(httpAddr, grpcAddr); err != nil {
		log.Fatalf("[MAIN] Server failed: %v", err)
	}
}
