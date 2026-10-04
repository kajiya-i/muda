package consultation_test

import (
	"errors"
	"testing"
	"time"

	"github.com/kajiya-i/muda/backend/internal/domain/consultation"
	"github.com/kajiya-i/muda/backend/internal/domain/household"
	"github.com/kajiya-i/muda/backend/internal/domain/like"
	"github.com/kajiya-i/muda/backend/internal/domain/money"
	"github.com/kajiya-i/muda/backend/internal/domain/purpose"
)

// world は、Decide のテストで使うおうちと家族。
type world struct {
	mom, dad, child household.MemberID
	twoKeepers      household.Household // mom と dad がおさいふ係、child はおさいふ係ではない
	oneKeeper       household.Household // mom だけがおさいふ係、child はおさいふ係ではない
	tokyo           *time.Location
	now             time.Time
	thrillLine      money.NonNegativeMoney
}

func newWorld(t *testing.T) world {
	t.Helper()
	f := newFixture(t)
	build := func(keepers []household.MemberID, others []household.MemberID) household.Household {
		h := household.New(keepers[0])
		var err error
		for _, k := range keepers[1:] {
			if h, err = h.AddMember(k); err != nil {
				t.Fatal(err)
			}
			if h, err = h.MakeKeeper(k); err != nil {
				t.Fatal(err)
			}
		}
		for _, o := range others {
			if h, err = h.AddMember(o); err != nil {
				t.Fatal(err)
			}
		}
		return h
	}
	line, err := money.NewNonNegative(10000, money.JPY)
	if err != nil {
		t.Fatal(err)
	}
	return world{
		mom: f.mom, dad: f.dad, child: f.child,
		twoKeepers: build([]household.MemberID{f.mom, f.dad}, []household.MemberID{f.child}),
		oneKeeper:  build([]household.MemberID{f.mom}, []household.MemberID{f.child}),
		tokyo:      f.tokyo,
		now:        f.at,
		thrillLine: line,
	}
}

func yen(t *testing.T, minor int64) money.PositiveMoney {
	t.Helper()
	p, err := money.NewPositive(minor, money.JPY)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// submit は、相談を出して始めた相談を返す。相談 ID は c1 にする。
func (w world) submit(t *testing.T, h household.Household, requester household.MemberID, amount int64) consultation.Consultation {
	t.Helper()
	return w.submitWithID(t, "c1", h, requester, amount)
}

// submitWithID は、指定した相談 ID で、相談を出して始めた相談を返す。
func (w world) submitWithID(t *testing.T, id string, h household.Household, requester household.MemberID, amount int64) consultation.Consultation {
	t.Helper()
	in := consultation.SubmitInput{Item: "スニーカー", Amount: yen(t, amount), Purpose: purpose.BuiltinClothing}
	e, err := consultation.Submit(requester, in, h, w.thrillLine, w.now)
	if err != nil {
		t.Fatal(err)
	}
	cid, err := consultation.NewID(id)
	if err != nil {
		t.Fatal(err)
	}
	return consultation.Start(cid, e)
}

// like は、いいねを決めて、そのイベントを適用した相談と、決まったイベントを返す。
func (w world) like(t *testing.T, c consultation.Consultation, liker household.MemberID, remaining int64) (consultation.Consultation, []consultation.Event) {
	t.Helper()
	events, err := consultation.Like(c, liker, money.New(remaining, money.JPY), w.tokyo, w.now)
	if err != nil {
		t.Fatal(err)
	}
	return evolveAll(t, c, events...), events
}

func TestSubmit(t *testing.T) {
	w := newWorld(t)
	tests := []struct {
		name       string
		amount     int64
		wantOver   bool
		wantLikers int
	}{
		{name: "below thrill line", amount: 9999, wantOver: false, wantLikers: 2},
		{name: "equal to thrill line", amount: 10000, wantOver: false, wantLikers: 2},
		{name: "over thrill line", amount: 10001, wantOver: true, wantLikers: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := consultation.SubmitInput{Item: "スニーカー", Amount: yen(t, tt.amount), Purpose: purpose.BuiltinClothing}
			e, err := consultation.Submit(w.child, in, w.twoKeepers, w.thrillLine, w.now)
			if err != nil {
				t.Fatal(err)
			}
			if e.OverThrillLine() != tt.wantOver {
				t.Errorf("OverThrillLine = %v, want %v", e.OverThrillLine(), tt.wantOver)
			}
			route, ok := e.Route().(like.ByKeepers)
			if !ok || len(route.Likers()) != tt.wantLikers {
				t.Errorf("Route = %#v, want ByKeepers with %d likers", e.Route(), tt.wantLikers)
			}
			if e.By() != w.child || !e.OccurredAt().Equal(w.now) {
				t.Errorf("By = %v, OccurredAt = %v", e.By(), e.OccurredAt())
			}
		})
	}
}

