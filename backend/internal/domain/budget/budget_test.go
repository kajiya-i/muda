package budget_test

import (
	"errors"
	"testing"
	"time"
	_ "time/tzdata" // 実行環境に依存せず、タイムゾーンのデータを使えるようにする

	"github.com/kajiya-i/muda/backend/internal/domain/budget"
	"github.com/kajiya-i/muda/backend/internal/domain/money"
)

func yen(t *testing.T, minor int64) money.PositiveMoney {
	t.Helper()
	p, err := money.NewPositive(minor, money.JPY)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func monthly(t *testing.T, minor int64) money.NonNegativeMoney {
	t.Helper()
	n, err := money.NewNonNegative(minor, money.JPY)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func october(t *testing.T) budget.Month {
	t.Helper()
	tokyo, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatal(err)
	}
	return budget.MonthOf(time.Date(2026, 10, 15, 0, 0, 0, 0, tokyo), tokyo)
}

func TestMonthOf(t *testing.T) {
	tokyo, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatal(err)
	}
	// UTC では 10 月 31 日だが、東京では 11 月 1 日。
	got := budget.MonthOf(time.Date(2026, 10, 31, 20, 0, 0, 0, time.UTC), tokyo)
	if got.String() != "2026-11" {
		t.Errorf("MonthOf = %s, want 2026-11", got)
	}
}

// docs/domain/budget.md の例の表を、そのまま確かめる。
func TestSummarizeBudgetDocExample(t *testing.T) {
	oct := october(t)
	tests := []struct {
		name          string
		monthly       int64
		effects       []budget.Effect
		wallet        int64
		wantSpent     int64
		wantReserved  int64
		wantRemaining int64
		wantOver      bool
	}{
		{
			name: "start of the month", monthly: 100000, wallet: 150000,
			wantSpent: 0, wantReserved: 0, wantRemaining: 100000,
		},
		{
			name: "30,000 yen consultation is liked", monthly: 100000, wallet: 150000,
			effects:   []budget.Effect{budget.NewEffect(oct, budget.KindReserved, yen(t, 30000))},
			wantSpent: 0, wantReserved: 30000, wantRemaining: 70000,
		},
		{
			name: "bought for 28,000 yen from the wallet", monthly: 100000, wallet: 122000,
			effects:   []budget.Effect{budget.NewEffect(oct, budget.KindSpent, yen(t, 28000))},
			wantSpent: 28000, wantReserved: 0, wantRemaining: 72000,
		},
		{
			name: "monthly budget is reduced to 20,000 yen", monthly: 20000, wallet: 122000,
			effects:   []budget.Effect{budget.NewEffect(oct, budget.KindSpent, yen(t, 28000))},
			wantSpent: 28000, wantReserved: 0, wantRemaining: -8000, wantOver: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := budget.Summarize(monthly(t, tt.monthly), oct, tt.effects)
			if err != nil {
				t.Fatal(err)
			}
			if s.Spent().Minor() != tt.wantSpent || s.Reserved().Minor() != tt.wantReserved || s.Remaining().Minor() != tt.wantRemaining {
				t.Errorf("spent=%d reserved=%d remaining=%d, want %d %d %d",
					s.Spent().Minor(), s.Reserved().Minor(), s.Remaining().Minor(),
					tt.wantSpent, tt.wantReserved, tt.wantRemaining)
			}
			signs, err := budget.DecideSigns(s, money.New(tt.wallet, money.JPY))
			if err != nil {
				t.Fatal(err)
			}
			if signs.OverBudget() != tt.wantOver {
				t.Errorf("OverBudget = %v, want %v", signs.OverBudget(), tt.wantOver)
			}
		})
	}
}

func TestSummarizeIgnoresOtherMonths(t *testing.T) {
	oct := october(t)
	tokyo, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatal(err)
	}
	sep := budget.MonthOf(time.Date(2026, 9, 30, 0, 0, 0, 0, tokyo), tokyo)

	s, err := budget.Summarize(monthly(t, 100000), oct, []budget.Effect{
		budget.NewEffect(sep, budget.KindSpent, yen(t, 50000)),
		budget.NewEffect(oct, budget.KindSpent, yen(t, 1000)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if s.Remaining().Minor() != 99000 {
		t.Errorf("Remaining = %d, want 99000", s.Remaining().Minor())
	}
}

func TestSummarizeCurrencyMismatch(t *testing.T) {
	usd, err := money.NewPositive(100, money.USD)
	if err != nil {
		t.Fatal(err)
	}
	_, err = budget.Summarize(monthly(t, 100000), october(t), []budget.Effect{budget.NewEffect(october(t), budget.KindSpent, usd)})
	if !errors.Is(err, money.ErrCurrencyMismatch) {
		t.Errorf("Summarize error = %v, want %v", err, money.ErrCurrencyMismatch)
	}
}

func TestDecideSigns(t *testing.T) {
	tests := []struct {
		name          string
		monthly       int64
		spent         int64
		reserved      int64
		wallet        int64
		wantOver      bool
		wantPinch     bool
		wantOverdrawn bool
	}{
		{name: "no signs", monthly: 100000, spent: 30000, wallet: 100000},
		{name: "spent equals the budget", monthly: 100000, spent: 100000, wallet: 50000},
		{name: "spent exceeds the budget", monthly: 100000, spent: 100001, wallet: 50000, wantOver: true},
		{name: "remaining equals the wallet", monthly: 100000, spent: 30000, wallet: 70000},
		{name: "remaining exceeds the wallet", monthly: 100000, spent: 30000, wallet: 69999, wantPinch: true},
		{name: "reserved money counts for the pinch", monthly: 100000, reserved: 30000, wallet: 69999, wantPinch: true},
		{name: "wallet is zero", monthly: 0, wallet: 0},
		{name: "wallet is negative", monthly: 0, wallet: -1, wantPinch: true, wantOverdrawn: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oct := october(t)
			var effects []budget.Effect
			if tt.spent > 0 {
				effects = append(effects, budget.NewEffect(oct, budget.KindSpent, yen(t, tt.spent)))
			}
			if tt.reserved > 0 {
				effects = append(effects, budget.NewEffect(oct, budget.KindReserved, yen(t, tt.reserved)))
			}
			s, err := budget.Summarize(monthly(t, tt.monthly), oct, effects)
			if err != nil {
				t.Fatal(err)
			}
			signs, err := budget.DecideSigns(s, money.New(tt.wallet, money.JPY))
			if err != nil {
				t.Fatal(err)
			}
			if signs.OverBudget() != tt.wantOver || signs.WalletPinch() != tt.wantPinch || signs.WalletOverdrawn() != tt.wantOverdrawn {
				t.Errorf("signs = over %v, pinch %v, overdrawn %v; want %v %v %v",
					signs.OverBudget(), signs.WalletPinch(), signs.WalletOverdrawn(),
					tt.wantOver, tt.wantPinch, tt.wantOverdrawn)
			}
		})
	}
}
