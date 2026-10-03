// Package purpose は、つかいみちを表す（docs/domain/wallet.md）。
package purpose

import (
	"errors"
	"fmt"
)

var (
	// ErrUnknownBuiltin は、アプリが用意したつかいみちにないコードが渡されたときのエラー。
	ErrUnknownBuiltin = errors.New("unknown builtin purpose")
	// ErrEmptyCustomID は、空のおうちで追加したつかいみちの ID が渡されたときのエラー。
	ErrEmptyCustomID = errors.New("custom purpose id is empty")
)

// Builtin は、アプリが用意したつかいみち。おうちで変えたり、しまったり、消したりできない。
type Builtin int

// アプリが用意したつかいみち。
const (
	BuiltinFood Builtin = iota + 1
	BuiltinDailyGoods
	BuiltinClothing
	BuiltinTransport
	BuiltinHealth
	BuiltinEducation
	BuiltinHobby
	BuiltinSocial
	BuiltinHome
	BuiltinOther
)

var builtinCodes = map[Builtin]string{
	BuiltinFood:       "food",
	BuiltinDailyGoods: "daily_goods",
	BuiltinClothing:   "clothing",
	BuiltinTransport:  "transport",
	BuiltinHealth:     "health",
	BuiltinEducation:  "education",
	BuiltinHobby:      "hobby",
	BuiltinSocial:     "social",
	BuiltinHome:       "home",
	BuiltinOther:      "other",
}

// ParseBuiltin は、コードに対応するアプリが用意したつかいみちを返す。
func ParseBuiltin(code string) (Builtin, error) {
	for b, c := range builtinCodes {
		if c == code {
			return b, nil
		}
	}
	return 0, fmt.Errorf("builtin purpose %q: %w", code, ErrUnknownBuiltin)
}

// Code は、つかいみちのコード（"food" など）を返す。
func (b Builtin) Code() string { return builtinCodes[b] }

// CustomID は、おうちで追加したつかいみちを一意に表す ID。
type CustomID struct {
	value string
}

// NewCustomID は、おうちで追加したつかいみちの ID を返す。空文字列なら ErrEmptyCustomID を返す。
func NewCustomID(value string) (CustomID, error) {
	if value == "" {
		return CustomID{}, fmt.Errorf("new custom purpose id: %w", ErrEmptyCustomID)
	}
	return CustomID{value: value}, nil
}

// String は、ID の文字列を返す。
func (id CustomID) String() string { return id.value }

// Ref は、相談や台帳の記録が参照するつかいみち。Builtin と CustomID のどちらかである。
//
//sumtype:decl
type Ref interface {
	isRef()
}

func (Builtin) isRef()  {}
func (CustomID) isRef() {}
