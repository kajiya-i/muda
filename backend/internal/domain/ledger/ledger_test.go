package ledger_test

import (
	"errors"
	"math/rand/v2"
	"testing"

	"github.com/kajiya-i/muda/backend/internal/domain/household"
	"github.com/kajiya-i/muda/backend/internal/domain/ledger"
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

func member(t *testing.T, id string) household.MemberID {
	t.Helper()
	m, err := household.NewMemberID(id)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestNewTransfer(t *testing.T) {
	wallet := ledger.WalletAccount(money.JPY)
	expense := ledger.ExpenseAccount(money.JPY)
	usdExpense := ledger.ExpenseAccount(money.USD)

	tests := []struct {
		name     string
		from, to ledger.Account
		wantErr  error
	}{
		{name: "valid", from: wallet, to: expense},
		{name: "same account", from: wallet, to: wallet, wantErr: ledger.ErrSameAccount},
		{name: "currency mismatch", from: wallet, to: usdExpense, wantErr: money.ErrCurrencyMismatch},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ledger.NewTransfer(tt.from, tt.to, yen(t, 1000))
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("NewTransfer error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// docs/domain/budget.md の例と同じ流れで、各勘定の残高を確かめる。
func TestBalances(t *testing.T) {
	child := member(t, "child")
	transfers := []ledger.Transfer{
		ledger.TopUp(yen(t, 150000)),
		ledger.WalletPayment(yen(t, 28000)),
		ledger.Advance(child, yen(t, 3480)),
		ledger.Repayment(child, yen(t, 3480)),
		ledger.TakeOut(yen(t, 20000)),
	}

	tests := []struct {
		name    string
		account ledger.Account
		want    int64
	}{
		{name: "wallet", account: ledger.WalletAccount(money.JPY), want: 150000 - 28000 - 3480 - 20000},
		{name: "outstanding", account: ledger.OutstandingAccount(child, money.JPY), want: 0},
		{name: "expense", account: ledger.ExpenseAccount(money.JPY), want: 28000 + 3480},
		{name: "contribution", account: ledger.ContributionAccount(money.JPY), want: -150000 + 20000},
		{name: "unused account", account: ledger.WalletAccount(money.USD), want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ledger.BalanceOf(transfers, tt.account)
			if err != nil {
				t.Fatal(err)
			}
			if got.Minor() != tt.want {
				t.Errorf("BalanceOf = %d, want %d", got.Minor(), tt.want)
			}
		})
	}
}

// たてかえだけでおかえしが済んでいないとき、おかえし待ちの残高はマイナスになる。
func TestBalancesOutstandingIsNegativeBeforeRepayment(t *testing.T) {
	child := member(t, "child")
	got, err := ledger.BalanceOf([]ledger.Transfer{ledger.Advance(child, yen(t, 3480))},
		ledger.OutstandingAccount(child, money.JPY))
	if err != nil {
		t.Fatal(err)
	}
	if got.Minor() != -3480 {
		t.Errorf("BalanceOf = %d, want -3480", got.Minor())
	}
}

// ADR-0007 の Compliance：移動から求めた各勘定の残高の合計は、どんな移動の列でも
// 通貨ごとに 0 になる。乱数で作った移動の列で確かめる。
func TestBalancesSumToZeroPerCurrency(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	members := []household.MemberID{member(t, "a"), member(t, "b")}
	currencies := []money.Currency{money.JPY, money.USD}

	for i := range 1000 {
		n := r.IntN(20)
		transfers := make([]ledger.Transfer, 0, n)
		for range n {
			c := currencies[r.IntN(len(currencies))]
			amount, err := money.NewPositive(r.Int64N(1_000_000)+1, c)
			if err != nil {
				t.Fatal(err)
			}
			m := members[r.IntN(len(members))]
			ops := []func() ledger.Transfer{
				func() ledger.Transfer { return ledger.TopUp(amount) },
				func() ledger.Transfer { return ledger.TakeOut(amount) },
				func() ledger.Transfer { return ledger.WalletPayment(amount) },
				func() ledger.Transfer { return ledger.Advance(m, amount) },
				func() ledger.Transfer { return ledger.Repayment(m, amount) },
			}
			transfers = append(transfers, ops[r.IntN(len(ops))]())
		}

		balances, err := ledger.Balances(transfers)
		if err != nil {
			t.Fatal(err)
		}
		sums := map[money.Currency]money.Money{}
		for account, b := range balances {
			c := account.Currency()
			sum, ok := sums[c]
			if !ok {
				sum = money.Zero(c)
			}
			if sums[c], err = sum.Add(b); err != nil {
				t.Fatal(err)
			}
		}
		for c, sum := range sums {
			if !sum.IsZero() {
				t.Fatalf("case %d: sum of %s balances = %d, want 0", i, c, sum.Minor())
			}
		}
	}
}
