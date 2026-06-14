package handler

import (
	"context"
	"errors"
	"strconv"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	bookingv1 "github.com/rent-a-girlfriend/booking-service/gen/proto"
	"github.com/rent-a-girlfriend/booking-service/internal/application/command"
	"github.com/rent-a-girlfriend/booking-service/internal/application/query"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/aggregate"
	domainerr "github.com/rent-a-girlfriend/booking-service/internal/domain/errors"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
	"github.com/rent-a-girlfriend/booking-service/internal/interfaces/grpc/util"
)

// BookingGRPCHandler implements the gRPC BookingServiceServer interface.
type BookingGRPCHandler struct {
	bookingv1.UnimplementedBookingServiceServer
	requestBooking  *command.RequestBookingHandler
	acceptBooking   *command.AcceptBookingHandler
	rejectBooking   *command.RejectBookingHandler
	cancelBooking   *command.CancelBookingHandler
	completeBooking *command.CompleteBookingHandler
	getBooking      *query.GetBookingHandler
	listBookings    *query.ListBookingsHandler
}

// NewBookingGRPCHandler creates a new BookingGRPCHandler instance.
func NewBookingGRPCHandler(
	requestBooking *command.RequestBookingHandler,
	acceptBooking *command.AcceptBookingHandler,
	rejectBooking *command.RejectBookingHandler,
	cancelBooking *command.CancelBookingHandler,
	completeBooking *command.CompleteBookingHandler,
	getBooking *query.GetBookingHandler,
	listBookings *query.ListBookingsHandler,
) *BookingGRPCHandler {
	return &BookingGRPCHandler{
		requestBooking:  requestBooking,
		acceptBooking:   acceptBooking,
		rejectBooking:   rejectBooking,
		cancelBooking:   cancelBooking,
		completeBooking: completeBooking,
		getBooking:      getBooking,
		listBookings:    listBookings,
	}
}

// RequestBooking handles client booking creation request.
func (h *BookingGRPCHandler) RequestBooking(ctx context.Context, req *bookingv1.RequestBookingRequest) (*bookingv1.RequestBookingResponse, error) {
	clientID := util.GetUserID(ctx)
	if clientID == "" {
		return nil, status.Error(codes.Unauthenticated, "client identity is missing")
	}

	startTime := req.StartTime.AsTime()
	booking, err := h.requestBooking.Handle(ctx, command.RequestBookingCmd{
		ClientID:    clientID,
		CompanionID: req.CompanionId,
		ScenarioID:  req.ScenarioId,
		StartTime:   startTime,
	})
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &bookingv1.RequestBookingResponse{
		BookingId: booking.ID().String(),
		Status:    mapBookingStatus(booking.Status()),
		Message:   "Booking request created successfully",
	}, nil
}

// AcceptBooking handles companion booking acceptance request.
func (h *BookingGRPCHandler) AcceptBooking(ctx context.Context, req *bookingv1.AcceptBookingRequest) (*bookingv1.AcceptBookingResponse, error) {
	companionID := util.GetUserID(ctx)
	if companionID == "" {
		return nil, status.Error(codes.Unauthenticated, "companion identity is missing")
	}

	booking, err := h.acceptBooking.Handle(ctx, command.AcceptBookingCmd{
		BookingID:   req.BookingId,
		CompanionID: companionID,
	})
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &bookingv1.AcceptBookingResponse{
		BookingId: booking.ID().String(),
		Status:    mapBookingStatus(booking.Status()),
		Message:   "Booking accepted and escrow holds processed",
	}, nil
}

// RejectBooking handles companion booking rejection request.
func (h *BookingGRPCHandler) RejectBooking(ctx context.Context, req *bookingv1.RejectBookingRequest) (*bookingv1.RejectBookingResponse, error) {
	companionID := util.GetUserID(ctx)
	if companionID == "" {
		return nil, status.Error(codes.Unauthenticated, "companion identity is missing")
	}

	booking, err := h.rejectBooking.Handle(ctx, command.RejectBookingCmd{
		BookingID:   req.BookingId,
		CompanionID: companionID,
		Reason:      req.Reason,
	})
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &bookingv1.RejectBookingResponse{
		BookingId: booking.ID().String(),
		Status:    mapBookingStatus(booking.Status()),
		Message:   "Booking rejected successfully",
	}, nil
}

