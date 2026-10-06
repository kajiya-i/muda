// Package proposal は、おねがいを表す（docs/domain/proposal.md）。
package proposal

import (
	"errors"
	"fmt"
	"time"

	"github.com/kajiya-i/muda/backend/internal/domain/household"
	"github.com/kajiya-i/muda/backend/internal/domain/money"
	"github.com/kajiya-i/muda/backend/internal/domain/settings"
)

var (
	// ErrEmptyID は、空のおねがい ID が渡されたときのエラー。
	ErrEmptyID = errors.New("proposal id is empty")
	// ErrSameTimeZone は、今と同じおうちの時間に変えようとしたときのエラー。
	ErrSameTimeZone = errors.New("time zone is the same as the current one")
	// ErrInsufficientWallet は、おうちのおさいふのお金を超えてだそうとしたときのエラー。
	ErrInsufficientWallet = errors.New("amount exceeds the money in the wallet")
)

// ID は、おねがいを一意に表す ID。
type ID struct {
	value string
}

// NewID は、おねがい ID を返す。空文字列なら ErrEmptyID を返す。
func NewID(value string) (ID, error) {
	if value == "" {
		return ID{}, fmt.Errorf("new proposal id: %w", ErrEmptyID)
	}
	return ID{value: value}, nil
}

// String は、おねがい ID の文字列を返す。
func (id ID) String() string { return id.value }

// Kind は、おねがいの種類と内容。おうちのお金を変えるおねがいは、MVP の後に加える。
//
//sumtype:decl
type Kind interface {
	isKind()
}

// ChangeMonthlyBudget は、今月のやりくりを変えるおねがい。
type ChangeMonthlyBudget struct {
	amount money.NonNegativeMoney
}

// ChangeThrillLine は、どきどきラインを変えるおねがい。
type ChangeThrillLine struct {
	amount money.NonNegativeMoney
}

// ChangeTimeZone は、おうちの時間を変えるおねがい。
type ChangeTimeZone struct {
	location *time.Location
}

// TakeOut は、おさいふからだすおねがい。
type TakeOut struct {
	amount money.PositiveMoney
}

// MakeKeeper は、家族をおさいふ係にするおねがい。
type MakeKeeper struct {
	member household.MemberID
}

// RemoveKeeper は、家族をおさいふ係から外すおねがい。
type RemoveKeeper struct {
	member household.MemberID
}

func (ChangeMonthlyBudget) isKind() {}
func (ChangeThrillLine) isKind()    {}
func (ChangeTimeZone) isKind()      {}
func (TakeOut) isKind()             {}
func (MakeKeeper) isKind()          {}
func (RemoveKeeper) isKind()        {}

// NewChangeMonthlyBudget は、今月のやりくりを変えるおねがいの内容を返す。
func NewChangeMonthlyBudget(amount money.NonNegativeMoney) ChangeMonthlyBudget {
	return ChangeMonthlyBudget{amount: amount}
}

// NewChangeThrillLine は、どきどきラインを変えるおねがいの内容を返す。
func NewChangeThrillLine(amount money.NonNegativeMoney) ChangeThrillLine {
	return ChangeThrillLine{amount: amount}
}

// NewChangeTimeZone は、おうちの時間を変えるおねがいの内容を返す。
func NewChangeTimeZone(location *time.Location) ChangeTimeZone {
	return ChangeTimeZone{location: location}
}

// NewTakeOut は、おさいふからだすおねがいの内容を返す。
func NewTakeOut(amount money.PositiveMoney) TakeOut {
	return TakeOut{amount: amount}
}

// NewMakeKeeper は、家族をおさいふ係にするおねがいの内容を返す。
func NewMakeKeeper(member household.MemberID) MakeKeeper {
	return MakeKeeper{member: member}
}

// NewRemoveKeeper は、家族をおさいふ係から外すおねがいの内容を返す。
func NewRemoveKeeper(member household.MemberID) RemoveKeeper {
	return RemoveKeeper{member: member}
}

// Amount は、新しい今月のやりくりを返す。
func (k ChangeMonthlyBudget) Amount() money.NonNegativeMoney { return k.amount }

// Amount は、新しいどきどきラインを返す。
func (k ChangeThrillLine) Amount() money.NonNegativeMoney { return k.amount }

// Location は、新しいおうちの時間を返す。
func (k ChangeTimeZone) Location() *time.Location { return k.location }

// Amount は、だす金額を返す。
func (k TakeOut) Amount() money.PositiveMoney { return k.amount }

// Member は、おさいふ係にする家族を返す。
func (k MakeKeeper) Member() household.MemberID { return k.member }

// Member は、おさいふ係から外す家族を返す。
func (k RemoveKeeper) Member() household.MemberID { return k.member }

// Context は、おねがいを確かめるときのおうちのようす。出すときと、いいねのときに用意する。
type Context struct {
	Household household.Household
	Settings  settings.Snapshot
	Wallet    money.Money
}

// Check は、おねがいの内容が、今のおうちのようすで認められるかを確かめる（docs/domain/proposal.md）。
func Check(k Kind, ctx Context) error {
	switch k := k.(type) {
	case ChangeMonthlyBudget:
		return sameCurrency(k.amount.Money(), ctx.Settings.Currency())
	case ChangeThrillLine:
		return sameCurrency(k.amount.Money(), ctx.Settings.Currency())
	case ChangeTimeZone:
		if k.location == nil {
			return settings.ErrNoLocation
		}
		if k.location.String() == ctx.Settings.Location().String() {
			return ErrSameTimeZone
		}
		return nil
	case TakeOut:
		if err := sameCurrency(k.amount.Money(), ctx.Settings.Currency()); err != nil {
			return err
		}
		c, err := k.amount.Money().Compare(ctx.Wallet)
		if err != nil {
			return err
		}
		if c > 0 {
			return ErrInsufficientWallet
		}
		return nil
	case MakeKeeper:
		_, err := ctx.Household.MakeKeeper(k.member)
		return err
	case RemoveKeeper:
		_, err := ctx.Household.RemoveKeeper(k.member)
		return err
	}
	panic("proposal: unknown kind")
}

func sameCurrency(m money.Money, c money.Currency) error {
	if m.Currency() != c {
		return fmt.Errorf("%s and %s: %w", m.Currency(), c, money.ErrCurrencyMismatch)
	}
	return nil
}
