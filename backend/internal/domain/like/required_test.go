package like_test

import (
	"errors"
	"testing"

	"github.com/kajiya-i/muda/backend/internal/domain/household"
	"github.com/kajiya-i/muda/backend/internal/domain/like"
	"github.com/kajiya-i/muda/backend/internal/domain/money"
)

func positive(t *testing.T, minor int64) money.PositiveMoney {
	t.Helper()
	p, err := money.NewPositive(minor, money.JPY)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func nonNegative(t *testing.T, minor int64) money.NonNegativeMoney {
	t.Helper()
	n, err := money.NewNonNegative(minor, money.JPY)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func TestExceedsThrillLine(t *testing.T) {
	tests := []struct {
		name   string
		amount int64
		line   int64
		want   bool
	}{
		{name: "below", amount: 9999, line: 10000, want: false},
		{name: "equal", amount: 10000, line: 10000, want: false},
		{name: "above", amount: 10001, line: 10000, want: true},
		{name: "zero line", amount: 1, line: 0, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := like.ExceedsThrillLine(positive(t, tt.amount), nonNegative(t, tt.line))
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("ExceedsThrillLine(%d, %d) = %v, want %v", tt.amount, tt.line, got, tt.want)
			}
		})
	}

	usdLine, err := money.NewNonNegative(100, money.USD)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := like.ExceedsThrillLine(positive(t, 1), usdLine); !errors.Is(err, money.ErrCurrencyMismatch) {
		t.Errorf("currency mismatch error = %v, want %v", err, money.ErrCurrencyMismatch)
	}
}

func TestExceedsRemainingBudget(t *testing.T) {
	tests := []struct {
		name      string
		amount    int64
		remaining int64
		want      bool
	}{
		{name: "below", amount: 9999, remaining: 10000, want: false},
		{name: "equal", amount: 10000, remaining: 10000, want: false},
		{name: "above", amount: 10001, remaining: 10000, want: true},
		{name: "zero remaining", amount: 1, remaining: 0, want: true},
		{name: "negative remaining", amount: 1, remaining: -8000, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := like.ExceedsRemainingBudget(positive(t, tt.amount), money.New(tt.remaining, money.JPY))
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("ExceedsRemainingBudget(%d, %d) = %v, want %v", tt.amount, tt.remaining, got, tt.want)
			}
		})
	}
}

// requirement は、必要ないいねを比べやすい形にする。
func requirement(r like.Requirement) (self bool, count int, reasons like.Reasons) {
	switch r := r.(type) {
	case like.LikesFromKeepers:
		return false, r.Count(), r.Reasons()
	case like.LikeFromSelf:
		return true, 0, r.Reasons()
	}
	return false, 0, like.Reasons{}
}

func TestConsultationRequirement(t *testing.T) {
	mom, dad, child := id(t, "mom"), id(t, "dad"), id(t, "child")
	twoKeepers := newHousehold(t, []household.MemberID{mom, dad}, []household.MemberID{child})
	oneKeeper := newHousehold(t, []household.MemberID{mom}, []household.MemberID{child})

	route := func(h household.Household, proposer household.MemberID) like.Route {
		r, err := like.DecideRoute(h, proposer)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	childWithTwo := route(twoKeepers, child)
	keeperWithOther := route(twoKeepers, mom)
	childWithOne := route(oneKeeper, child)
	onlyKeeper := route(oneKeeper, mom)

	none := like.NewReasons(false, false)
	thrill := like.NewReasons(true, false)
	budget := like.NewReasons(false, true)
	both := like.NewReasons(true, true)

	tests := []struct {
		name      string
		route     like.Route
		reasons   like.Reasons
		wantSelf  bool
		wantCount int
	}{
		{name: "two likers, no reason", route: childWithTwo, reasons: none, wantCount: 1},
		{name: "two likers, over thrill line", route: childWithTwo, reasons: thrill, wantCount: 2},
		{name: "two likers, over remaining budget", route: childWithTwo, reasons: budget, wantCount: 2},
		{name: "two likers, both reasons", route: childWithTwo, reasons: both, wantCount: 2},
		{name: "one liker (other keeper), both reasons", route: keeperWithOther, reasons: both, wantCount: 1},
		{name: "one liker (only keeper), over thrill line", route: childWithOne, reasons: thrill, wantCount: 1},
		{name: "self like, both reasons", route: onlyKeeper, reasons: both, wantSelf: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			self, count, reasons := requirement(like.ConsultationRequirement(tt.route, tt.reasons))
			if self != tt.wantSelf || count != tt.wantCount {
				t.Errorf("got self=%v count=%d, want self=%v count=%d", self, count, tt.wantSelf, tt.wantCount)
			}
			if reasons != tt.reasons {
				t.Errorf("reasons = %+v, want %+v (reasons are kept as a record)", reasons, tt.reasons)
			}
		})
	}
}

func TestProposalRequirement(t *testing.T) {
	mom, dad := id(t, "mom"), id(t, "dad")
	twoKeepers := newHousehold(t, []household.MemberID{mom, dad}, nil)
	alone := newHousehold(t, []household.MemberID{mom}, nil)

	byKeepers, err := like.DecideRoute(twoKeepers, mom)
	if err != nil {
		t.Fatal(err)
	}
	if self, count, _ := requirement(like.ProposalRequirement(byKeepers)); self || count != 1 {
		t.Errorf("by keepers: self=%v count=%d, want self=false count=1", self, count)
	}

	selfRoute, err := like.DecideRoute(alone, mom)
	if err != nil {
		t.Fatal(err)
	}
	if self, _, _ := requirement(like.ProposalRequirement(selfRoute)); !self {
		t.Error("self route: self=false, want true")
	}
}
