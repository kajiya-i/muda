package money

import (
	"errors"
	"fmt"
)

var ErrUnsupportedCurrency = errors.New("unsupported currency")

// Currency は、アプリケーションが扱える ISO 4217 の通貨。
type Currency struct {
	code string
	minorUnits int
}

var (
	JPY = Currency{code: "JPY", minorUnits: 0}
	USD = Currency{code: "USD", minorUnits: 2}
)

var supported = []Currency{JPY, USD}

// ParseCurrency は、ISO 4217 の通貨コードに対応する、扱える通貨を返す。
func ParseCurrency(code string) (Currency, error) {
	for _, c := range supported {
		if c.code == code {
			return c, nil
		}
	}
	return Currency{}, fmt.Errorf("currency %q: %w", code, ErrUnsupportedCurrency)
}

// Code は、ISO 4217 の通貨コードを返す。例: JPY, USD
func (c Currency) Code() string { return c.code }

// MinorUnits は、小数点以下の桁数を返す。例: JPY は 0、USD は 2。
func (c Currency) MinorUnits() int { return c.minorUnits}

// String は、fmt.Stringer インターフェースを実装する。ISO 4217 の通貨コードを返す。
func (c Currency) String() string { return c.code }