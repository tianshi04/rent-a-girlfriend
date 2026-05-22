package port

import (
	"context"

	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
)

// FinanceService is the port for synchronous calls to Finance Context.
type FinanceService interface {
	// FreezeCoin freezes Kano-Coin in the client's wallet when a booking is requested.
	FreezeCoin(ctx context.Context, clientID vo.ClientID, amount vo.Money) error

	// UnfreezeCoin releases frozen Kano-Coin back to the client's wallet
	// (e.g., on reject or early cancellation).
	UnfreezeCoin(ctx context.Context, clientID vo.ClientID, amount vo.Money) error
}
