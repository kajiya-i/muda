package money_test

import (
	"errors"
	"math"
	"testing"

	"github.com/kajiya-i/muda/backend/internal/domain/money"
)

func TestParseCurrency(t *testing.T) {
	tests := []struct {
		code    string
		want    money.Currency
		wantErr error
	}{
		{code: "JPY", want: money.JPY},
		{code: "USD", want: money.USD},
		{code: "EUR", wantErr: money.ErrUnsupportedCurrency},
		{code: "jpy", wantErr: money.ErrUnsupportedCurrency},
		{code: "", wantErr: money.ErrUnsupportedCurrency},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			got, err := money.ParseCurrency(tt.code)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ParseCurrency(%q) error = %v, want %v", tt.code, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("ParseCurrency(%q) = %v, want %v", tt.code, got, tt.want)
			}
		})
	}
}

func TestMoneyAdd(t *testing.T) {
	tests := []struct {
		name    string
		a, b    money.Money
		want    money.Money
		wantErr error
	}{
		{name: "positive", a: money.New(1000, money.JPY), b: money.New(480, money.JPY), want: money.New(1480, money.JPY)},
		{name: "negative", a: money.New(1000, money.JPY), b: money.New(-1480, money.JPY), want: money.New(-480, money.JPY)},
		{name: "currency mismatch", a: money.New(1000, money.JPY), b: money.New(1000, money.USD), wantErr: money.ErrCurrencyMismatch},
		{name: "overflow", a: money.New(math.MaxInt64, money.JPY), b: money.New(1, money.JPY), wantErr: money.ErrOverflow},
		{name: "underflow", a: money.New(math.MinInt64, money.JPY), b: money.New(-1, money.JPY), wantErr: money.ErrOverflow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.a.Add(tt.b)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Add error = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("Add = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMoneySub(t *testing.T) {
	got, err := money.New(1000, money.JPY).Sub(money.New(1480, money.JPY))
	if err != nil {
		t.Fatalf("Sub error = %v", err)
	}
	if want := money.New(-480, money.JPY); got != want {
		t.Errorf("Sub = %v, want %v", got, want)
	}

	if _, err := money.New(0, money.JPY).Sub(money.New(math.MinInt64, money.JPY)); !errors.Is(err, money.ErrOverflow) {
		t.Errorf("Sub(MinInt64) error = %v, want %v", err, money.ErrOverflow)
	}
}

func TestMoneyCompare(t *testing.T) {
	tests := []struct {
		name    string
		a, b    money.Money
		want    int
		wantErr error
	}{
		{name: "less", a: money.New(1, money.JPY), b: money.New(2, money.JPY), want: -1},
		{name: "equal", a: money.New(2, money.JPY), b: money.New(2, money.JPY), want: 0},
		{name: "greater", a: money.New(3, money.JPY), b: money.New(2, money.JPY), want: 1},
		{name: "currency mismatch", a: money.New(1, money.JPY), b: money.New(1, money.USD), wantErr: money.ErrCurrencyMismatch},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.a.Compare(tt.b)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Compare error = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("Compare = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestNewPositive(t *testing.T) {
	tests := []struct {
		minor   int64
		wantErr error
	}{
		{minor: 1},
		{minor: 0, wantErr: money.ErrNotPositive},
		{minor: -1, wantErr: money.ErrNotPositive},
	}
	for _, tt := range tests {
		got, err := money.NewPositive(tt.minor, money.JPY)
		if !errors.Is(err, tt.wantErr) {
			t.Fatalf("NewPositive(%d) error = %v, want %v", tt.minor, err, tt.wantErr)
		}
		if err == nil && got.Money() != money.New(tt.minor, money.JPY) {
			t.Errorf("NewPositive(%d).Money() = %v", tt.minor, got.Money())
		}
	}
}

func TestNewNonNegative(t *testing.T) {
	tests := []struct {
		minor   int64
		wantErr error
	}{
		{minor: 1},
		{minor: 0},
		{minor: -1, wantErr: money.ErrNegative},
	}
	for _, tt := range tests {
		got, err := money.NewNonNegative(tt.minor, money.JPY)
		if !errors.Is(err, tt.wantErr) {
			t.Fatalf("NewNonNegative(%d) error = %v, want %v", tt.minor, err, tt.wantErr)
		}
		if err == nil && got.Money() != money.New(tt.minor, money.JPY) {
			t.Errorf("NewNonNegative(%d).Money() = %v", tt.minor, got.Money())
		}
	}
}
