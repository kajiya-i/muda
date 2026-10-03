package ledger

import (
	"errors"
	"fmt"
)

// ErrEmptyRepaymentID は、空のおかえしの ID が渡されたときのエラー。
var ErrEmptyRepaymentID = errors.New("repayment id is empty")

// RepaymentID は、たてかえのおかえしを一意に表す ID。1 回のおかえしで、
// 複数の買い物をまとめておかえしできる（docs/domain/wallet.md）。
type RepaymentID struct {
	value string
}

// NewRepaymentID は、おかえしの ID を返す。空文字列なら ErrEmptyRepaymentID を返す。
func NewRepaymentID(value string) (RepaymentID, error) {
	if value == "" {
		return RepaymentID{}, fmt.Errorf("new repayment id: %w", ErrEmptyRepaymentID)
	}
	return RepaymentID{value: value}, nil
}

// String は、ID の文字列を返す。
func (id RepaymentID) String() string { return id.value }
