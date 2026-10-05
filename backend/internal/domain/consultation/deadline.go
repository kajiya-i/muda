package consultation

import (
	"time"

	"github.com/kajiya-i/muda/backend/internal/domain/budget"
)

// reportGraceDays は、いいねがそろった月の翌月の何日の終わりまで、買ったよ報告ができるか。
const reportGraceDays = 7

// Deadlines は、いいね済みの相談の 2 つの期限（docs/domain/consultation.md）。
// どちらも「この瞬間以降は期限を過ぎている」という瞬間で表す。
// いいねがそろった月も合わせて持ち、買った金額や使う予定のお金を、どの月のやりくりに数えるかに使う。
type Deadlines struct {
	month    budget.Month
	purchase time.Time
	report   time.Time
}

// NewDeadlines は、いいねがそろった瞬間 likedAt と、そのときのおうちの時間 loc から、
// 2 つの期限を求める。
//
//   - 買える期限：いいねがそろった月の終わり（翌月 1 日の 0 時）
//   - 報告の期限：いいねがそろった月の翌月の 7 日の終わり（翌月 8 日の 0 時）
//
// 「月」は、おうちの時間で区切る。期限はいいねがそろったときに決めて記録し、
// あとからおうちの時間が変わっても変えない。
func NewDeadlines(likedAt time.Time, loc *time.Location) Deadlines {
	local := likedAt.In(loc)
	// time.Date は月の繰り上がりを正規化するので、12 月の翌月は翌年の 1 月になる。
	nextMonth := time.Date(local.Year(), local.Month()+1, 1, 0, 0, 0, 0, loc)
	return Deadlines{
		month:    budget.MonthOf(likedAt, loc),
		purchase: nextMonth,
		report:   nextMonth.AddDate(0, 0, reportGraceDays),
	}
}

// Month は、いいねがそろった月を返す。
func (d Deadlines) Month() budget.Month { return d.month }

// Purchase は、買える期限を返す。
func (d Deadlines) Purchase() time.Time { return d.purchase }

// Report は、報告の期限を返す。
func (d Deadlines) Report() time.Time { return d.report }

// ReportExpired は、now の時点で報告の期限を過ぎているか（時間切れか）を返す。
func (d Deadlines) ReportExpired(now time.Time) bool {
	return !now.Before(d.report)
}