// CancelBooking handles client/companion booking cancellation request.
func (h *BookingGRPCHandler) CancelBooking(ctx context.Context, req *bookingv1.CancelBookingRequest) (*bookingv1.CancelBookingResponse, error) {
	actorID := util.GetUserID(ctx)
	if actorID == "" {
		return nil, status.Error(codes.Unauthenticated, "actor identity is missing")
	}

	actorRole := util.GetUserRole(ctx)
	if actorRole == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user role header")
	}

	booking, err := h.cancelBooking.Handle(ctx, command.CancelBookingCmd{
		BookingID: req.BookingId,
		ActorID:   actorID,
		ActorRole: actorRole,
	})
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &bookingv1.CancelBookingResponse{
		BookingId: booking.ID().String(),
		Status:    mapBookingStatus(booking.Status()),
		Message:   "Booking cancelled successfully",
	}, nil
}

// CompleteBooking handles client booking completion request.
func (h *BookingGRPCHandler) CompleteBooking(ctx context.Context, req *bookingv1.CompleteBookingRequest) (*bookingv1.CompleteBookingResponse, error) {
	clientID := util.GetUserID(ctx)
	if clientID == "" {
		return nil, status.Error(codes.Unauthenticated, "client identity is missing")
	}

	booking, err := h.completeBooking.Handle(ctx, command.CompleteBookingCmd{
		BookingID: req.BookingId,
		ClientID:  clientID,
	})
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &bookingv1.CompleteBookingResponse{
		BookingId: booking.ID().String(),
		Status:    mapBookingStatus(booking.Status()),
		Message:   "Booking completed successfully",
	}, nil
}

// GetBooking handles fetching detailed information for a single booking.
func (h *BookingGRPCHandler) GetBooking(ctx context.Context, req *bookingv1.GetBookingRequest) (*bookingv1.BookingDetailResponse, error) {
	booking, err := h.getBooking.Handle(ctx, query.GetBookingQuery{
		BookingID: req.BookingId,
	})
	if err != nil {
		return nil, mapDomainError(err)
	}

	// Verify authorization: caller must be client, companion, or admin
	callerID := util.GetUserID(ctx)
	callerRole := util.GetUserRole(ctx)
	if callerRole != "ADMIN" {
		if callerID == "" || (booking.ClientID().String() != callerID && booking.CompanionID().String() != callerID) {
			return nil, status.Error(codes.PermissionDenied, "unauthorized to access this booking")
		}
	}

	return mapToBookingDetailResponse(booking), nil
}

