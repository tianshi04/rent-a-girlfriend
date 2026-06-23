package event

import (
	"time"

	financev1 "github.com/rent-a-girlfriend/booking-service/gen/proto/financev1"
	interactionv1 "github.com/rent-a-girlfriend/booking-service/gen/proto/interactionv1"
	"google.golang.org/protobuf/proto"
)

// TransferToEscrowCommand is raised by the SAGA coordinator to request Escrow coin hold.
type TransferToEscrowCommand struct {
	*financev1.TransferToEscrowRequest
	Timestamp time.Time
}

func (e TransferToEscrowCommand) EventType() string { return "finance.transfer-to-escrow.v1" }
func (e TransferToEscrowCommand) OccurredAt() time.Time { return e.Timestamp }
func (e TransferToEscrowCommand) ToProto() proto.Message { return e.TransferToEscrowRequest }

// CreateChatRoomCommand is raised by the SAGA coordinator to request chatroom creation.
type CreateChatRoomCommand struct {
	*interactionv1.CreateChatRoomRequest
	Timestamp time.Time
}

func (e CreateChatRoomCommand) EventType() string { return "interaction.create-chat-room.v1" }
func (e CreateChatRoomCommand) OccurredAt() time.Time { return e.Timestamp }
func (e CreateChatRoomCommand) ToProto() proto.Message { return e.CreateChatRoomRequest }

// RefundEscrowCommand is raised by the SAGA coordinator as a compensating action to refund escrowed coins.
type RefundEscrowCommand struct {
	*financev1.RefundEscrowRequest
	Timestamp time.Time
}

func (e RefundEscrowCommand) EventType() string { return "finance.refund-escrow.v1" }
func (e RefundEscrowCommand) OccurredAt() time.Time { return e.Timestamp }
func (e RefundEscrowCommand) ToProto() proto.Message { return e.RefundEscrowRequest }
