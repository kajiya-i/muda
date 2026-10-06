package settings

import (
	"time"

	"github.com/kajiya-i/muda/backend/internal/domain/money"
)

// Change は、おうちの設定の変更。
//
//sumtype:decl
type Change interface {
	isChange()
}

// BudgetChange は、今月のやりくりを変えること。
type BudgetChange struct {
	amount money.NonNegativeMoney
}

// ThrillLineChange は、どきどきラインを変えること。
type ThrillLineChange struct {
	amount money.NonNegativeMoney
}

// TimeZoneChange は、おうちの時間を変えること。
type TimeZoneChange struct {
	location *time.Location
}

// CurrencyChange は、おうちのお金を変えること。今月のやりくりとどきどきラインも、
// 新しいおうちのお金で決め直す（docs/domain/household.md）。
type CurrencyChange struct {
	currency   money.Currency
	budget     money.NonNegativeMoney
	thrillLine money.NonNegativeMoney
}

func (BudgetChange) isChange()     {}
func (ThrillLineChange) isChange() {}
func (TimeZoneChange) isChange()   {}
func (CurrencyChange) isChange()   {}

// NewBudgetChange は、今月のやりくりを変えることを返す。
func NewBudgetChange(amount money.NonNegativeMoney) BudgetChange {
	return BudgetChange{amount: amount}
}

// NewThrillLineChange は、どきどきラインを変えることを返す。
func NewThrillLineChange(amount money.NonNegativeMoney) ThrillLineChange {
	return ThrillLineChange{amount: amount}
}

// NewTimeZoneChange は、おうちの時間を変えることを返す。
func NewTimeZoneChange(location *time.Location) TimeZoneChange {
	return TimeZoneChange{location: location}
}

// NewCurrencyChange は、おうちのお金を変えることを返す。
func NewCurrencyChange(currency money.Currency, monthlyBudget, thrillLine money.NonNegativeMoney) CurrencyChange {
	return CurrencyChange{currency: currency, budget: monthlyBudget, thrillLine: thrillLine}
}

// Amount は、新しい今月のやりくりを返す。
func (c BudgetChange) Amount() money.NonNegativeMoney { return c.amount }

// Amount は、新しいどきどきラインを返す。
func (c ThrillLineChange) Amount() money.NonNegativeMoney { return c.amount }

// Location は、新しいおうちの時間を返す。
func (c TimeZoneChange) Location() *time.Location { return c.location }

// Currency は、新しいおうちのお金を返す。
func (c CurrencyChange) Currency() money.Currency { return c.currency }

// MonthlyBudget は、新しいおうちのお金での今月のやりくりを返す。
func (c CurrencyChange) MonthlyBudget() money.NonNegativeMoney { return c.budget }

// ThrillLine は、新しいおうちのお金でのどきどきラインを返す。
func (c CurrencyChange) ThrillLine() money.NonNegativeMoney { return c.thrillLine }

// EffectiveAt は、change が now にいいねがそろったときに、いつから有効になるかを返す。
//
//   - おうちのお金の変更は、いいねがそろった月の翌月の 1 日から（おうちの時間 loc で）。
//   - それ以外の変更は、いいねがそろった瞬間から。
func EffectiveAt(change Change, now time.Time, loc *time.Location) time.Time {
	switch change.(type) {
	case CurrencyChange:
		local := now.In(loc)
		return time.Date(local.Year(), local.Month()+1, 1, 0, 0, 0, 0, loc)
	case BudgetChange, ThrillLineChange, TimeZoneChange:
		return now
	}
	panic("settings: unknown change")
}
