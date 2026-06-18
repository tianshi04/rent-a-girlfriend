package contract

import (
	"encoding/json"
	"testing"
	"time"

	identityv1 "github.com/rent-a-girlfriend/identity-service/gen/proto"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// --- Serialization Tests ---

func TestProtobufSerialization_TokenResponse(t *testing.T) {
	// 1. Xác minh serialization của TokenResponse
	resp := &identityv1.TokenResponse{
		AccessToken:  "access-token-abc123",
		RefreshToken: "refresh-token-xyz456",
		ExpiresIn:    3600,
	}

	// Protobuf Binary Marshal
	binData, err := proto.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal TokenResponse to protobuf binary: %v", err)
	}

	// Protobuf Binary Unmarshal
	var unmarshaledResp identityv1.TokenResponse
	err = proto.Unmarshal(binData, &unmarshaledResp)
	if err != nil {
		t.Fatalf("failed to unmarshal TokenResponse from protobuf binary: %v", err)
	}

	if unmarshaledResp.AccessToken != resp.AccessToken ||
		unmarshaledResp.RefreshToken != resp.RefreshToken ||
		unmarshaledResp.ExpiresIn != resp.ExpiresIn {
		t.Errorf("mismatched unmarshaled fields: got %+v, expected %+v", &unmarshaledResp, resp)
	}

	// JSON Marshal Compatibility
	jsonData, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal TokenResponse to JSON: %v", err)
	}

	var jsonResp identityv1.TokenResponse
	err = json.Unmarshal(jsonData, &jsonResp)
	if err != nil {
		t.Fatalf("failed to unmarshal TokenResponse from JSON: %v", err)
	}

	if jsonResp.AccessToken != resp.AccessToken {
		t.Errorf("mismatched JSON deserialization: got %s, expected %s", jsonResp.AccessToken, resp.AccessToken)
	}
}

func TestProtobufSerialization_RefreshTokenRequest(t *testing.T) {
	// 2. Xác minh serialization của RefreshTokenRequest
	req := &identityv1.RefreshTokenRequest{
		RefreshToken: "rt-test-token-abc",
	}

	binData, err := proto.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal RefreshTokenRequest to protobuf binary: %v", err)
	}

	var unmarshaledReq identityv1.RefreshTokenRequest
	err = proto.Unmarshal(binData, &unmarshaledReq)
	if err != nil {
		t.Fatalf("failed to unmarshal RefreshTokenRequest from protobuf binary: %v", err)
	}

	if unmarshaledReq.RefreshToken != req.RefreshToken {
		t.Errorf("mismatched RefreshToken: got %s, expected %s", unmarshaledReq.RefreshToken, req.RefreshToken)
	}
}

func TestProtobufSerialization_GetAccountRequest(t *testing.T) {
	// 3. Xác minh serialization của GetAccountRequest
	req := &identityv1.GetAccountRequest{
		Id: "550e8400-e29b-41d4-a716-446655440000",
	}

	binData, err := proto.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal GetAccountRequest to protobuf binary: %v", err)
	}

	var unmarshaledReq identityv1.GetAccountRequest
	err = proto.Unmarshal(binData, &unmarshaledReq)
	if err != nil {
		t.Fatalf("failed to unmarshal GetAccountRequest from protobuf binary: %v", err)
	}

	if unmarshaledReq.Id != req.Id {
		t.Errorf("mismatched Id: got %s, expected %s", unmarshaledReq.Id, req.Id)
	}
}

func TestProtobufSerialization_AccountResponse(t *testing.T) {
	// 4. Xác minh serialization của AccountResponse (composite message với enum + timestamp)
	parsedTime, err := time.Parse(time.RFC3339, "2026-05-10T11:00:00Z")
	if err != nil {
		t.Fatalf("failed to parse time: %v", err)
	}

	resp := &identityv1.AccountResponse{
		Id:             "550e8400-e29b-41d4-a716-446655440000",
		Email:          "test@example.com",
		Role:           identityv1.AccountRole_ACCOUNT_ROLE_CLIENT,
		Status:         identityv1.AccountStatus_ACCOUNT_STATUS_ACTIVE,
		ViolationCount: 0,
		CreatedAt:      timestamppb.New(parsedTime),
	}

	binData, err := proto.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal AccountResponse to protobuf binary: %v", err)
	}

	var unmarshaledResp identityv1.AccountResponse
	err = proto.Unmarshal(binData, &unmarshaledResp)
	if err != nil {
		t.Fatalf("failed to unmarshal AccountResponse from protobuf binary: %v", err)
	}

	if unmarshaledResp.Id != resp.Id ||
		unmarshaledResp.Email != resp.Email ||
		unmarshaledResp.Role != resp.Role ||
		unmarshaledResp.Status != resp.Status {
		t.Errorf("mismatched unmarshaled fields: got %+v, expected %+v", &unmarshaledResp, resp)
	}
}

