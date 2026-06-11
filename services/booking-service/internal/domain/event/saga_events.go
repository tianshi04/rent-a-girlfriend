package event

import "time"

// TransferToEscrowCommand is raised by the SAGA coordinator to request Escrow coin hold.
type TransferToEscrowCommand struct {
	BookingID   string    `json:"bookingId"`
	ClientID    string    `json:"clientId"`
	CompanionID string    `json:"companionId"`
	Amount      int64     `json:"amount"`
	Timestamp   time.Time `json:"timestamp"`
}

func (e TransferToEscrowCommand) EventType() string     { return "finance.transfer-to-escrow.v1" }
func (e TransferToEscrowCommand) OccurredAt() time.Time { return e.Timestamp }

// CreateChatRoomCommand is raised by the SAGA coordinator to request chatroom creation.
type CreateChatRoomCommand struct {
	BookingID   string    `json:"bookingId"`
	ClientID    string    `json:"clientId"`
	CompanionID string    `json:"companionId"`
	Timestamp   time.Time `json:"timestamp"`
}

func (e CreateChatRoomCommand) EventType() string     { return "interaction.create-chat-room.v1" }
func (e CreateChatRoomCommand) OccurredAt() time.Time { return e.Timestamp }

// RefundEscrowCommand is raised by the SAGA coordinator as a compensating action to refund escrowed coins.
type RefundEscrowCommand struct {
	BookingID   string    `json:"bookingId"`
	ClientID    string    `json:"clientId"`
	CompanionID string    `json:"companionId"`
	Amount      int64     `json:"amount"`
	Timestamp   time.Time `json:"timestamp"`
}

func (e RefundEscrowCommand) EventType() string     { return "finance.refund-escrow.v1" }
func (e RefundEscrowCommand) OccurredAt() time.Time { return e.Timestamp }
