package settings

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/kajiya-i/muda/backend/internal/domain/budget"
	"github.com/kajiya-i/muda/backend/internal/domain/money"
)

var (
	// ErrBeforeCreation は、おうちをつくる前に有効になる変更を加えようとしたときのエラー。
	ErrBeforeCreation = errors.New("change takes effect before the household was created")
)

// Entry は、履歴の 1 行。いつから有効になるかと、変更の内容を持つ。
type Entry struct {
	effectiveAt time.Time
	change      Change
}

// NewEntry は、履歴の 1 行を返す。
func NewEntry(effectiveAt time.Time, change Change) Entry {
	return Entry{effectiveAt: effectiveAt, change: change}
}

// EffectiveAt は、変更が有効になる瞬間を返す。
func (e Entry) EffectiveAt() time.Time { return e.effectiveAt }

// Change は、変更の内容を返す。
func (e Entry) Change() Change { return e.change }

// History は、おうちの設定の履歴。おうちをつくったときの設定と、その後の変更を、
// 更新せずに追記していく（ADR-0007）。過去の設定は、履歴から求める。
type History struct {
	createdAt time.Time
	initial   Snapshot
	entries   []Entry
}

// NewHistory は、おうちをつくったときの設定から、履歴を始める。
func NewHistory(createdAt time.Time, initial Snapshot) History {
	return History{createdAt: createdAt, initial: initial}
}

// Append は、変更を加えた履歴を返す。受け取った履歴は変えない。
// 変更が有効になる瞬間の、おうちのお金と通貨が合わない今月のやりくりやどきどきラインは加えられない。
func (h History) Append(e Entry) (History, error) {
	if e.effectiveAt.Before(h.createdAt) {
		return History{}, fmt.Errorf("append settings change: %w", ErrBeforeCreation)
	}
	if _, err := apply(h.At(e.effectiveAt), e.change); err != nil {
		return History{}, fmt.Errorf("append settings change: %w", err)
	}
	entries := append(slices.Clone(h.entries), e)
	return History{createdAt: h.createdAt, initial: h.initial, entries: entries}, nil
}

// At は、瞬間 t に有効なおうちの設定を返す。
// 有効になる瞬間の順に変更を適用する。同じ瞬間の変更は、加えた順に適用する。
func (h History) At(t time.Time) Snapshot {
	sorted := slices.Clone(h.entries)
	slices.SortStableFunc(sorted, func(a, b Entry) int { return a.effectiveAt.Compare(b.effectiveAt) })

	s := h.initial
	for _, e := range sorted {
		if e.effectiveAt.After(t) {
			break
		}
		// Append で検証済みなので、ここでエラーにはならない。
		if next, err := apply(s, e.change); err == nil {
			s = next
		}
	}
	return s
}

// MonthlyBudgetFor は、月 m の今月のやりくりを返す。
// 過去の月のやりくりは、その月の最後に有効だった金額とする（docs/domain/budget.md）。
// 月の終わりは、その月の終わりに有効なおうちの時間で区切る。
func (h History) MonthlyBudgetFor(m budget.Month) money.NonNegativeMoney {
	// 月の終わりの瞬間を求めるため、まず翌月のはじめのおうちの時間を、ほぼ同じ瞬間で調べる。
	approx := time.Date(m.Year(), m.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	loc := h.At(approx).location
	end := time.Date(m.Year(), m.Month()+1, 1, 0, 0, 0, 0, loc).Add(-time.Nanosecond)
	return h.At(end).budget
}

func apply(s Snapshot, change Change) (Snapshot, error) {
	switch c := change.(type) {
	case BudgetChange:
		if err := sameCurrency(s.currency, c.amount); err != nil {
			return Snapshot{}, err
		}
		s.budget = c.amount
	case ThrillLineChange:
		if err := sameCurrency(s.currency, c.amount); err != nil {
			return Snapshot{}, err
		}
		s.thrillLine = c.amount
	case TimeZoneChange:
		if c.location == nil {
			return Snapshot{}, ErrNoLocation
		}
		s.location = c.location
	case CurrencyChange:
		if err := sameCurrency(c.currency, c.budget, c.thrillLine); err != nil {
			return Snapshot{}, err
		}
		s.currency, s.budget, s.thrillLine = c.currency, c.budget, c.thrillLine
	}
	return s, nil
}
