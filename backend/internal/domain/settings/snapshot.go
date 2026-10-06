// Package settings は、おうちの設定（今月のやりくり、どきどきライン、おうちの時間、おうちのお金）と、
// その履歴を表す（docs/domain/household.md、docs/domain/budget.md、ADR-0007）。
package settings

import (
	"errors"
	"fmt"
	"time"

	"github.com/kajiya-i/muda/backend/internal/domain/money"
)

var (
	// ErrNoLocation は、おうちの時間が指定されていないときのエラー。
	ErrNoLocation = errors.New("location is not set")
)

// Snapshot は、ある瞬間に有効なおうちの設定。
// 今月のやりくりとどきどきラインは、常におうちのお金の金額である。
type Snapshot struct {
	currency   money.Currency
	location   *time.Location
	budget     money.NonNegativeMoney
	thrillLine money.NonNegativeMoney
}

// NewSnapshot は、おうちの設定を返す。今月のやりくりやどきどきラインの通貨が
// おうちのお金と異なるとき、おうちの時間がないときはエラーを返す。
func NewSnapshot(currency money.Currency, location *time.Location,
	monthlyBudget, thrillLine money.NonNegativeMoney,
) (Snapshot, error) {
	if location == nil {
		return Snapshot{}, fmt.Errorf("new snapshot: %w", ErrNoLocation)
	}
	if err := sameCurrency(currency, monthlyBudget, thrillLine); err != nil {
		return Snapshot{}, fmt.Errorf("new snapshot: %w", err)
	}
	return Snapshot{currency: currency, location: location, budget: monthlyBudget, thrillLine: thrillLine}, nil
}

// Currency は、おうちのお金を返す。
func (s Snapshot) Currency() money.Currency { return s.currency }

// Location は、おうちの時間を返す。
func (s Snapshot) Location() *time.Location { return s.location }

// MonthlyBudget は、今月のやりくりを返す。
func (s Snapshot) MonthlyBudget() money.NonNegativeMoney { return s.budget }

// ThrillLine は、どきどきラインを返す。
func (s Snapshot) ThrillLine() money.NonNegativeMoney { return s.thrillLine }

func sameCurrency(currency money.Currency, amounts ...money.NonNegativeMoney) error {
	for _, a := range amounts {
		if a.Money().Currency() != currency {
			return fmt.Errorf("%s and %s: %w", a.Money().Currency(), currency, money.ErrCurrencyMismatch)
		}
	}
	return nil
}
