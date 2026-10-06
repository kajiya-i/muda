package settings_test

import (
	"errors"
	"testing"
	"time"
	_ "time/tzdata" // 実行環境に依存せず、タイムゾーンのデータを使えるようにする

	"github.com/kajiya-i/muda/backend/internal/domain/budget"
	"github.com/kajiya-i/muda/backend/internal/domain/money"
	"github.com/kajiya-i/muda/backend/internal/domain/settings"
)

func location(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

func amount(t *testing.T, minor int64, c money.Currency) money.NonNegativeMoney {
	t.Helper()
	n, err := money.NewNonNegative(minor, c)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// newHistory は、2026 年 10 月 1 日に、今月のやりくり 10 万円、どきどきライン 1 万円でつくったおうちの履歴を返す。
func newHistory(t *testing.T) (settings.History, *time.Location) {
	t.Helper()
	tokyo := location(t, "Asia/Tokyo")
	s, err := settings.NewSnapshot(money.JPY, tokyo, amount(t, 100000, money.JPY), amount(t, 10000, money.JPY))
	if err != nil {
		t.Fatal(err)
	}
	return settings.NewHistory(time.Date(2026, 10, 1, 9, 0, 0, 0, tokyo), s), tokyo
}

func appendAll(t *testing.T, h settings.History, entries ...settings.Entry) settings.History {
	t.Helper()
	for _, e := range entries {
		var err error
		if h, err = h.Append(e); err != nil {
			t.Fatal(err)
		}
	}
	return h
}

func TestNewSnapshotErrors(t *testing.T) {
	tokyo := location(t, "Asia/Tokyo")
	if _, err := settings.NewSnapshot(money.JPY, nil, amount(t, 0, money.JPY), amount(t, 0, money.JPY)); !errors.Is(err, settings.ErrNoLocation) {
		t.Errorf("nil location error = %v, want %v", err, settings.ErrNoLocation)
	}
	if _, err := settings.NewSnapshot(money.JPY, tokyo, amount(t, 0, money.USD), amount(t, 0, money.JPY)); !errors.Is(err, money.ErrCurrencyMismatch) {
		t.Errorf("currency mismatch error = %v, want %v", err, money.ErrCurrencyMismatch)
	}
}

func TestBudgetChangeTakesEffectImmediately(t *testing.T) {
	h, tokyo := newHistory(t)
	changedAt := time.Date(2026, 10, 20, 12, 0, 0, 0, tokyo)
	h = appendAll(t, h, settings.NewEntry(changedAt, settings.NewBudgetChange(amount(t, 120000, money.JPY))))

	if got := h.At(changedAt.Add(-time.Second)).MonthlyBudget().Money().Minor(); got != 100000 {
		t.Errorf("before change = %d, want 100000", got)
	}
	if got := h.At(changedAt).MonthlyBudget().Money().Minor(); got != 120000 {
		t.Errorf("at change = %d, want 120000", got)
	}
}

func TestMonthlyBudgetForUsesTheLastValueOfTheMonth(t *testing.T) {
	h, tokyo := newHistory(t)
	h = appendAll(t, h,
		settings.NewEntry(time.Date(2026, 10, 10, 0, 0, 0, 0, tokyo), settings.NewBudgetChange(amount(t, 80000, money.JPY))),
		settings.NewEntry(time.Date(2026, 10, 31, 23, 59, 59, 0, tokyo), settings.NewBudgetChange(amount(t, 90000, money.JPY))),
		settings.NewEntry(time.Date(2026, 11, 1, 0, 0, 0, 0, tokyo), settings.NewBudgetChange(amount(t, 50000, money.JPY))),
	)

	oct := budget.MonthOf(time.Date(2026, 10, 15, 0, 0, 0, 0, tokyo), tokyo)
	nov := budget.MonthOf(time.Date(2026, 11, 15, 0, 0, 0, 0, tokyo), tokyo)
	if got := h.MonthlyBudgetFor(oct).Money().Minor(); got != 90000 {
		t.Errorf("October = %d, want 90000 (the last value in October)", got)
	}
	if got := h.MonthlyBudgetFor(nov).Money().Minor(); got != 50000 {
		t.Errorf("November = %d, want 50000", got)
	}
}

func TestCurrencyChangeTakesEffectNextMonth(t *testing.T) {
	h, tokyo := newHistory(t)
	likedAt := time.Date(2026, 12, 20, 12, 0, 0, 0, tokyo)
	change := settings.NewCurrencyChange(money.USD, amount(t, 300000, money.USD), amount(t, 5000, money.USD))
	effectiveAt := settings.EffectiveAt(change, likedAt, tokyo)

	if want := time.Date(2027, 1, 1, 0, 0, 0, 0, tokyo); !effectiveAt.Equal(want) {
		t.Fatalf("EffectiveAt = %v, want %v", effectiveAt, want)
	}
	h = appendAll(t, h, settings.NewEntry(effectiveAt, change))

	before := h.At(effectiveAt.Add(-time.Nanosecond))
	if before.Currency() != money.JPY || before.MonthlyBudget().Money().Minor() != 100000 {
		t.Errorf("before = %v %d, want JPY 100000", before.Currency(), before.MonthlyBudget().Money().Minor())
	}
	after := h.At(effectiveAt)
	if after.Currency() != money.USD || after.MonthlyBudget().Money().Minor() != 300000 || after.ThrillLine().Money().Minor() != 5000 {
		t.Errorf("after = %v %d %d, want USD 300000 5000",
			after.Currency(), after.MonthlyBudget().Money().Minor(), after.ThrillLine().Money().Minor())
	}
}

func TestAppendRejectsAmountsInTheWrongCurrency(t *testing.T) {
	h, tokyo := newHistory(t)
	currencyAt := time.Date(2026, 11, 1, 0, 0, 0, 0, tokyo)
	h = appendAll(t, h, settings.NewEntry(currencyAt,
		settings.NewCurrencyChange(money.USD, amount(t, 300000, money.USD), amount(t, 5000, money.USD))))

	tests := []struct {
		name    string
		entry   settings.Entry
		wantErr error
	}{
		{
			name:  "yen budget before the currency change",
			entry: settings.NewEntry(currencyAt.Add(-time.Hour), settings.NewBudgetChange(amount(t, 90000, money.JPY))),
		},
		{
			name:    "yen budget after the currency change",
			entry:   settings.NewEntry(currencyAt.Add(time.Hour), settings.NewBudgetChange(amount(t, 90000, money.JPY))),
			wantErr: money.ErrCurrencyMismatch,
		},
		{
			name:  "dollar thrill line after the currency change",
			entry: settings.NewEntry(currencyAt.Add(time.Hour), settings.NewThrillLineChange(amount(t, 3000, money.USD))),
		},
		{
			name:    "before the household was created",
			entry:   settings.NewEntry(time.Date(2026, 9, 30, 0, 0, 0, 0, tokyo), settings.NewBudgetChange(amount(t, 1, money.JPY))),
			wantErr: settings.ErrBeforeCreation,
		},
		{
			name:    "time zone change without location",
			entry:   settings.NewEntry(currencyAt.Add(time.Hour), settings.NewTimeZoneChange(nil)),
			wantErr: settings.ErrNoLocation,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := h.Append(tt.entry); !errors.Is(err, tt.wantErr) {
				t.Errorf("Append error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestTimeZoneChange(t *testing.T) {
	h, tokyo := newHistory(t)
	newYork := location(t, "America/New_York")
	changedAt := time.Date(2026, 10, 20, 12, 0, 0, 0, tokyo)
	h = appendAll(t, h, settings.NewEntry(settings.EffectiveAt(settings.NewTimeZoneChange(newYork), changedAt, tokyo), settings.NewTimeZoneChange(newYork)))

	if got := h.At(changedAt).Location(); got != newYork {
		t.Errorf("Location = %v, want America/New_York", got)
	}
}

func TestAppendDoesNotChangeOriginal(t *testing.T) {
	h, tokyo := newHistory(t)
	at := time.Date(2026, 10, 20, 0, 0, 0, 0, tokyo)
	if _, err := h.Append(settings.NewEntry(at, settings.NewBudgetChange(amount(t, 1, money.JPY)))); err != nil {
		t.Fatal(err)
	}
	if got := h.At(at).MonthlyBudget().Money().Minor(); got != 100000 {
		t.Errorf("original was changed: budget = %d, want 100000", got)
	}
}
