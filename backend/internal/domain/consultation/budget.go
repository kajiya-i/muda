package consultation

import (
	"time"

	"github.com/kajiya-i/muda/backend/internal/domain/budget"
)

// BudgetEffect は、相談が now の時点でやりくりに与える影響を返す。影響がなければ false を返す。
//
//   - いいね済みで、時間切れになっていない相談は、相談した金額を使う予定のお金として確保する。
//   - 買ったよ・おかえし済みの相談は、買った金額を使ったお金として数える。
//   - いいね待ち、見送り、とりやめ、時間切れの相談は、影響しない。
//
// どちらも、いいねがそろった月のやりくりに数える（docs/domain/budget.md）。
func BudgetEffect(c Consultation, now time.Time) (budget.Effect, bool) {
	switch s := c.state.(type) {
	case LikedState:
		if s.deadlines.ReportExpired(now) {
			return budget.Effect{}, false
		}
		return budget.NewEffect(s.deadlines.Month(), budget.KindReserved, s.details.amount), true
	case PurchasedState:
		return budget.NewEffect(s.deadlines.Month(), budget.KindSpent, s.amount), true
	case RepaidState:
		return budget.NewEffect(s.deadlines.Month(), budget.KindSpent, s.amount), true
	case AwaitingLikes, PassedState, WithdrawnState:
		return budget.Effect{}, false
	}
	panic("consultation: unknown state")
}
