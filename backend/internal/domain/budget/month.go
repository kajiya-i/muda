// Package budget は、今月のやりくりを表す（docs/domain/budget.md）。
package budget

import (
	"fmt"
	"time"
)

// Month は、おうちの時間で区切った 1 か月。
type Month struct {
	year  int
	month time.Month
}

// MonthOf は、瞬間 t が、おうちの時間 loc でどの月に入るかを返す。
func MonthOf(t time.Time, loc *time.Location) Month {
	local := t.In(loc)
	return Month{year: local.Year(), month: local.Month()}
}

// Year は、年を返す。
func (m Month) Year() int { return m.year }

// Month は、月を返す。
func (m Month) Month() time.Month { return m.month }

// String は、"2026-10" の形で返す。
func (m Month) String() string { return fmt.Sprintf("%04d-%02d", m.year, int(m.month)) }
