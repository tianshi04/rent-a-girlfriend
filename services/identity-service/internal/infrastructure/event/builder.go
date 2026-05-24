// Package event cung cấp builder để tạo CloudEvents envelope cho các domain event
// của identity-service theo spec CloudEvents v1.0.
//
// Luồng sử dụng:
//
//	evt, err := event.BuildAccountLocked(userID, reason)
//	if err != nil { return err }
//	return outbox.Save(ctx, tx, evt) // lưu vào outbox trong cùng transaction
//
// Builder dùng protojson.Marshal để serialize payload — type-safe, backward-compatible.
package event

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	pb "github.com/rent-a-girlfriend/identity-service/gen/proto"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// CloudEvent là envelope theo CloudEvents spec v1.0.
// Dùng chung cho tất cả domain events trước khi đưa vào Outbox.
type CloudEvent struct {
	SpecVersion     string          `json:"specversion"`
	ID              string          `json:"id"`
	Source          string          `json:"source"`
	Type            string          `json:"type"`
	DataContentType string          `json:"datacontenttype"`
	Time            time.Time       `json:"time"`
	CorrelationID   string          `json:"correlationId,omitempty"`
	Data            json.RawMessage `json:"data"`
}

// ─── Public Builders ──────────────────────────────────────────────────────────

// BuildAccountLocked tạo CloudEvent cho domain event AccountLocked.
// Phát ra khi Admin khóa tài khoản hoặc hệ thống tự động khóa do vi phạm quá ngưỡng.
// Kafka topic: com.rentagf.identity.AccountLocked.v1
func BuildAccountLocked(userID, reason string) (*CloudEvent, error) {
	return build(
		"com.rentagf.identity.AccountLocked.v1",
		fmt.Sprintf("/rent-a-gf/identity-context/account/%s", userID),
		&pb.AccountLockedPayload{UserId: userID, Reason: reason},
	)
}

// BuildViolationRecorded tạo CloudEvent cho domain event ViolationRecorded.
// Phát ra sau mỗi lần ghi nhận vi phạm.
// Kafka topic: com.rentagf.identity.ViolationRecorded.v1
func BuildViolationRecorded(userID string, currentCount int32) (*CloudEvent, error) {
	return build(
		"com.rentagf.identity.ViolationRecorded.v1",
		fmt.Sprintf("/rent-a-gf/identity-context/account/%s", userID),
		&pb.ViolationRecordedPayload{UserId: userID, CurrentCount: currentCount},
	)
}

// BuildUpgradeRequested tạo CloudEvent cho domain event UpgradeRequested.
// Phát ra khi Client gửi yêu cầu nâng cấp thành Companion.
// Kafka topic: com.rentagf.identity.UpgradeRequested.v1
func BuildUpgradeRequested(userID, requestID, reason string) (*CloudEvent, error) {
	return build(
		"com.rentagf.identity.UpgradeRequested.v1",
		fmt.Sprintf("/rent-a-gf/identity-context/upgrade/%s", requestID),
		&pb.UpgradeRequestedPayload{UserId: userID, RequestId: requestID, Reason: reason},
	)
}

// BuildUpgradeApproved tạo CloudEvent cho domain event UpgradeApproved.
// Phát ra khi Admin duyệt yêu cầu → profile-service sẽ tạo Companion profile.
// Kafka topic: com.rentagf.identity.UpgradeApproved.v1
func BuildUpgradeApproved(userID, requestID, approvedBy string) (*CloudEvent, error) {
	return build(
		"com.rentagf.identity.UpgradeApproved.v1",
		fmt.Sprintf("/rent-a-gf/identity-context/upgrade/%s", requestID),
		&pb.UpgradeApprovedPayload{UserId: userID, RequestId: requestID, ApprovedBy: approvedBy},
	)
}

// BuildUpgradeRejected tạo CloudEvent cho domain event UpgradeRejected.
// Phát ra khi Admin từ chối yêu cầu nâng cấp.
// Kafka topic: com.rentagf.identity.UpgradeRejected.v1
func BuildUpgradeRejected(userID, requestID, rejectedBy, reason string) (*CloudEvent, error) {
	return build(
		"com.rentagf.identity.UpgradeRejected.v1",
		fmt.Sprintf("/rent-a-gf/identity-context/upgrade/%s", requestID),
		&pb.UpgradeRejectedPayload{
			UserId:     userID,
			RequestId:  requestID,
			RejectedBy: rejectedBy,
			Reason:     reason,
		},
	)
}

// ─── Internal ─────────────────────────────────────────────────────────────────

// build là helper nội bộ: serialize proto payload → CloudEvent envelope.
func build(eventType, source string, payload proto.Message) (*CloudEvent, error) {
	// protojson.Marshal — type-safe, camelCase JSON, forward-compatible
	data, err := protojson.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("event.build: marshal payload: %w", err)
	}

	return &CloudEvent{
		SpecVersion:     "1.0",
		ID:              uuid.NewString(),
		Source:          source,
		Type:            eventType,
		DataContentType: "application/json",
		Time:            time.Now().UTC(),
		Data:            json.RawMessage(data),
	}, nil
}
