package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rent-a-girlfriend/booking-service/internal/interfaces/http/handler"
)

// NewRouter creates and configures the Gin router with all routes.
func NewRouter(bookingHandler *handler.BookingHandler) *gin.Engine {
	r := gin.Default()

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "booking-service"})
	})

	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		bookings := v1.Group("/bookings")
		{
			bookings.POST("", bookingHandler.RequestBooking)
			bookings.GET("", bookingHandler.ListBookings)
			bookings.GET("/:id", bookingHandler.GetBooking)
			bookings.PUT("/:id/accept", bookingHandler.AcceptBooking)
			bookings.PUT("/:id/reject", bookingHandler.RejectBooking)
			bookings.PUT("/:id/cancel", bookingHandler.CancelBooking)
		}
	}

	return r
}
