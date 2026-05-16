package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rent-a-girlfriend/booking-service/internal/application/command"
	"github.com/rent-a-girlfriend/booking-service/internal/application/query"
	domainerr "github.com/rent-a-girlfriend/booking-service/internal/domain/errors"
	"github.com/rent-a-girlfriend/booking-service/internal/interfaces/http/dto"
)

// BookingHandler handles HTTP requests for booking operations.
type BookingHandler struct {
	requestBooking *command.RequestBookingHandler
	acceptBooking  *command.AcceptBookingHandler
	rejectBooking  *command.RejectBookingHandler
	cancelBooking  *command.CancelBookingHandler
	getBooking     *query.GetBookingHandler
	listBookings   *query.ListBookingsHandler
}

func NewBookingHandler(
	requestBooking *command.RequestBookingHandler,
	acceptBooking *command.AcceptBookingHandler,
	rejectBooking *command.RejectBookingHandler,
	cancelBooking *command.CancelBookingHandler,
	getBooking *query.GetBookingHandler,
	listBookings *query.ListBookingsHandler,
) *BookingHandler {
	return &BookingHandler{
		requestBooking: requestBooking,
		acceptBooking:  acceptBooking,
		rejectBooking:  rejectBooking,
		cancelBooking:  cancelBooking,
		getBooking:     getBooking,
		listBookings:   listBookings,
	}
}

// RequestBooking handles POST /api/v1/bookings
func (h *BookingHandler) RequestBooking(c *gin.Context) {
	var req dto.RequestBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "VALIDATION_ERROR", Message: err.Error()})
		return
	}

	booking, err := h.requestBooking.Handle(c.Request.Context(), command.RequestBookingCmd{
		ClientID:    req.ClientID,
		CompanionID: req.CompanionID,
		ScenarioID:  req.ScenarioID,
		StartTime:   req.StartTime,
	})
	if err != nil {
		mapDomainError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.ToBookingResponse(booking))
}

// AcceptBooking handles PUT /api/v1/bookings/:id/accept
func (h *BookingHandler) AcceptBooking(c *gin.Context) {
	var req dto.AcceptBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "VALIDATION_ERROR", Message: err.Error()})
		return
	}

	booking, err := h.acceptBooking.Handle(c.Request.Context(), command.AcceptBookingCmd{
		BookingID:   c.Param("id"),
		CompanionID: req.CompanionID,
	})
	if err != nil {
		mapDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ToBookingResponse(booking))
}

// RejectBooking handles PUT /api/v1/bookings/:id/reject
func (h *BookingHandler) RejectBooking(c *gin.Context) {
	var req dto.RejectBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "VALIDATION_ERROR", Message: err.Error()})
		return
	}

	booking, err := h.rejectBooking.Handle(c.Request.Context(), command.RejectBookingCmd{
		BookingID:   c.Param("id"),
		CompanionID: req.CompanionID,
	})
	if err != nil {
		mapDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ToBookingResponse(booking))
}

// CancelBooking handles PUT /api/v1/bookings/:id/cancel
func (h *BookingHandler) CancelBooking(c *gin.Context) {
	var req dto.CancelBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "VALIDATION_ERROR", Message: err.Error()})
		return
	}

	booking, err := h.cancelBooking.Handle(c.Request.Context(), command.CancelBookingCmd{
		BookingID: c.Param("id"),
		ActorID:   req.ActorID,
		ActorRole: req.ActorRole,
	})
	if err != nil {
		mapDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ToBookingResponse(booking))
}

// GetBooking handles GET /api/v1/bookings/:id
func (h *BookingHandler) GetBooking(c *gin.Context) {
	booking, err := h.getBooking.Handle(c.Request.Context(), query.GetBookingQuery{
		BookingID: c.Param("id"),
	})
	if err != nil {
		mapDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ToBookingResponse(booking))
}

// ListBookings handles GET /api/v1/bookings
func (h *BookingHandler) ListBookings(c *gin.Context) {
	var req dto.ListBookingsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "VALIDATION_ERROR", Message: err.Error()})
		return
	}

	result, err := h.listBookings.Handle(c.Request.Context(), query.ListBookingsQuery{
		ClientID:    req.ClientID,
		CompanionID: req.CompanionID,
		Status:      req.Status,
		Page:        req.Page,
		PageSize:    req.PageSize,
	})
	if err != nil {
		mapDomainError(c, err)
		return
	}

	bookings := make([]dto.BookingResponse, 0, len(result.Bookings))
	for _, b := range result.Bookings {
		bookings = append(bookings, dto.ToBookingResponse(b))
	}

	c.JSON(http.StatusOK, dto.ListBookingsResponse{
		Data:     bookings,
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	})
}

// mapDomainError maps domain errors to appropriate HTTP status codes.
func mapDomainError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domainerr.ErrBookingNotFound):
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "NOT_FOUND", Message: err.Error()})
	case errors.Is(err, domainerr.ErrInvalidStatus):
		c.JSON(http.StatusConflict, dto.ErrorResponse{Error: "INVALID_STATUS", Message: err.Error()})
	case errors.Is(err, domainerr.ErrPendingCapExceeded):
		c.JSON(http.StatusTooManyRequests, dto.ErrorResponse{Error: "PENDING_CAP_EXCEEDED", Message: err.Error()})
	case errors.Is(err, domainerr.ErrConcurrencyConflict):
		c.JSON(http.StatusConflict, dto.ErrorResponse{Error: "CONCURRENCY_CONFLICT", Message: err.Error()})
	case errors.Is(err, domainerr.ErrUnauthorized):
		c.JSON(http.StatusForbidden, dto.ErrorResponse{Error: "UNAUTHORIZED", Message: err.Error()})
	case errors.Is(err, domainerr.ErrInsufficientFunds):
		c.JSON(http.StatusPaymentRequired, dto.ErrorResponse{Error: "INSUFFICIENT_FUNDS", Message: err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "INTERNAL_ERROR", Message: err.Error()})
	}
}
