package ledger

import (
	"errors"
	"fmt"

	"github.com/kajiya-i/muda/backend/internal/domain/household"
	"github.com/kajiya-i/muda/backend/internal/domain/money"
)

// ErrSameAccount は、移動元と移動先が同じ勘定のときのエラー。
var ErrSameAccount = errors.New("from and to are the same account")

// Transfer は、ある勘定から別の勘定へ、正の金額のお金が動いた記録。
// 1 つの移動の中で、出ていく金額と入る金額は同じなので、どの移動の組でも
// 通貨ごとの会計は必ず 0 になる（ADR-0007）。
type Transfer struct {
	from   Account
	to     Account
	amount money.PositiveMoney
}

// NewTransfer は、移動を返す。移動元と移動先が同じとき、または勘定と金額の
// 通貨が一致しない時はエラーを返す。
func NewTransfer(from, to Account, amount money.PositiveMoney) (Transfer, error) {
	if from == to {
		return Transfer{}, fmt.Errorf("new transfer: %w", ErrSameAccount)
	}
	currency := amount.Money().Currency()
	if from.currency != currency || to.currency != currency {
		return Transfer{}, fmt.Errorf("new transfer from %s to %s with %s: %w",
			from.currency, to.currency, currency, money.ErrCurrencyMismatch)
	}
	return Transfer{from: from, to: to, amount: amount}, nil
}

// From は、移動元の勘定を返す
func (t Transfer) From() Account { return t.from }

// To は、移動先の勘定を返す。
func (t Transfer) To() Account { return t.to }

// Amount は、移動した金額を返す。
func (t Transfer) Amount() money.PositiveMoney { return t.amount }

// 以下は、docs/domain/wallet.md の「移動」の表を、お金の動きごとの関数にしたもの。
// 勘定は金額の通貨から決まるので、エラーは起きない。

// TopUp は、おさいふいれる移動（搬出 → おさいふ）を返す。
func TopUp(amount money.PositiveMoney) Transfer {
	c := amount.Money().Currency()
	return Transfer{from: ContributionAccount(c), to: WalletAccount(c), amount: amount}
}

// TakeOut は、おさいふからだす移動（おさいふ → 搬出）を返す。
func TakeOut(amount money.PositiveMoney) Transfer {
	c := amount.Money().Currency()
	return Transfer{from: WalletAccount(c), to: ContributionAccount(c), amount: amount}
}

// WalletPayment は、おさいふから払う移動（おさいふ → 支出）を返す。
func WalletPayment(amount money.PositiveMoney) Transfer {
	c := amount.Money().Currency()
	return Transfer{from: WalletAccount(c), to: ExpenseAccount(c), amount: amount}
}

// Advance は、たてかえの移動（おかえし待ち → 支出）を返す。
func Advance(member household.MemberID, amount money.PositiveMoney) Transfer {
	c := amount.Money().Currency()
	return Transfer{from: OutstandingAccount(member, c), to: ExpenseAccount(c), amount: amount}
}

// Repayment は、たてかえのおかえしの移動（おさいふ → おかえし待ち）を返す。
func Repayment(member household.MemberID, amount money.PositiveMoney) Transfer {
	c := amount.Money().Currency()
	return Transfer{from: WalletAccount(c), to: OutstandingAccount(member, c), amount: amount}
}
