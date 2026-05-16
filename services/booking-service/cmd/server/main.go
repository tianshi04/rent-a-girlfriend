package main

import (
	"log"

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

	// 4. Start HTTP server
	addr := ":" + cfg.Server.Port
	log.Printf("[SERVER] Booking Service starting on %s", addr)
	if err := server.Router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
