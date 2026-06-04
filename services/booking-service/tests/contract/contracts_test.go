package contract

import (
	"encoding/json"
	"testing"
	"time"

	bookingv1 "github.com/rent-a-girlfriend/booking-service/gen/proto"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestProtobufSerialization_Contracts(t *testing.T) {
	parsedTime, err := time.Parse(time.RFC3339, "2026-05-22T10:00:00Z")
	if err != nil {
		panic(err)
	}
	// 1. Verify RequestBookingRequest serialization
	req := &bookingv1.RequestBookingRequest{
		CompanionId: "companion-123",
		ScenarioId:  "scenario-123",
		StartTime:   timestamppb.New(parsedTime),
	}

	// Protobuf Binary Marshal
	binData, err := proto.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal RequestBookingRequest to protobuf binary: %v", err)
	}

	// Protobuf Binary Unmarshal
	var unmarshaledReq bookingv1.RequestBookingRequest
	err = proto.Unmarshal(binData, &unmarshaledReq)
	if err != nil {
		t.Fatalf("failed to unmarshal RequestBookingRequest from protobuf binary: %v", err)
	}

	if unmarshaledReq.CompanionId != req.CompanionId || unmarshaledReq.ScenarioId != req.ScenarioId {
		t.Errorf("mismatched unmarshaled fields: got %+v, expected %+v", unmarshaledReq, req)
	}

	// JSON Marshal Compatibility
	jsonData, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal RequestBookingRequest to JSON: %v", err)
	}

	var jsonReq bookingv1.RequestBookingRequest
	err = json.Unmarshal(jsonData, &jsonReq)
	if err != nil {
		t.Fatalf("failed to unmarshal RequestBookingRequest from JSON: %v", err)
	}

	if jsonReq.CompanionId != req.CompanionId {
		t.Errorf("mismatched JSON deserialization fields: got %s, expected %s", jsonReq.CompanionId, req.CompanionId)
	}
}

func TestProtobufEnumMappping_Contracts(t *testing.T) {
	// Verify enum definitions are compatible
	pendingVal := bookingv1.BookingStatus_BOOKING_STATUS_PENDING
	acceptedVal := bookingv1.BookingStatus_BOOKING_STATUS_ACCEPTED

	if int64(pendingVal) != 1 {
		t.Errorf("expected pending enum value to be 1, got %d", pendingVal)
	}

	if int64(acceptedVal) != 2 {
		t.Errorf("expected accepted enum value to be 2, got %d", acceptedVal)
	}
}
