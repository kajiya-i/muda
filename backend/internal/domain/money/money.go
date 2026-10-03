// Package money は、金額を最小単位の整数と通貨の組で表す（ADR-0004）。
package money

import (
	"errors"
	"fmt"
	"math"
)

var (
	// ErrCurrencyMismatch は、異なる通貨の金額を計算・比較しようとしたときのエラー。
	ErrCurrencyMismatch = errors.New("currency mismatch")
	// ErrOverflow は、計算の結果が int64 の範囲を超えたときのエラー。
	ErrOverflow = errors.New("amount overflow")
	// ErrNotPositive は、正の金額が必要なところに 0 以下の値が渡されたときのエラー。
	ErrNotPositive = errors.New("amount is not positive")
	// ErrNegative は、0 以上の金額が必要なところに負の値が渡されたときのエラー。
	ErrNegative = errors.New("amount is negative")
)

// Money は、符号付きの金額。台帳の計算や残高に使う。
type Money struct {
	minor int64
	currency Currency
}

// New は、最小単位で表した符号付きの金額を返す。
func New(minor int64, currency Currency) Money {
	return Money{minor: minor, currency: currency}
}

// Zero は、指定した通貨の 0 の金額を返す。
func Zero(currency Currency) Money {
	return Money{minor: 0, currency: currency}
}

// Minor は、最小単位で表した金額を返す。例: 100 JPY は 100、1.23 USD は 123。
func (m Money) Minor() int64 { return m.minor }

// Currency は、金額の通貨を返す。
func (m Money) Currency() Currency { return m.currency }

// IsZero は、金額が 0  かどうかを返す。
func (m Money) IsZero() bool { return m.minor == 0 }

// IsNegative は、金額が 0 より小さいかどうかを返す。
func (m Money) IsNegative() bool { return m.minor < 0 }

// IsPositive は、金額が 0 より大きいかどうかを返す。
func (m Money) IsPositive() bool { return m.minor > 0 }

// Add は、金額を加算する。通貨が異なる場合は ErrCurrencyMismatch を返す。
// 計算結果が int64 の範囲を超える場合は ErrOverflow を返す。
func (m Money) Add(other Money) (Money, error) {
	if err := m.sameCurrency(other); err != nil {
		return Money{}, err
	}
	if (other.minor > 0 && m.minor > math.MaxInt64-other.minor) ||
		(other.minor < 0 && m.minor < math.MinInt64-other.minor) {
			return Money{}, fmt.Errorf("add %d and %d: %w", m.minor, other.minor, ErrOverflow)
		}
	return Money{minor: m.minor + other.minor, currency: m.currency}, nil
}

// Sub は、金額を減算する。通貨が異なる場合は ErrCurrencyMismatch を返す。
// 計算結果が int64 の範囲を超える場合は ErrOverflow を返す。
func (m Money) Sub(other Money) (Money, error) {
	negated, err := other.Negate()
	if err != nil {
		return Money{}, err
	}
	return m.Add(negated)
}

// Negate は、金額の符号を反転する。
// 計算結果が int64 の範囲を超える場合は ErrOverflow を返す。
func (m Money) Negate() (Money, error) {
	if m.minor == math.MinInt64 {
		return Money{}, fmt.Errorf("negate %d: %w", m.minor, ErrOverflow)
	}
	return Money{minor: -m.minor, currency: m.currency}, nil
}

// Compare は、金額を比較する。
// 通貨が異なる場合は ErrCurrencyMismatch を返す。
// 戻り値は、a < b の場合 -1、a == b の場合 0、a > b の場合 1。
func (m Money) Compare(other Money) (int, error) {
	if err := m.sameCurrency(other); err != nil {
		return 0, err
	}
	switch {
	case m.minor < other.minor:
		return -1, nil
	case m.minor > other.minor:
		return 1, nil
	default:
		return 0, nil
	}
}

// sameCurrency は、金額の通貨が同じかどうかを確認する。
// 異なる場合は ErrCurrencyMismatch を返す。
func (m Money) sameCurrency(other Money) error {
	if m.currency != other.currency {
		return fmt.Errorf("%s and %s: %w", m.currency, other.currency, ErrCurrencyMismatch)
	}
	return nil
}

// PositiveMoney は、0 より大きい金額。相談した金額や買った金額など、
// 家族が入力する金額に使う。
type PositiveMoney struct {
	value Money
}

// NewPositive は、正の金額を返す。minor が 0 以下なら ErrNotPisitive を返す。
func NewPositive(minor int64, currency Currency) (PositiveMoney, error) {
	if minor <= 0 {
		return PositiveMoney{}, fmt.Errorf("%d %s: %w", minor, currency, ErrNotPositive)
	}
	return PositiveMoney{value: New(minor, currency)}, nil
}

// Money は、符号付きの金額に変換する。逆向きの変換はできない。
func (p PositiveMoney) Money() Money { return p.value }

// NonNegativeMoney は、0 以上の金額。今月のやりくりやどきどきラインに使う。
type NonNegativeMoney struct {
	value Money
}

// NewNonNegative は、0 以上の金額を返す。minor が負なら ErrNegative を返す。
func NewNonNegative(minor int64, currency Currency) (NonNegativeMoney, error) {
	if minor < 0 {
		return NonNegativeMoney{}, fmt.Errorf("%d %s: %w", minor, currency, ErrNegative)
	}
	return NonNegativeMoney{value: New(minor, currency)}, nil
}

// Money は、符号付きの金額に変換する。逆向きの変換はできない。
func (n NonNegativeMoney) Money() Money { return n.value }
