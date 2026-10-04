package consultation_test

import (
	"errors"
	"testing"
	"time"

	"github.com/kajiya-i/muda/backend/internal/domain/consultation"
	"github.com/kajiya-i/muda/backend/internal/domain/household"
	"github.com/kajiya-i/muda/backend/internal/domain/ledger"
	"github.com/kajiya-i/muda/backend/internal/domain/money"
	"github.com/kajiya-i/muda/backend/internal/domain/purpose"
)

// liked は、child が相談し、mom がいいねして、いいね済みになった相談を返す。
func (w world) liked(t *testing.T, id string, amount int64) consultation.Consultation {
	t.Helper()
	c, _ := w.like(t, w.submitWithID(t, id, w.twoKeepers, w.child, amount), w.mom, 50000)
	if _, ok := c.State().(consultation.LikedState); !ok {
		t.Fatalf("State = %T, want LikedState", c.State())
	}
	return c
}

// purchased は、いいね済みの相談に買ったよ報告をして、イベントを適用した相談を返す。
func (w world) purchased(t *testing.T, c consultation.Consultation, amount int64, method consultation.PaymentMethod) consultation.Consultation {
	t.Helper()
	e, _, err := consultation.ReportPurchase(c, w.child, yen(t, amount), method, w.now)
	if err != nil {
		t.Fatal(err)
	}
	return evolveAll(t, c, e)
}

func TestReportPurchase(t *testing.T) {
	tests := []struct {
		name     string
		method   consultation.PaymentMethod
		wantFrom ledger.Account
	}{
		{name: "paid from wallet", method: consultation.PaidFromWallet, wantFrom: ledger.WalletAccount(money.JPY)},
		{name: "paid by advance", method: consultation.PaidByAdvance},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := newWorld(t)
			if tt.method == consultation.PaidByAdvance {
				tt.wantFrom = ledger.OutstandingAccount(w.child, money.JPY)
			}
			c := w.liked(t, "c1", 5000)

			e, transfer, err := consultation.ReportPurchase(c, w.child, yen(t, 4800), tt.method, w.now)
			if err != nil {
				t.Fatal(err)
			}
			if e.Amount() != yen(t, 4800) || e.Method() != tt.method {
				t.Errorf("event = %v %v", e.Amount(), e.Method())
			}
			if transfer.From() != tt.wantFrom || transfer.To() != ledger.ExpenseAccount(money.JPY) {
				t.Errorf("transfer = %v -> %v, want %v -> expense", transfer.From(), transfer.To(), tt.wantFrom)
			}
			if p, ok := transfer.Purpose(); !ok || p != purpose.BuiltinClothing {
				t.Errorf("transfer purpose = %v (%v), want clothing", p, ok)
			}
			if transfer.Amount() != yen(t, 4800) {
				t.Errorf("transfer amount = %v, want 4800", transfer.Amount())
			}
			c = evolveAll(t, c, e)
			if _, ok := c.State().(consultation.PurchasedState); !ok {
				t.Errorf("State = %T, want PurchasedState", c.State())
			}
		})
	}
}

