package budget

import (
	"fmt"

	"github.com/kajiya-i/muda/backend/internal/domain/money"
)

// Kind は、相談がやりくりに与える影響の種類。
type Kind int

const (
	// KindSpent は、買ったよ報告が済み、使ったお金になったもの。
	KindSpent Kind = iota + 1
	// KindReserved は、いいね済みでまだ買われておらず、使う予定のお金として確保されているもの。
	KindReserved
)

// Effect は、1 つの相談が、ある月のやりくりに与える影響。
type Effect struct {
	month  Month
	kind   Kind
	amount money.PositiveMoney
}

// NewEffect は、相談がやりくりに与える影響を返す。
func NewEffect(month Month, kind Kind, amount money.PositiveMoney) Effect {
	return Effect{month: month, kind: kind, amount: amount}
}

// Summary は、ある月のやりくりのようす。
type Summary struct {
	budget    money.Money
	spent     money.Money
	reserved  money.Money
	remaining money.Money
}

// Budget は、今月のやりくりを返す。
func (s Summary) Budget() money.Money { return s.budget }

// Spent は、今月使ったお金を返す。
func (s Summary) Spent() money.Money { return s.spent }

// Reserved は、使う予定のお金を返す。
func (s Summary) Reserved() money.Money { return s.reserved }

// Remaining は、やりくりの残りを返す。マイナスになることがある。
func (s Summary) Remaining() money.Money { return s.remaining }

// Summarize は、月 month のやりくりのようすを求める。
//
//	やりくりの残り = 今月のやりくり − 今月使ったお金 − 使う予定のお金
//
// effects のうち、month 以外の月のものは無視する。
// 今月使ったお金と使う予定のお金は、いいねがそろった月に数える（docs/domain/budget.md）。
func Summarize(monthly money.NonNegativeMoney, month Month, effects []Effect) (Summary, error) {
	currency := monthly.Money().Currency()
	spent, reserved := money.Zero(currency), money.Zero(currency)
	for _, e := range effects {
		if e.month != month {
			continue
		}
		var err error
		switch e.kind {
		case KindSpent:
			spent, err = spent.Add(e.amount.Money())
		case KindReserved:
			reserved, err = reserved.Add(e.amount.Money())
		default:
			err = fmt.Errorf("unknown effect kind %d", e.kind)
		}
		if err != nil {
			return Summary{}, fmt.Errorf("summarize %s: %w", month, err)
		}
	}

	remaining, err := monthly.Money().Sub(spent)
	if err != nil {
		return Summary{}, fmt.Errorf("summarize %s: %w", month, err)
	}
	if remaining, err = remaining.Sub(reserved); err != nil {
		return Summary{}, fmt.Errorf("summarize %s: %w", month, err)
	}
	return Summary{budget: monthly.Money(), spent: spent, reserved: reserved, remaining: remaining}, nil
}
