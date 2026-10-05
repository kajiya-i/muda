package consultation_test

import (
	"testing"
	"time"

	"github.com/kajiya-i/muda/backend/internal/domain/budget"
	"github.com/kajiya-i/muda/backend/internal/domain/consultation"
	"github.com/kajiya-i/muda/backend/internal/domain/like"
	"github.com/kajiya-i/muda/backend/internal/domain/money"
)

func nonNegativeYen(t *testing.T, minor int64) money.NonNegativeMoney {
	t.Helper()
	n, err := money.NewNonNegative(minor, money.JPY)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func TestBudgetEffect(t *testing.T) {
	w := newWorld(t)
	oct := budget.MonthOf(w.now, w.tokyo)
	reportDeadline := time.Date(2026, 11, 8, 0, 0, 0, 0, w.tokyo)

	pass, err := like.NewPass([]like.ConsultationPassReason{like.ConsultationTooExpensive}, "")
	if err != nil {
		t.Fatal(err)
	}
	passEvent, err := consultation.Pass(w.submit(t, w.twoKeepers, w.child, 3000), w.mom, pass, w.now)
	if err != nil {
		t.Fatal(err)
	}

	awaiting := w.submit(t, w.twoKeepers, w.child, 3000)
	liked := w.liked(t, "c1", 3000)
	purchased := w.purchased(t, w.liked(t, "c2", 3000), 2500, consultation.PaidFromWallet)
	passed := evolveAll(t, w.submit(t, w.twoKeepers, w.child, 3000), passEvent)

	tests := []struct {
		name       string
		c          consultation.Consultation
		now        time.Time
		wantOK     bool
		wantKind   budget.Kind
		wantAmount int64
	}{
		{name: "awaiting likes", c: awaiting, now: w.now},
		{name: "liked is reserved", c: liked, now: w.now, wantOK: true, wantKind: budget.KindReserved, wantAmount: 3000},
		{name: "liked is still reserved in the grace period", c: liked, now: reportDeadline.Add(-time.Second), wantOK: true, wantKind: budget.KindReserved, wantAmount: 3000},
		{name: "expired is not reserved", c: liked, now: reportDeadline},
		{name: "purchased is spent", c: purchased, now: w.now, wantOK: true, wantKind: budget.KindSpent, wantAmount: 2500},
		{name: "passed", c: passed, now: w.now},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, ok := consultation.BudgetEffect(tt.c, tt.now)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			// 効果は、いいねがそろった 10 月のやりくりに数える。
			s, err := budget.Summarize(nonNegativeYen(t, 100000), oct, []budget.Effect{e})
			if err != nil {
				t.Fatal(err)
			}
			got := s.Spent().Minor()
			if tt.wantKind == budget.KindReserved {
				got = s.Reserved().Minor()
			}
			if got != tt.wantAmount {
				t.Errorf("amount = %d, want %d", got, tt.wantAmount)
			}
		})
	}
}

// いいねがそろった月の翌月 7 日までに報告した買い物は、いいねがそろった月の使ったお金に入る。
func TestBudgetEffectCountsInLikedMonth(t *testing.T) {
	w := newWorld(t)
	liked := w.liked(t, "c1", 3000)
	nov3 := time.Date(2026, 11, 3, 10, 0, 0, 0, w.tokyo)
	e, _, err := consultation.ReportPurchase(liked, w.child, yen(t, 3000), consultation.PaidFromWallet, nov3)
	if err != nil {
		t.Fatal(err)
	}
	purchased := evolveAll(t, liked, e)

	effect, ok := consultation.BudgetEffect(purchased, nov3)
	if !ok {
		t.Fatal("no effect")
	}
	oct := budget.MonthOf(w.now, w.tokyo)
	nov := budget.MonthOf(nov3, w.tokyo)
	for _, tt := range []struct {
		month budget.Month
		want  int64
	}{{oct, 3000}, {nov, 0}} {
		s, err := budget.Summarize(nonNegativeYen(t, 100000), tt.month, []budget.Effect{effect})
		if err != nil {
			t.Fatal(err)
		}
		if s.Spent().Minor() != tt.want {
			t.Errorf("%s spent = %d, want %d", tt.month, s.Spent().Minor(), tt.want)
		}
	}
}