func TestReportPurchaseErrors(t *testing.T) {
	w := newWorld(t)
	liked := w.liked(t, "c1", 5000)
	awaiting := w.submit(t, w.twoKeepers, w.child, 12000)
	reportDeadline := time.Date(2026, 11, 8, 0, 0, 0, 0, w.tokyo)

	tests := []struct {
		name    string
		c       consultation.Consultation
		by      household.MemberID
		amount  int64
		method  consultation.PaymentMethod
		now     time.Time
		wantErr error
	}{
		{name: "equal to consulted amount", c: liked, by: w.child, amount: 5000, method: consultation.PaidFromWallet, now: w.now},
		{name: "just before report deadline", c: liked, by: w.child, amount: 5000, method: consultation.PaidFromWallet, now: reportDeadline.Add(-time.Second)},
		{name: "over consulted amount", c: liked, by: w.child, amount: 5001, method: consultation.PaidFromWallet, now: w.now, wantErr: consultation.ErrOverConsultedAmount},
		{name: "at report deadline", c: liked, by: w.child, amount: 5000, method: consultation.PaidFromWallet, now: reportDeadline, wantErr: consultation.ErrReportExpired},
		{name: "someone else reports", c: liked, by: w.mom, amount: 5000, method: consultation.PaidFromWallet, now: w.now, wantErr: consultation.ErrNotRequester},
		{name: "not liked yet", c: awaiting, by: w.child, amount: 5000, method: consultation.PaidFromWallet, now: w.now, wantErr: consultation.ErrNotLiked},
		{name: "unknown payment method", c: liked, by: w.child, amount: 5000, method: 0, now: w.now, wantErr: consultation.ErrUnknownPaymentMethod},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := consultation.ReportPurchase(tt.c, tt.by, yen(t, tt.amount), tt.method, tt.now)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ReportPurchase error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestRepay(t *testing.T) {
	w := newWorld(t)
	first := w.purchased(t, w.liked(t, "c1", 3000), 2980, consultation.PaidByAdvance)
	second := w.purchased(t, w.liked(t, "c2", 1500), 1500, consultation.PaidByAdvance)
	repayment, err := ledger.NewRepaymentID("r1")
	if err != nil {
		t.Fatal(err)
	}

	results, err := consultation.Repay([]consultation.Consultation{first, second}, w.child, w.twoKeepers, w.mom, repayment, w.now)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("results = %d, want 2", len(results))
	}

	// たてかえとおかえしの移動を合わせると、child のおかえし待ちは 0 に戻る。
	transfers := []ledger.Transfer{
		ledger.Advance(w.child, yen(t, 2980), purpose.BuiltinClothing),
		ledger.Advance(w.child, yen(t, 1500), purpose.BuiltinClothing),
	}
	for i, r := range results {
		transfers = append(transfers, r.Transfer())
		c := evolveAll(t, []consultation.Consultation{first, second}[i], r.Event())
		if _, ok := c.State().(consultation.RepaidState); !ok {
			t.Errorf("results[%d]: State = %T, want RepaidState", i, c.State())
		}
	}
	outstanding, err := ledger.BalanceOf(transfers, ledger.OutstandingAccount(w.child, money.JPY))
	if err != nil {
		t.Fatal(err)
	}
	if !outstanding.IsZero() {
		t.Errorf("outstanding = %d, want 0", outstanding.Minor())
	}
}

func TestRepayErrors(t *testing.T) {
	w := newWorld(t)
	advance := w.purchased(t, w.liked(t, "c1", 3000), 3000, consultation.PaidByAdvance)
	walletPaid := w.purchased(t, w.liked(t, "c2", 3000), 3000, consultation.PaidFromWallet)
	repayment, err := ledger.NewRepaymentID("r1")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		cs      []consultation.Consultation
		member  household.MemberID
		by      household.MemberID
		wantErr error
	}{
		{name: "not a keeper", cs: []consultation.Consultation{advance}, member: w.child, by: w.child, wantErr: household.ErrNotKeeper},
		{name: "nothing to repay", member: w.child, by: w.mom, wantErr: consultation.ErrNothingToRepay},
		{name: "paid from wallet", cs: []consultation.Consultation{walletPaid}, member: w.child, by: w.mom, wantErr: consultation.ErrNotRepayable},
		{name: "different member", cs: []consultation.Consultation{advance}, member: w.dad, by: w.mom, wantErr: consultation.ErrNotRequester},
		{name: "duplicate", cs: []consultation.Consultation{advance, advance}, member: w.child, by: w.mom, wantErr: consultation.ErrDuplicateConsultation},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := consultation.Repay(tt.cs, tt.member, w.twoKeepers, tt.by, repayment, w.now); !errors.Is(err, tt.wantErr) {
				t.Errorf("Repay error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
