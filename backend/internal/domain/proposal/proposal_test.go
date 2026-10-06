package proposal_test

import (
	"errors"
	"testing"
	"time"
	_ "time/tzdata" // 実行環境に依存せず、タイムゾーンのデータを使えるようにする

	"github.com/kajiya-i/muda/backend/internal/domain/household"
	"github.com/kajiya-i/muda/backend/internal/domain/ledger"
	"github.com/kajiya-i/muda/backend/internal/domain/like"
	"github.com/kajiya-i/muda/backend/internal/domain/money"
	"github.com/kajiya-i/muda/backend/internal/domain/proposal"
	"github.com/kajiya-i/muda/backend/internal/domain/settings"
)

type world struct {
	mom, dad, child household.MemberID
	twoKeepers      household.Household
	oneKeeper       household.Household
	tokyo, newYork  *time.Location
	now             time.Time
	settings        settings.Snapshot
}

func newWorld(t *testing.T) world {
	t.Helper()
	id := func(v string) household.MemberID {
		m, err := household.NewMemberID(v)
		if err != nil {
			t.Fatal(err)
		}
		return m
	}
	mom, dad, child := id("mom"), id("dad"), id("child")
	must := func(h household.Household, err error) household.Household {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return h
	}
	two := must(household.New(mom).AddMember(dad))
	two = must(two.MakeKeeper(dad))
	two = must(two.AddMember(child))
	one := must(household.New(mom).AddMember(child))

	load := func(name string) *time.Location {
		loc, err := time.LoadLocation(name)
		if err != nil {
			t.Fatal(err)
		}
		return loc
	}
	tokyo := load("Asia/Tokyo")
	s, err := settings.NewSnapshot(money.JPY, tokyo, nonNegative(t, 100000, money.JPY), nonNegative(t, 10000, money.JPY))
	if err != nil {
		t.Fatal(err)
	}
	return world{
		mom: mom, dad: dad, child: child, twoKeepers: two, oneKeeper: one,
		tokyo: tokyo, newYork: load("America/New_York"),
		now: time.Date(2026, 10, 20, 12, 0, 0, 0, tokyo), settings: s,
	}
}