// ListBookings handles listing and filtering bookings.
func (h *BookingGRPCHandler) ListBookings(ctx context.Context, req *bookingv1.ListBookingsRequest) (*bookingv1.ListBookingsResponse, error) {
	// Strictly extract authenticated user ID and role from context (injected by Istio Waypoint)
	authID := util.GetUserID(ctx)
	authRole := util.GetUserRole(ctx)

	// Fallback to request payload ONLY if context metadata is missing (e.g. in local development / unit tests)
	callerID := authID
	if callerID == "" {
		callerID = req.ActorId
	}
	callerRole := authRole
	if callerRole == "" {
		callerRole = req.ActorRole
	}

	page := int64(1)
	if req.PageToken != "" {
		if p, err := strconv.ParseInt(req.PageToken, 10, 64); err == nil {
			page = p
		}
	}

	pageSize := int64(req.PageSize)
	if pageSize <= 0 {
		pageSize = 20
	}

	var clientIDPtr, companionIDPtr *string
	switch callerRole {
	case "CLIENT":
		// Non-admin CLIENT is strictly restricted to their own authenticated ID
		actualID := authID
		if actualID == "" {
			actualID = callerID // fallback for local/unit testing
		}
		clientIDPtr = &actualID
	case "COMPANION":
		// Non-admin COMPANION is strictly restricted to their own authenticated ID
		actualID := authID
		if actualID == "" {
			actualID = callerID // fallback for local/unit testing
		}
		companionIDPtr = &actualID
	case "ADMIN":
		// Admin can filter by any actor using request parameters
		if req.ActorId != "" {
			switch req.ActorRole {
			case "CLIENT":
				clientIDPtr = &req.ActorId
			case "COMPANION":
				companionIDPtr = &req.ActorId
			}
		}
	default:
		// Missing or unrecognized role must be rejected to prevent BOLA data leakage
		return nil, status.Error(codes.PermissionDenied, "unauthorized or missing user role header")
	}

	var statusesFilter []string
	if req.View != "" {
		switch req.View {
		case "pending":
			statusesFilter = []string{"PENDING"}
		case "upcoming":
			statusesFilter = []string{"ACCEPTED"}
		case "history":
			statusesFilter = []string{"COMPLETED", "CANCELLED", "DISPUTED"}
		default:
			statusesFilter = []string{req.View}
		}
	}

	result, err := h.listBookings.Handle(ctx, query.ListBookingsQuery{
		ClientID:    clientIDPtr,
		CompanionID: companionIDPtr,
		Statuses:    statusesFilter,
		Page:        page,
		PageSize:    pageSize,
	})
	if err != nil {
		return nil, mapDomainError(err)
	}

	bookingsProto := make([]*bookingv1.BookingDetailResponse, 0, len(result.Bookings))
	for _, b := range result.Bookings {
		bookingsProto = append(bookingsProto, mapToBookingDetailResponse(b))
	}

	var nextPageToken string
	if page*pageSize < result.Total {
		nextPageToken = strconv.FormatInt(page+1, 10)
	}

	return &bookingv1.ListBookingsResponse{
		Bookings:      bookingsProto,
		NextPageToken: nextPageToken,
	}, nil
}

// --- Helpers and Mappers ---

func mapBookingStatus(status vo.BookingStatus) bookingv1.BookingStatus {
	switch status {
	case vo.StatusPendingReserving:
		return bookingv1.BookingStatus_BOOKING_STATUS_PENDING_RESERVING
	case vo.StatusPending:
		return bookingv1.BookingStatus_BOOKING_STATUS_PENDING
	case vo.StatusAccepted:
		return bookingv1.BookingStatus_BOOKING_STATUS_ACCEPTED
	case vo.StatusCompleted:
		return bookingv1.BookingStatus_BOOKING_STATUS_COMPLETED
	case vo.StatusCancelled:
		return bookingv1.BookingStatus_BOOKING_STATUS_CANCELLED
	case vo.StatusDisputed:
		return bookingv1.BookingStatus_BOOKING_STATUS_DISPUTED
	case vo.StatusResolved:
		return bookingv1.BookingStatus_BOOKING_STATUS_RESOLVED
	default:
		return bookingv1.BookingStatus_BOOKING_STATUS_UNSPECIFIED
	}
}

func mapToBookingDetailResponse(b *aggregate.Booking) *bookingv1.BookingDetailResponse {
	return &bookingv1.BookingDetailResponse{
		BookingId:       b.ID().String(),
		ClientId:        b.ClientID().String(),
		CompanionId:     b.CompanionID().String(),
		Price:           int64(b.Scenario().Price().Amount()),
		DurationMinutes: int64(b.Scenario().DurationMinutes()),
		StartTime:       timestamppb.New(b.TimeRange().StartTime()),
		EndTime:         timestamppb.New(b.TimeRange().EndTime()),
		Status:          mapBookingStatus(b.Status()),
		CreatedAt:       timestamppb.New(b.CreatedAt()),
	}
}

func mapDomainError(err error) error {
	switch {
	case errors.Is(err, domainerr.ErrBookingNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domainerr.ErrInvalidStatus):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domainerr.ErrPendingCapExceeded):
		return status.Error(codes.ResourceExhausted, err.Error())
	case errors.Is(err, domainerr.ErrConcurrencyConflict):
		return status.Error(codes.Aborted, err.Error())
	case errors.Is(err, domainerr.ErrUnauthorized):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, domainerr.ErrInsufficientFunds):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
