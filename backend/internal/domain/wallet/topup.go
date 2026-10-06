// Package wallet は、おうちのおさいふに関わる操作を表す（docs/domain/wallet.md）。
package wallet

import (
	"fmt"
	"time"

	"github.com/kajiya-i/muda/backend/internal/domain/household"
	"github.com/kajiya-i/muda/backend/internal/domain/ledger"
	"github.com/kajiya-i/muda/backend/internal/domain/money"
	"github.com/kajiya-i/muda/backend/internal/domain/settings"
)

// TopUp は、おさいふにいれたことの記録。誰が、いつ、いくらいれたかを、家族の誰からも見えるように残す。
type TopUp struct {
	by       household.MemberID
	at       time.Time
	transfer ledger.Transfer
}

// By は、いれたおさいふ係を返す。
func (t TopUp) By() household.MemberID { return t.by }

// At は、いれた日時を返す。
func (t TopUp) At() time.Time { return t.at }

// Transfer は、拠出 → おさいふの台帳の移動を返す。
func (t TopUp) Transfer() ledger.Transfer { return t.transfer }

// DecideTopUp は、おさいふ係 by がおさいふにいれることを決める。いいねは要らない。
// いれる金額は、そのときのおうちのお金でなければならない。
func DecideTopUp(h household.Household, s settings.Snapshot, by household.MemberID,
	amount money.PositiveMoney, now time.Time,
) (TopUp, error) {
	m, ok := h.Member(by)
	if !ok || !m.IsActiveKeeper() {
		return TopUp{}, fmt.Errorf("top up: %w", household.ErrNotKeeper)
	}
	if c := amount.Money().Currency(); c != s.Currency() {
		return TopUp{}, fmt.Errorf("top up %s into %s wallet: %w", c, s.Currency(), money.ErrCurrencyMismatch)
	}
	return TopUp{by: by, at: now, transfer: ledger.TopUp(amount)}, nil
}
