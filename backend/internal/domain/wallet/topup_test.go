package wallet_test

import (
	"errors"
	"testing"
	"time"
	_ "time/tzdata" // 実行環境に依存せず、タイムゾーンのデータを使えるようにする

	"github.com/kajiya-i/muda/backend/internal/domain/household"
	"github.com/kajiya-i/muda/backend/internal/domain/ledger"
	"github.com/kajiya-i/muda/backend/internal/domain/money"
	"github.com/kajiya-i/muda/backend/internal/domain/settings"
	"github.com/kajiya-i/muda/backend/internal/domain/wallet"
)

func TestDecideTopUp(t *testing.T) {
	id := func(v string) household.MemberID {
		m, err := household.NewMemberID(v)
		if err != nil {
			t.Fatal(err)
		}
		return m
	}
	mom, child := id("mom"), id("child")
	h, err := household.New(mom).AddMember(child)
	if err != nil {
		t.Fatal(err)
	}
	tokyo, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatal(err)
	}
	zero, err := money.NewNonNegative(0, money.JPY)
	if err != nil {
		t.Fatal(err)
	}
	s, err := settings.NewSnapshot(money.JPY, tokyo, zero, zero)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 25, 9, 0, 0, 0, tokyo)
	yen, err := money.NewPositive(50000, money.JPY)
	if err != nil {
		t.Fatal(err)
	}
	dollars, err := money.NewPositive(100, money.USD)
	if err != nil {
		t.Fatal(err)
	}

	got, err := wallet.DecideTopUp(h, s, mom, yen, now)
	if err != nil {
		t.Fatal(err)
	}
	if got.By() != mom || !got.At().Equal(now) {
		t.Errorf("TopUp = by %v at %v", got.By(), got.At())
	}
	tr := got.Transfer()
	if tr.From() != ledger.ContributionAccount(money.JPY) || tr.To() != ledger.WalletAccount(money.JPY) || tr.Amount() != yen {
		t.Errorf("transfer = %v -> %v %v", tr.From(), tr.To(), tr.Amount())
	}

	tests := []struct {
		name    string
		by      household.MemberID
		amount  money.PositiveMoney
		wantErr error
	}{
		{name: "not a keeper", by: child, amount: yen, wantErr: household.ErrNotKeeper},
		{name: "another currency", by: mom, amount: dollars, wantErr: money.ErrCurrencyMismatch},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := wallet.DecideTopUp(h, s, tt.by, tt.amount, now); !errors.Is(err, tt.wantErr) {
				t.Errorf("DecideTopUp error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