// --- Enum Mapping Tests ---

func TestProtobufEnumMapping_AccountRole(t *testing.T) {
	// Xác minh numeric value của AccountRole không thay đổi giữa các version
	if int32(identityv1.AccountRole_ACCOUNT_ROLE_UNSPECIFIED) != 0 {
		t.Errorf("expected ACCOUNT_ROLE_UNSPECIFIED = 0, got %d", identityv1.AccountRole_ACCOUNT_ROLE_UNSPECIFIED)
	}
	if int32(identityv1.AccountRole_ACCOUNT_ROLE_CLIENT) != 1 {
		t.Errorf("expected ACCOUNT_ROLE_CLIENT = 1, got %d", identityv1.AccountRole_ACCOUNT_ROLE_CLIENT)
	}
	if int32(identityv1.AccountRole_ACCOUNT_ROLE_COMPANION) != 2 {
		t.Errorf("expected ACCOUNT_ROLE_COMPANION = 2, got %d", identityv1.AccountRole_ACCOUNT_ROLE_COMPANION)
	}
	if int32(identityv1.AccountRole_ACCOUNT_ROLE_ADMIN) != 3 {
		t.Errorf("expected ACCOUNT_ROLE_ADMIN = 3, got %d", identityv1.AccountRole_ACCOUNT_ROLE_ADMIN)
	}
}

func TestProtobufEnumMapping_AccountStatus(t *testing.T) {
	// Xác minh numeric value của AccountStatus không thay đổi giữa các version
	if int32(identityv1.AccountStatus_ACCOUNT_STATUS_UNSPECIFIED) != 0 {
		t.Errorf("expected ACCOUNT_STATUS_UNSPECIFIED = 0, got %d", identityv1.AccountStatus_ACCOUNT_STATUS_UNSPECIFIED)
	}
	if int32(identityv1.AccountStatus_ACCOUNT_STATUS_ACTIVE) != 1 {
		t.Errorf("expected ACCOUNT_STATUS_ACTIVE = 1, got %d", identityv1.AccountStatus_ACCOUNT_STATUS_ACTIVE)
	}
	if int32(identityv1.AccountStatus_ACCOUNT_STATUS_LOCKED) != 2 {
		t.Errorf("expected ACCOUNT_STATUS_LOCKED = 2, got %d", identityv1.AccountStatus_ACCOUNT_STATUS_LOCKED)
	}
}

func TestProtobufEnumMapping_UpgradeStatus(t *testing.T) {
	// Xác minh numeric value của UpgradeStatus không thay đổi giữa các version
	if int32(identityv1.UpgradeStatus_UPGRADE_STATUS_UNSPECIFIED) != 0 {
		t.Errorf("expected UPGRADE_STATUS_UNSPECIFIED = 0, got %d", identityv1.UpgradeStatus_UPGRADE_STATUS_UNSPECIFIED)
	}
	if int32(identityv1.UpgradeStatus_UPGRADE_STATUS_PENDING) != 1 {
		t.Errorf("expected UPGRADE_STATUS_PENDING = 1, got %d", identityv1.UpgradeStatus_UPGRADE_STATUS_PENDING)
	}
	if int32(identityv1.UpgradeStatus_UPGRADE_STATUS_APPROVED) != 2 {
		t.Errorf("expected UPGRADE_STATUS_APPROVED = 2, got %d", identityv1.UpgradeStatus_UPGRADE_STATUS_APPROVED)
	}
	if int32(identityv1.UpgradeStatus_UPGRADE_STATUS_REJECTED) != 3 {
		t.Errorf("expected UPGRADE_STATUS_REJECTED = 3, got %d", identityv1.UpgradeStatus_UPGRADE_STATUS_REJECTED)
	}
}
