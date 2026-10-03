package consultation_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/kajiya-i/muda/backend/internal/domain/consultation"
	"github.com/kajiya-i/muda/backend/internal/domain/household"
	"github.com/kajiya-i/muda/backend/internal/domain/ledger"
	"github.com/kajiya-i/muda/backend/internal/domain/like"
	"github.com/kajiya-i/muda/backend/internal/domain/money"
	"github.com/kajiya-i/muda/backend/internal/domain/purpose"
)

// fixture は、テストで使う家族、おうち、時刻、イベントをまとめたもの。
type fixture struct {
	mom, dad, child household.MemberID
	tokyo           *time.Location
	at              time.Time
	deadlines       consultation.Deadlines
	submitted       consultation.Submitted
	first           consultation.FirstLike
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	mustID := func(v string) household.MemberID {
		id, err := household.NewMemberID(v)
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	mom, dad, child := mustID("mom"), mustID("dad"), mustID("child")

	h := household.New(mom)
	var err error
	for _, step := range []func() (household.Household, error){
		func() (household.Household, error) { return h.AddMember(dad) },
		func() (household.Household, error) { return h.MakeKeeper(dad) },
		func() (household.Household, error) { return h.AddMember(child) },
	} {
		if h, err = step(); err != nil {
			t.Fatal(err)
		}
	}
	route, err := like.DecideRoute(h, child)
	if err != nil {
		t.Fatal(err)
	}
	amount, err := money.NewPositive(12000, money.JPY)
	if err != nil {
		t.Fatal(err)
	}

	// どきどきラインを超えているので、いいねが 2 つ必要になる。
	requirement, ok := like.ConsultationRequirement(route, like.NewReasons(true, false)).(like.LikesFromKeepers)
	if !ok {
		t.Fatal("requirement is not LikesFromKeepers")
	}

	tokyo := location(t, "Asia/Tokyo")
	at := time.Date(2026, 10, 15, 12, 0, 0, 0, tokyo)
	return fixture{
		mom: mom, dad: dad, child: child, tokyo: tokyo, at: at,
		deadlines: consultation.NewDeadlines(at, tokyo),
		submitted: consultation.NewSubmitted(at, child, "スニーカー", amount, purpose.BuiltinClothing, "", route, true),
		first:     consultation.NewFirstLike(requirement, money.New(50000, money.JPY)),
	}
}

func (f fixture) id(t *testing.T) consultation.ID {
	t.Helper()
	id, err := consultation.NewID("c1")
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func evolveAll(t *testing.T, c consultation.Consultation, events ...consultation.Event) consultation.Consultation {
	t.Helper()
	for _, e := range events {
		var err error
		if c, err = consultation.Evolve(c, e); err != nil {
			t.Fatal(err)
		}
	}
	return c
}

func TestStart(t *testing.T) {
	f := newFixture(t)
	c := consultation.Start(f.id(t), f.submitted)

	if c.Version() != 1 {
		t.Errorf("Version = %d, want 1", c.Version())
	}
	a, ok := c.State().(consultation.AwaitingLikes)
	if !ok {
		t.Fatalf("State = %T, want AwaitingLikes", c.State())
	}
	if a.Details().Requester() != f.child || a.Details().Item() != "スニーカー" || !a.Details().OverThrillLine() {
		t.Errorf("Details = %+v", a.Details())
	}
	if _, ok := a.FirstLike(); ok {
		t.Error("FirstLike exists before any like")
	}
}

func TestEvolveWalletPaymentFlow(t *testing.T) {
	f := newFixture(t)
	amount, err := money.NewPositive(11800, money.JPY)
	if err != nil {
		t.Fatal(err)
	}
	c := consultation.Start(f.id(t), f.submitted)
	c = evolveAll(t, c, consultation.NewFirstLikeGiven(f.at, f.mom, f.first))

	a, ok := c.State().(consultation.AwaitingLikes)
	if !ok {
		t.Fatalf("after first like: State = %T, want AwaitingLikes", c.State())
	}
	if got, ok := a.FirstLike(); !ok || got.RemainingBudget() != money.New(50000, money.JPY) || got.Requirement().Count() != 2 {
		t.Errorf("FirstLike = %+v (found %v)", got, ok)
	}

	c = evolveAll(t, c,
		consultation.NewLikeGiven(f.at, f.dad),
		consultation.NewLiked(f.at, f.dad, f.deadlines),
	)
	liked, ok := c.State().(consultation.LikedState)
	if !ok {
		t.Fatalf("after liked: State = %T, want LikedState", c.State())
	}
	if !slices.Equal(liked.Likes(), []household.MemberID{f.mom, f.dad}) {
		t.Errorf("Likes = %v, want [mom dad]", liked.Likes())
	}

	c = evolveAll(t, c, consultation.NewPurchaseReported(f.at, f.child, amount, consultation.PaidFromWallet))
	purchased, ok := c.State().(consultation.PurchasedState)
	if !ok {
		t.Fatalf("after purchase: State = %T, want PurchasedState", c.State())
	}
	if purchased.Amount() != amount || purchased.Method() != consultation.PaidFromWallet {
		t.Errorf("Purchased = %v %v", purchased.Amount(), purchased.Method())
	}
	if c.Version() != 5 {
		t.Errorf("Version = %d, want 5", c.Version())
	}
}

func TestEvolveAdvanceAndRepayment(t *testing.T) {
	f := newFixture(t)
	repayment, err := ledger.NewRepaymentID("r1")
	if err != nil {
		t.Fatal(err)
	}
	c := evolveAll(t, consultation.Start(f.id(t), f.submitted),
		consultation.NewSelfLiked(f.at, f.mom, f.deadlines),
		consultation.NewPurchaseReported(f.at, f.child, f.submitted.Amount(), consultation.PaidByAdvance),
		consultation.NewAdvanceRepaid(f.at, f.mom, repayment),
	)
	repaid, ok := c.State().(consultation.RepaidState)
	if !ok {
		t.Fatalf("State = %T, want RepaidState", c.State())
	}
	if repaid.Repayment() != repayment {
		t.Errorf("Repayment = %v, want %v", repaid.Repayment(), repayment)
	}
}

func TestEvolveInvalidTransitions(t *testing.T) {
	f := newFixture(t)
	pass, err := like.NewPass([]like.ConsultationPassReason{like.ConsultationTooExpensive}, "")
	if err != nil {
		t.Fatal(err)
	}
	repayment, err := ledger.NewRepaymentID("r1")
	if err != nil {
		t.Fatal(err)
	}

	start := consultation.Start(f.id(t), f.submitted)
	liked := evolveAll(t, start, consultation.NewLiked(f.at, f.mom, f.deadlines))
	passed := evolveAll(t, start, consultation.NewPassed(f.at, f.mom, pass))
	paidFromWallet := evolveAll(t, liked,
		consultation.NewPurchaseReported(f.at, f.child, f.submitted.Amount(), consultation.PaidFromWallet))

	tests := []struct {
		name  string
		c     consultation.Consultation
		event consultation.Event
	}{
		{name: "submit twice", c: start, event: f.submitted},
		{name: "purchase before liked", c: start, event: consultation.NewPurchaseReported(f.at, f.child, f.submitted.Amount(), consultation.PaidFromWallet)},
		{name: "like after liked", c: liked, event: consultation.NewLikeGiven(f.at, f.dad)},
		{name: "purchase after passed", c: passed, event: consultation.NewPurchaseReported(f.at, f.child, f.submitted.Amount(), consultation.PaidFromWallet)},
		{name: "withdraw after passed", c: passed, event: consultation.NewWithdrawn(f.at, f.child, consultation.WithdrawnByRequester)},
		{name: "repay a wallet payment", c: paidFromWallet, event: consultation.NewAdvanceRepaid(f.at, f.mom, repayment)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := consultation.Evolve(tt.c, tt.event); !errors.Is(err, consultation.ErrInvalidTransition) {
				t.Errorf("Evolve error = %v, want %v", err, consultation.ErrInvalidTransition)
			}
		})
	}
}

func TestEvolveDoesNotChangeOriginal(t *testing.T) {
	f := newFixture(t)
	original := evolveAll(t, consultation.Start(f.id(t), f.submitted), consultation.NewFirstLikeGiven(f.at, f.mom, f.first))

	if _, err := consultation.Evolve(original, consultation.NewLikeGiven(f.at, f.dad)); err != nil {
		t.Fatal(err)
	}
	a, _ := original.State().(consultation.AwaitingLikes)
	if len(a.Likes()) != 1 || original.Version() != 2 {
		t.Errorf("original was changed: likes = %v, version = %d", a.Likes(), original.Version())
	}
}

func TestWithdrawFromLiked(t *testing.T) {
	f := newFixture(t)
	c := evolveAll(t, consultation.Start(f.id(t), f.submitted),
		consultation.NewLiked(f.at, f.mom, f.deadlines),
		consultation.NewWithdrawn(f.at, f.child, consultation.WithdrawnBecauseRequesterLeft),
	)
	w, ok := c.State().(consultation.WithdrawnState)
	if !ok || w.Reason() != consultation.WithdrawnBecauseRequesterLeft {
		t.Errorf("State = %+v, want withdrawn because requester left", c.State())
	}
}

func TestReplay(t *testing.T) {
	f := newFixture(t)
	events := []consultation.Event{
		f.submitted,
		consultation.NewLiked(f.at, f.mom, f.deadlines),
		consultation.NewPurchaseReported(f.at, f.child, f.submitted.Amount(), consultation.PaidFromWallet),
	}
	c, err := consultation.Replay(f.id(t), events)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := c.State().(consultation.PurchasedState); !ok || c.Version() != 3 {
		t.Errorf("Replay = %T version %d, want PurchasedState version 3", c.State(), c.Version())
	}

	if _, err := consultation.Replay(f.id(t), nil); !errors.Is(err, consultation.ErrNoEvents) {
		t.Errorf("Replay(nil) error = %v, want %v", err, consultation.ErrNoEvents)
	}
	if _, err := consultation.Replay(f.id(t), events[1:]); !errors.Is(err, consultation.ErrInvalidTransition) {
		t.Errorf("Replay(without submitted) error = %v, want %v", err, consultation.ErrInvalidTransition)
	}
}

func TestIsExpired(t *testing.T) {
	f := newFixture(t)
	start := consultation.Start(f.id(t), f.submitted)
	liked := evolveAll(t, start, consultation.NewLiked(f.at, f.mom, f.deadlines))
	afterReportDeadline := f.deadlines.Report()

	if consultation.IsExpired(start, afterReportDeadline) {
		t.Error("awaiting likes is expired, want not expired")
	}
	if consultation.IsExpired(liked, afterReportDeadline.Add(-time.Second)) {
		t.Error("liked is expired before the report deadline")
	}
	if !consultation.IsExpired(liked, afterReportDeadline) {
		t.Error("liked is not expired at the report deadline")
	}
}
