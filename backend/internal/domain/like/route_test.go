package like_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/kajiya-i/muda/backend/internal/domain/household"
	"github.com/kajiya-i/muda/backend/internal/domain/like"
)

func id(t *testing.T, v string) household.MemberID {
	t.Helper()
	m, err := household.NewMemberID(v)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// newHousehold は、keepers をおさいふ係、others をおさいふ係ではない家族とするおうちを返す。
func newHousehold(t *testing.T, keepers, others []household.MemberID) household.Household {
	t.Helper()
	h := household.New(keepers[0])
	for _, k := range keepers[1:] {
		var err error
		if h, err = h.AddMember(k); err != nil {
			t.Fatal(err)
		}
		if h, err = h.MakeKeeper(k); err != nil {
			t.Fatal(err)
		}
	}
	for _, o := range others {
		var err error
		if h, err = h.AddMember(o); err != nil {
			t.Fatal(err)
		}
	}
	return h
}

// describe は、ルートを文字列にする。ルートの種類で分岐するときの書き方の例であり、
// default を書かないことで、ルートの種類を増やしたときに gochecksumtype が指摘する。
func describe(r like.Route) string {
	switch r := r.(type) {
	case like.ByKeepers:
		ids := make([]string, 0, len(r.Likers()))
		for _, m := range r.Likers() {
			ids = append(ids, m.String())
		}
		slices.Sort(ids)
		return "keepers:" + strings.Join(ids, ",")
	case like.SelfLike:
		return "self:" + r.Member().String()
	}
	return ""
}

func TestDecideRoute(t *testing.T) {
	mom, dad, child := id(t, "mom"), id(t, "dad"), id(t, "child")

	twoKeepers := newHousehold(t, []household.MemberID{mom, dad}, []household.MemberID{child})
	oneKeeper := newHousehold(t, []household.MemberID{mom}, []household.MemberID{child})
	alone := newHousehold(t, []household.MemberID{mom}, nil)

	tests := []struct {
		name     string
		h        household.Household
		proposer household.MemberID
		want     string
	}{
		{name: "child with two keepers", h: twoKeepers, proposer: child, want: "keepers:dad,mom"},
		{name: "keeper with another keeper", h: twoKeepers, proposer: mom, want: "keepers:dad"},
		{name: "child with one keeper", h: oneKeeper, proposer: child, want: "keepers:mom"},
		{name: "only keeper with a child", h: oneKeeper, proposer: mom, want: "self:mom"},
		{name: "only keeper alone", h: alone, proposer: mom, want: "self:mom"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := like.DecideRoute(tt.h, tt.proposer)
			if err != nil {
				t.Fatal(err)
			}
			if describe(got) != tt.want {
				t.Errorf("DecideRoute = %s, want %s", describe(got), tt.want)
			}
		})
	}
}

func TestDecideRouteErrors(t *testing.T) {
	mom, child := id(t, "mom"), id(t, "child")
	h := newHousehold(t, []household.MemberID{mom}, []household.MemberID{child})
	left, err := h.RemoveMember(child)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := like.DecideRoute(h, id(t, "unknown")); !errors.Is(err, household.ErrMemberNotFound) {
		t.Errorf("unknown proposer error = %v, want %v", err, household.ErrMemberNotFound)
	}
	if _, err := like.DecideRoute(left, child); !errors.Is(err, household.ErrMemberLeft) {
		t.Errorf("left proposer error = %v, want %v", err, household.ErrMemberLeft)
	}
}

func TestByKeepersCanLike(t *testing.T) {
	mom, dad, child := id(t, "mom"), id(t, "dad"), id(t, "child")
	h := newHousehold(t, []household.MemberID{mom, dad}, []household.MemberID{child})

	route, err := like.DecideRoute(h, mom)
	if err != nil {
		t.Fatal(err)
	}
	byKeepers, ok := route.(like.ByKeepers)
	if !ok {
		t.Fatalf("route = %T, want like.ByKeepers", route)
	}
	if !byKeepers.CanLike(dad) {
		t.Error("CanLike(dad) = false, want true")
	}
	for _, m := range []household.MemberID{mom, child} {
		if byKeepers.CanLike(m) {
			t.Errorf("CanLike(%s) = true, want false", m)
		}
	}
}
