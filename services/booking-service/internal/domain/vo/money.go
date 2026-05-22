package vo

import "fmt"

// Money represents Kano-Coin amount as integer (BR-04).
// 1 Kano-Coin = 1,000 VND.
type Money struct {
	amount int64
}

// NewMoney creates a Money value object. Amount must be non-negative.
func NewMoney(amount int64) (Money, error) {
	if amount < 0 {
		return Money{}, fmt.Errorf("money amount cannot be negative: %d", amount)
	}
	return Money{amount: amount}, nil
}

// MustMoney creates Money or panics. Use only in tests.
func MustMoney(amount int64) Money {
	m, err := NewMoney(amount)
	if err != nil {
		panic(err)
	}
	return m
}

// Amount returns the raw integer value.
func (m Money) Amount() int64 { return m.amount }

// IsZero checks if the amount is zero.
func (m Money) IsZero() bool { return m.amount == 0 }

// Equals checks equality.
func (m Money) Equals(other Money) bool { return m.amount == other.amount }