func nonNegative(t *testing.T, minor int64, c money.Currency) money.NonNegativeMoney {
	t.Helper()
	n, err := money.NewNonNegative(minor, c)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func positive(t *testing.T, minor int64) money.PositiveMoney {
	t.Helper()
	p, err := money.NewPositive(minor, money.JPY)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func (w world) ctx(h household.Household, wallet int64) proposal.Context {
	return proposal.Context{Household: h, Settings: w.settings, Wallet: money.New(wallet, money.JPY)}
}

func (w world) submit(t *testing.T, proposer household.MemberID, kind proposal.Kind, ctx proposal.Context) proposal.Proposal {
	t.Helper()
	e, err := proposal.Submit(proposer, kind, "", ctx, w.now)
	if err != nil {
		t.Fatal(err)
	}
	id, err := proposal.NewID("p1")
	if err != nil {
		t.Fatal(err)
	}
	return proposal.Start(id, e)
}

func evolve(t *testing.T, p proposal.Proposal, e proposal.Event) proposal.Proposal {
	t.Helper()
	next, err := proposal.Evolve(p, e)
	if err != nil {
		t.Fatal(err)
	}
	return next
}

func TestSubmitErrors(t *testing.T) {
	w := newWorld(t)
	tests := []struct {
		name     string
		proposer household.MemberID
		kind     proposal.Kind
		ctx      proposal.Context
		wantErr  error
	}{
		{name: "non-keeper proposes", proposer: w.child, kind: proposal.NewChangeMonthlyBudget(nonNegative(t, 1, money.JPY)), ctx: w.ctx(w.twoKeepers, 0), wantErr: household.ErrNotKeeper},
		{name: "budget in another currency", proposer: w.mom, kind: proposal.NewChangeMonthlyBudget(nonNegative(t, 1, money.USD)), ctx: w.ctx(w.twoKeepers, 0), wantErr: money.ErrCurrencyMismatch},
		{name: "same time zone", proposer: w.mom, kind: proposal.NewChangeTimeZone(w.tokyo), ctx: w.ctx(w.twoKeepers, 0), wantErr: proposal.ErrSameTimeZone},
		{name: "take out more than the wallet", proposer: w.mom, kind: proposal.NewTakeOut(positive(t, 50001)), ctx: w.ctx(w.twoKeepers, 50000), wantErr: proposal.ErrInsufficientWallet},
		{name: "make a keeper again", proposer: w.mom, kind: proposal.NewMakeKeeper(w.dad), ctx: w.ctx(w.twoKeepers, 0), wantErr: household.ErrAlreadyKeeper},
		{name: "remove the last keeper", proposer: w.mom, kind: proposal.NewRemoveKeeper(w.mom), ctx: w.ctx(w.oneKeeper, 0), wantErr: household.ErrLastKeeper},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := proposal.Submit(tt.proposer, tt.kind, "", tt.ctx, w.now); !errors.Is(err, tt.wantErr) {
				t.Errorf("Submit error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestLikeChangesMonthlyBudget(t *testing.T) {
	w := newWorld(t)
	ctx := w.ctx(w.twoKeepers, 0)
	p := w.submit(t, w.mom, proposal.NewChangeMonthlyBudget(nonNegative(t, 120000, money.JPY)), ctx)

	e, outcome, err := proposal.Like(p, w.dad, ctx, w.now)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := e.(proposal.Liked); !ok {
		t.Errorf("event = %T, want Liked", e)
	}
	changed, ok := outcome.(proposal.SettingsChanged)
	if !ok {
		t.Fatalf("outcome = %T, want SettingsChanged", outcome)
	}
	if !changed.Entry().EffectiveAt().Equal(w.now) {
		t.Errorf("EffectiveAt = %v, want %v", changed.Entry().EffectiveAt(), w.now)
	}
	budget, ok := changed.Entry().Change().(settings.BudgetChange)
	if !ok || budget.Amount().Money().Minor() != 120000 {
		t.Errorf("change = %#v, want budget 120000", changed.Entry().Change())
	}

	p = evolve(t, p, e)
	liked, ok := p.State().(proposal.LikedState)
	if !ok || liked.Liker() != w.dad {
		t.Errorf("State = %+v, want liked by dad", p.State())
	}
}

func TestSelfLikeTakesOut(t *testing.T) {
	w := newWorld(t)
	ctx := w.ctx(w.oneKeeper, 80000)
	p := w.submit(t, w.mom, proposal.NewTakeOut(positive(t, 30000)), ctx)

	e, outcome, err := proposal.Like(p, w.mom, ctx, w.now)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := e.(proposal.SelfLiked); !ok {
		t.Errorf("event = %T, want SelfLiked", e)
	}
	takenOut, ok := outcome.(proposal.TakenOut)
	if !ok {
		t.Fatalf("outcome = %T, want TakenOut", outcome)
	}
	tr := takenOut.Transfer()
	if tr.From() != ledger.WalletAccount(money.JPY) || tr.To() != ledger.ContributionAccount(money.JPY) || tr.Amount() != positive(t, 30000) {
		t.Errorf("transfer = %v -> %v %v", tr.From(), tr.To(), tr.Amount())
	}
}

// おねがいを出したあとに買い物があり、いいねのときにはおさいふのお金が足りなくなっている。
func TestLikeRechecksTheWallet(t *testing.T) {
	w := newWorld(t)
	p := w.submit(t, w.mom, proposal.NewTakeOut(positive(t, 30000)), w.ctx(w.twoKeepers, 50000))

	if _, _, err := proposal.Like(p, w.dad, w.ctx(w.twoKeepers, 29999), w.now); !errors.Is(err, proposal.ErrInsufficientWallet) {
		t.Errorf("Like error = %v, want %v", err, proposal.ErrInsufficientWallet)
	}
	// いいねできなくなったおねがいは、出した家族がとりやめる。
	e, err := proposal.Withdraw(p, w.mom, w.now)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := evolve(t, p, e).State().(proposal.WithdrawnState); !ok {
		t.Error("not withdrawn")
	}
}

// mom が自分をおさいふ係から外すおねがいを出したあと、dad がおさいふ係から外れると、
// いいねのときには mom が最後のおさいふ係になっているので、いいねできない。
func TestLikeRechecksTheLastKeeper(t *testing.T) {
	w := newWorld(t)
	p := w.submit(t, w.mom, proposal.NewRemoveKeeper(w.mom), w.ctx(w.twoKeepers, 0))

	withoutDad, err := w.twoKeepers.RemoveKeeper(w.dad)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := proposal.Like(p, w.dad, w.ctx(withoutDad, 0), w.now); !errors.Is(err, household.ErrLastKeeper) {
		t.Errorf("Like error = %v, want %v", err, household.ErrLastKeeper)
	}
}

func TestLikeMakesKeeper(t *testing.T) {
	w := newWorld(t)
	ctx := w.ctx(w.twoKeepers, 0)
	p := w.submit(t, w.mom, proposal.NewMakeKeeper(w.child), ctx)

	_, outcome, err := proposal.Like(p, w.dad, ctx, w.now)
	if err != nil {
		t.Fatal(err)
	}
	changed, ok := outcome.(proposal.HouseholdChanged)
	if !ok {
		t.Fatalf("outcome = %T, want HouseholdChanged", outcome)
	}
	if m, _ := changed.Household().Member(w.child); !m.IsActiveKeeper() {
		t.Error("child is not a keeper")
	}
}

func TestLikeChangesTimeZone(t *testing.T) {
	w := newWorld(t)
	ctx := w.ctx(w.twoKeepers, 0)
	p := w.submit(t, w.mom, proposal.NewChangeTimeZone(w.newYork), ctx)

	_, outcome, err := proposal.Like(p, w.dad, ctx, w.now)
	if err != nil {
		t.Fatal(err)
	}
	changed, ok := outcome.(proposal.SettingsChanged)
	if !ok {
		t.Fatalf("outcome = %T, want SettingsChanged", outcome)
	}
	if tz, ok := changed.Entry().Change().(settings.TimeZoneChange); !ok || tz.Location() != w.newYork {
		t.Errorf("change = %#v, want time zone New York", changed.Entry().Change())
	}
}

func TestLikePassWithdrawErrors(t *testing.T) {
	w := newWorld(t)
	ctx := w.ctx(w.twoKeepers, 0)
	kind := proposal.NewChangeThrillLine(nonNegative(t, 5000, money.JPY))
	awaiting := w.submit(t, w.mom, kind, ctx)
	e, _, err := proposal.Like(awaiting, w.dad, ctx, w.now)
	if err != nil {
		t.Fatal(err)
	}
	liked := evolve(t, awaiting, e)
	selfRoute := w.submit(t, w.mom, kind, w.ctx(w.oneKeeper, 0))
	pass, err := like.NewPass([]like.ProposalPassReason{like.ProposalTooBigChange}, "")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		do      func() error
		wantErr error
	}{
		{name: "proposer likes own proposal", do: func() error { _, _, err := proposal.Like(awaiting, w.mom, ctx, w.now); return err }, wantErr: proposal.ErrNotLiker},
		{name: "child likes", do: func() error { _, _, err := proposal.Like(awaiting, w.child, ctx, w.now); return err }, wantErr: proposal.ErrNotLiker},
		{name: "like after liked", do: func() error { _, _, err := proposal.Like(liked, w.dad, ctx, w.now); return err }, wantErr: proposal.ErrNotAwaitingLike},
		{name: "pass own proposal on self route", do: func() error { _, err := proposal.Pass(selfRoute, w.mom, pass, w.now); return err }, wantErr: proposal.ErrNotLiker},
		{name: "someone else withdraws", do: func() error { _, err := proposal.Withdraw(awaiting, w.dad, w.now); return err }, wantErr: proposal.ErrNotProposer},
		{name: "withdraw after liked", do: func() error { _, err := proposal.Withdraw(liked, w.mom, w.now); return err }, wantErr: proposal.ErrNotAwaitingLike},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.do(); !errors.Is(err, tt.wantErr) {
				t.Errorf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestPass(t *testing.T) {
	w := newWorld(t)
	ctx := w.ctx(w.twoKeepers, 0)
	p := w.submit(t, w.mom, proposal.NewChangeMonthlyBudget(nonNegative(t, 200000, money.JPY)), ctx)
	pass, err := like.NewPass([]like.ProposalPassReason{like.ProposalWalletCannotCover}, "")
	if err != nil {
		t.Fatal(err)
	}
	e, err := proposal.Pass(p, w.dad, pass, w.now)
	if err != nil {
		t.Fatal(err)
	}
	passed, ok := evolve(t, p, e).State().(proposal.PassedState)
	if !ok || passed.Passer() != w.dad {
		t.Errorf("State = %+v, want passed by dad", passed)
	}
}

func TestReplay(t *testing.T) {
	w := newWorld(t)
	ctx := w.ctx(w.twoKeepers, 0)
	submitted, err := proposal.Submit(w.mom, proposal.NewChangeThrillLine(nonNegative(t, 5000, money.JPY)), "", ctx, w.now)
	if err != nil {
		t.Fatal(err)
	}
	id, err := proposal.NewID("p1")
	if err != nil {
		t.Fatal(err)
	}
	p, err := proposal.Replay(id, []proposal.Event{submitted, proposal.NewLiked(w.now, w.dad, w.now)})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := p.State().(proposal.LikedState); !ok || p.Version() != 2 {
		t.Errorf("Replay = %T version %d, want LikedState version 2", p.State(), p.Version())
	}
	if _, err := proposal.Replay(id, nil); !errors.Is(err, proposal.ErrNoEvents) {
		t.Errorf("Replay(nil) error = %v, want %v", err, proposal.ErrNoEvents)
	}
	if _, err := proposal.Evolve(p, proposal.NewWithdrawn(w.now, w.mom)); !errors.Is(err, proposal.ErrInvalidTransition) {
		t.Errorf("Evolve(liked, withdrawn) error = %v, want %v", err, proposal.ErrInvalidTransition)
	}
}
