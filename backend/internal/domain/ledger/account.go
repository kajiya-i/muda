// Package ledger は、お金の動きを多通貨の複式の台帳として表す（ADR-0005、ADR-0007）。
// 台帳は家族が目にしない内部の仕組みである（docs/domain/wallet.md）。
package ledger

import (
	"github.com/kajiya-i/muda/backend/internal/domain/household"
	"github.com/kajiya-i/muda/backend/internal/domain/money"
)

// AccountKind は、勘定の種類。
type AccountKind int

// 勘定の種類。
const (
	// KindWallet は、おうちのおさいふのお金。
	KindWallet AccountKind = iota + 1
	// KindOutstanding は、おうちが家族に返すべきお金（おかえし待ち）。家族ごとに待つ。
	KindOutstanding
	// KindExpense は、使われたお金
	KindExpense
	// KindContribution は、おうちの外から入ってきたお金、おうちの外に戻したお金。
	KindContribution
)

// Account は、台帳でお金がある場所。種類、通貨、（おかえし待ちなら）家族の組で決まる。
// 比較可能なので、map のキーに使える。
type Account struct {
	kind     AccountKind
	member   household.MemberID
	currency money.Currency
}

// WalletAccount は、指定した通貨のおうちのおさいふの勘定を返す。
func WalletAccount(currency money.Currency) Account {
	return Account{kind: KindWallet, currency: currency}
}

// OutstandingAccount は、指定した家族と通貨のおかえし待ちの勘定を返す。
func OutstandingAccount(member household.MemberID, currency money.Currency) Account {
	return Account{kind: KindOutstanding, member: member, currency: currency}
}

// ExpenseAccount は、指定した通貨の支出の勘定を返す。
func ExpenseAccount(currency money.Currency) Account {
	return Account{kind: KindExpense, currency: currency}
}

// ContributionAccount は、指定した通貨の搬出の勘定を返す。
func ContributionAccount(currency money.Currency) Account {
	return Account{kind: KindContribution, currency: currency}
}

// Kind は、勘定の種類を返す。
func (a Account) Kind() AccountKind { return a.kind }

// Member は、おかえし待ちの勘定の家族を返す。ほかの種類の勘定では false を返す。
func (a Account) Member() (household.MemberID, bool) {
	return a.member, a.kind == KindOutstanding
}

// Currency は、勘定の通貨を返す。
func (a Account) Currency() money.Currency { return a.currency }
