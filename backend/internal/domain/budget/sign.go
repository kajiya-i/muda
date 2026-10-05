package budget

import (
	"fmt"

	"github.com/kajiya-i/muda/backend/internal/domain/money"
)

// Signs は、家族に出すサイン（docs/domain/budget.md）。
type Signs struct {
	overBudget      bool
	walletPinch     bool
	walletOverdrawn bool
}

// OverBudget は、やりくりオーバー（今月使ったお金が、今月のやりくりを超えている）かを返す。
func (s Signs) OverBudget() bool { return s.overBudget }

// WalletPinch は、おさいふピンチ（やりくりの残りが、おうちのおさいふのお金を超えている）かを返す。
func (s Signs) WalletPinch() bool { return s.walletPinch }

// WalletOverdrawn は、おさいふからっぽ（おうちのおさいふのお金が、マイナスになっている）かを返す。
func (s Signs) WalletOverdrawn() bool { return s.walletOverdrawn }

// DecideSigns は、やりくりのようすと、おうちのおさいふのお金から、出すサインを決める。
// サインは保存せず、必要なときにこの関数で判断する。
func DecideSigns(s Summary, wallet money.Money) (Signs, error) {
	overBudget, err := s.spent.Compare(s.budget)
	if err != nil {
		return Signs{}, fmt.Errorf("decide signs: %w", err)
	}
	pinch, err := s.remaining.Compare(wallet)
	if err != nil {
		return Signs{}, fmt.Errorf("decide signs: %w", err)
	}
	return Signs{
		overBudget:      overBudget > 0,
		walletPinch:     pinch > 0,
		walletOverdrawn: wallet.IsNegative(),
	}, nil
}