func TestSubmitErrors(t *testing.T) {
	w := newWorld(t)
	usdLine, err := money.NewNonNegative(100, money.USD)
	if err != nil {
		t.Fatal(err)
	}
	left, err := w.twoKeepers.RemoveMember(w.child)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		in      consultation.SubmitInput
		h       household.Household
		line    money.NonNegativeMoney
		wantErr error
	}{
		{name: "empty item", in: consultation.SubmitInput{Amount: yen(t, 1000), Purpose: purpose.BuiltinOther}, h: w.twoKeepers, line: w.thrillLine, wantErr: consultation.ErrEmptyItem},
		{name: "no purpose", in: consultation.SubmitInput{Item: "本", Amount: yen(t, 1000)}, h: w.twoKeepers, line: w.thrillLine, wantErr: consultation.ErrNoPurpose},
		{name: "currency mismatch", in: consultation.SubmitInput{Item: "本", Amount: yen(t, 1000), Purpose: purpose.BuiltinEducation}, h: w.twoKeepers, line: usdLine, wantErr: money.ErrCurrencyMismatch},
		{name: "requester has left", in: consultation.SubmitInput{Item: "本", Amount: yen(t, 1000), Purpose: purpose.BuiltinEducation}, h: left, line: w.thrillLine, wantErr: household.ErrMemberLeft},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := consultation.Submit(w.child, tt.in, tt.h, tt.line, w.now); !errors.Is(err, tt.wantErr) {
				t.Errorf("Submit error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestLikeNeedsOneLike(t *testing.T) {
	w := newWorld(t)
	c := w.submit(t, w.twoKeepers, w.child, 5000)

	c, events := w.like(t, c, w.mom, 50000)
	if len(events) != 2 {
		t.Fatalf("events = %d, want 2 (like given and liked)", len(events))
	}
	if _, ok := events[1].(consultation.Liked); !ok {
		t.Errorf("events[1] = %T, want Liked", events[1])
	}
	liked, ok := c.State().(consultation.LikedState)
	if !ok {
		t.Fatalf("State = %T, want LikedState", c.State())
	}
	if want := time.Date(2026, 11, 8, 0, 0, 0, 0, w.tokyo); !liked.Deadlines().Report().Equal(want) {
		t.Errorf("Report deadline = %v, want %v", liked.Deadlines().Report(), want)
	}
}

func TestLikeNeedsTwoLikes(t *testing.T) {
	tests := []struct {
		name      string
		amount    int64
		remaining int64
		wantOver  like.Reasons
	}{
		{name: "over thrill line", amount: 12000, remaining: 50000, wantOver: like.NewReasons(true, false)},
		{name: "over remaining budget", amount: 5000, remaining: 4999, wantOver: like.NewReasons(false, true)},
		{name: "remaining budget is negative", amount: 1, remaining: -8000, wantOver: like.NewReasons(false, true)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := newWorld(t)
			c := w.submit(t, w.twoKeepers, w.child, tt.amount)

			c, events := w.like(t, c, w.mom, tt.remaining)
			if len(events) != 1 {
				t.Fatalf("after first like: events = %d, want 1", len(events))
			}
			a, ok := c.State().(consultation.AwaitingLikes)
			if !ok {
				t.Fatalf("after first like: State = %T, want AwaitingLikes", c.State())
			}
			first, ok := a.FirstLike()
			if !ok || first.Requirement().Count() != 2 || first.Requirement().Reasons() != tt.wantOver {
				t.Errorf("FirstLike = %+v, want 2 likes with reasons %+v", first, tt.wantOver)
			}

			// 2 つ目のいいねでは、やりくりの残りを計算し直さない。
			c, events = w.like(t, c, w.dad, 1_000_000)
			if len(events) != 2 {
				t.Fatalf("after second like: events = %d, want 2", len(events))
			}
			if _, ok := c.State().(consultation.LikedState); !ok {
				t.Errorf("after second like: State = %T, want LikedState", c.State())
			}
		})
	}
}

func TestLikeWithOnlyOneLiker(t *testing.T) {
	w := newWorld(t)
	// dad がおさいふ係の 2 人のおうちで、mom が相談すると、いいねする人は dad だけになる。
	c := w.submit(t, w.twoKeepers, w.mom, 12000)

	c, events := w.like(t, c, w.dad, 0)
	if len(events) != 2 {
		t.Fatalf("events = %d, want 2", len(events))
	}
	if _, ok := c.State().(consultation.LikedState); !ok {
		t.Errorf("State = %T, want LikedState", c.State())
	}
}

func TestSelfLike(t *testing.T) {
	w := newWorld(t)
	c := w.submit(t, w.oneKeeper, w.mom, 12000)

	c, events := w.like(t, c, w.mom, 5000)
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	self, ok := events[0].(consultation.SelfLiked)
	if !ok {
		t.Fatalf("events[0] = %T, want SelfLiked", events[0])
	}
	if want := like.NewReasons(true, true); self.Requirement().Reasons() != want {
		t.Errorf("Reasons = %+v, want %+v", self.Requirement().Reasons(), want)
	}
	if self.RemainingBudget() != money.New(5000, money.JPY) {
		t.Errorf("RemainingBudget = %v, want 5000 JPY", self.RemainingBudget())
	}
	if _, ok := c.State().(consultation.LikedState); !ok {
		t.Errorf("State = %T, want LikedState", c.State())
	}
}

func TestLikeErrors(t *testing.T) {
	w := newWorld(t)
	byKeepers := w.submit(t, w.twoKeepers, w.child, 12000)
	afterFirst, _ := w.like(t, byKeepers, w.mom, 50000)
	selfRoute := w.submit(t, w.oneKeeper, w.mom, 1000)
	liked, _ := w.like(t, w.submit(t, w.twoKeepers, w.child, 1000), w.mom, 50000)

	tests := []struct {
		name    string
		c       consultation.Consultation
		liker   household.MemberID
		wantErr error
	}{
		{name: "requester likes own consultation", c: byKeepers, liker: w.child, wantErr: consultation.ErrNotLiker},
		{name: "same keeper likes twice", c: afterFirst, liker: w.mom, wantErr: consultation.ErrAlreadyLiked},
		{name: "someone else self-likes", c: selfRoute, liker: w.child, wantErr: consultation.ErrNotLiker},
		{name: "like after liked", c: liked, liker: w.dad, wantErr: consultation.ErrNotAwaitingLikes},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := consultation.Like(tt.c, tt.liker, money.New(50000, money.JPY), w.tokyo, w.now); !errors.Is(err, tt.wantErr) {
				t.Errorf("Like error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestPass(t *testing.T) {
	w := newWorld(t)
	pass, err := like.NewPass([]like.ConsultationPassReason{like.ConsultationCheaperAlternative}, "")
	if err != nil {
		t.Fatal(err)
	}

	// 1 人目がいいねし、2 人目が見送ると、見送りになる。
	c := w.submit(t, w.twoKeepers, w.child, 12000)
	c, _ = w.like(t, c, w.mom, 50000)
	e, err := consultation.Pass(c, w.dad, pass, w.now)
	if err != nil {
		t.Fatal(err)
	}
	c = evolveAll(t, c, e)
	passed, ok := c.State().(consultation.PassedState)
	if !ok || passed.Passer() != w.dad {
		t.Errorf("State = %+v, want passed by dad", c.State())
	}
}

func TestPassErrors(t *testing.T) {
	w := newWorld(t)
	pass, err := like.NewPass([]like.ConsultationPassReason{like.ConsultationTooExpensive}, "")
	if err != nil {
		t.Fatal(err)
	}
	byKeepers := w.submit(t, w.twoKeepers, w.child, 12000)
	afterFirst, _ := w.like(t, byKeepers, w.mom, 50000)
	selfRoute := w.submit(t, w.oneKeeper, w.mom, 1000)

	tests := []struct {
		name    string
		c       consultation.Consultation
		passer  household.MemberID
		wantErr error
	}{
		{name: "requester passes own consultation", c: byKeepers, passer: w.child, wantErr: consultation.ErrNotLiker},
		{name: "keeper who liked passes", c: afterFirst, passer: w.mom, wantErr: consultation.ErrAlreadyLiked},
		{name: "self like route", c: selfRoute, passer: w.mom, wantErr: consultation.ErrNotLiker},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := consultation.Pass(tt.c, tt.passer, pass, w.now); !errors.Is(err, tt.wantErr) {
				t.Errorf("Pass error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestWithdraw(t *testing.T) {
	w := newWorld(t)
	awaiting := w.submit(t, w.twoKeepers, w.child, 1000)
	liked, _ := w.like(t, awaiting, w.mom, 50000)
	pass, err := like.NewPass([]like.ConsultationPassReason{like.ConsultationTooExpensive}, "")
	if err != nil {
		t.Fatal(err)
	}
	passEvent, err := consultation.Pass(w.submit(t, w.twoKeepers, w.child, 1000), w.mom, pass, w.now)
	if err != nil {
		t.Fatal(err)
	}
	passed := evolveAll(t, w.submit(t, w.twoKeepers, w.child, 1000), passEvent)

	tests := []struct {
		name    string
		c       consultation.Consultation
		by      household.MemberID
		reason  consultation.WithdrawalReason
		wantErr error
	}{
		{name: "requester withdraws awaiting", c: awaiting, by: w.child, reason: consultation.WithdrawnByRequester},
		{name: "requester withdraws liked", c: liked, by: w.child, reason: consultation.WithdrawnByRequester},
		{name: "keeper withdraws because requester left", c: liked, by: w.mom, reason: consultation.WithdrawnBecauseRequesterLeft},
		{name: "someone else withdraws", c: awaiting, by: w.mom, reason: consultation.WithdrawnByRequester, wantErr: consultation.ErrNotRequester},
		{name: "withdraw passed", c: passed, by: w.child, reason: consultation.WithdrawnByRequester, wantErr: consultation.ErrCannotWithdraw},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, err := consultation.Withdraw(tt.c, tt.by, tt.reason, w.now)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Withdraw error = %v, want %v", err, tt.wantErr)
			}
			if err == nil {
				// Decide が返したイベントは、必ず Evolve で適用できる。
				evolveAll(t, tt.c, e)
			}
		})
	}
}
