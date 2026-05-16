package bootstrap

import (
	"github.com/gin-gonic/gin"

	"github.com/rent-a-girlfriend/booking-service/internal/application/command"
	"github.com/rent-a-girlfriend/booking-service/internal/application/query"
	"github.com/rent-a-girlfriend/booking-service/internal/infrastructure/client"
	"github.com/rent-a-girlfriend/booking-service/internal/infrastructure/persistence"
	httphandler "github.com/rent-a-girlfriend/booking-service/internal/interfaces/http/handler"
	router "github.com/rent-a-girlfriend/booking-service/internal/interfaces/http"

	"gorm.io/gorm"
)

// Server holds all wired dependencies.
type Server struct {
	Router *gin.Engine
}

// NewServer wires all dependencies and returns a configured server.
func NewServer(db *gorm.DB, cfg *Config) *Server {
	gin.SetMode(cfg.Server.Mode)

	// --- Infrastructure (Adapters) ---
	bookingRepo := persistence.NewBookingRepository(db)
	profileService := client.NewProfileServiceStub()
	financeService := client.NewFinanceServiceStub()

	// --- Application (Command Handlers) ---
	requestBookingHandler := command.NewRequestBookingHandler(bookingRepo, profileService, financeService)
	acceptBookingHandler := command.NewAcceptBookingHandler(bookingRepo)
	rejectBookingHandler := command.NewRejectBookingHandler(bookingRepo, financeService)
	cancelBookingHandler := command.NewCancelBookingHandler(bookingRepo)

	// --- Application (Query Handlers) ---
	getBookingHandler := query.NewGetBookingHandler(bookingRepo)
	listBookingsHandler := query.NewListBookingsHandler(bookingRepo)

	// --- Interfaces (HTTP Handlers) ---
	bookingHandler := httphandler.NewBookingHandler(
		requestBookingHandler,
		acceptBookingHandler,
		rejectBookingHandler,
		cancelBookingHandler,
		getBookingHandler,
		listBookingsHandler,
	)

	// --- Router ---
	r := router.NewRouter(bookingHandler)

	return &Server{Router: r}
}
