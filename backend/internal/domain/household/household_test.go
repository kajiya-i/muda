package household_test

import (
	"errors"
	"testing"

	"github.com/kajiya-i/muda/backend/internal/domain/household"
)

func id(t *testing.T, v string) household.MemberID {
	t.Helper()
	m, err := household.NewMemberID(v)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestNewMemberID(t *testing.T) {
	if _, err := household.NewMemberID(""); !errors.Is(err, household.ErrEmptyMemberID) {
		t.Errorf("NewMemberID(\"\") error = %v, want %v", err, household.ErrEmptyMemberID)
	}
}

func TestNew(t *testing.T) {
	founder := id(t, "founder")
	h := household.New(founder)

	keepers := h.ActiveKeepers()
	if len(keepers) != 1 || keepers[0].ID() != founder {
		t.Errorf("ActiveKeepers = %v, want only %v", keepers, founder)
	}
}

func TestAddMember(t *testing.T) {
	founder, child := id(t, "founder"), id(t, "child")
	h, err := household.New(founder).AddMember(child)
	if err != nil {
		t.Fatal(err)
	}

	m, ok := h.Member(child)
	if !ok {
		t.Fatalf("Member(%v) not found", child)
	}
	if m.Role() != household.RoleNonKeeper || m.Status() != household.StatusActive {
		t.Errorf("added member = role %v, status %v, want non-keeper and active", m.Role(), m.Status())
	}

	if _, err := h.AddMember(child); !errors.Is(err, household.ErrDuplicateMember) {
		t.Errorf("AddMember twice error = %v, want %v", err, household.ErrDuplicateMember)
	}
}

func TestOperationsDoNotChangeOriginal(t *testing.T) {
	founder, child := id(t, "founder"), id(t, "child")
	original, err := household.New(founder).AddMember(child)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := original.MakeKeeper(child); err != nil {
		t.Fatal(err)
	}

	m, _ := original.Member(child)
	if m.Role() != household.RoleNonKeeper {
		t.Errorf("original was changed: role = %v, want non-keeper", m.Role())
	}
}

func TestKeeperChanges(t *testing.T) {
	founder, partner, child := id(t, "founder"), id(t, "partner"), id(t, "child")
	h, err := household.New(founder).AddMember(partner)
	if err != nil {
		t.Fatal(err)
	}
	h, err = h.AddMember(child)
	if err != nil {
		t.Fatal(err)
	}
	twoKeepers, err := h.MakeKeeper(partner)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		do      func() (household.Household, error)
		wantErr error
	}{
		{name: "make keeper twice", do: func() (household.Household, error) { return twoKeepers.MakeKeeper(partner) }, wantErr: household.ErrAlreadyKeeper},
		{name: "remove non-keeper", do: func() (household.Household, error) { return twoKeepers.RemoveKeeper(child) }, wantErr: household.ErrNotKeeper},
		{name: "remove one of two keepers", do: func() (household.Household, error) { return twoKeepers.RemoveKeeper(partner) }},
		{name: "remove last keeper", do: func() (household.Household, error) { return h.RemoveKeeper(founder) }, wantErr: household.ErrLastKeeper},
		{name: "unknown member", do: func() (household.Household, error) { return h.MakeKeeper(id(t, "unknown")) }, wantErr: household.ErrMemberNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tt.do(); !errors.Is(err, tt.wantErr) {
				t.Errorf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestRemoveMember(t *testing.T) {
	founder, child := id(t, "founder"), id(t, "child")
	h, err := household.New(founder).AddMember(child)
	if err != nil {
		t.Fatal(err)
	}

	left, err := h.RemoveMember(child)
	if err != nil {
		t.Fatal(err)
	}
	m, ok := left.Member(child)
	if !ok || m.Status() != household.StatusLeft {
		t.Errorf("removed member = %v (found %v), want status left", m, ok)
	}

	if _, err := left.MakeKeeper(child); !errors.Is(err, household.ErrMemberLeft) {
		t.Errorf("MakeKeeper(left member) error = %v, want %v", err, household.ErrMemberLeft)
	}
	if _, err := h.RemoveMember(founder); !errors.Is(err, household.ErrLastKeeper) {
		t.Errorf("RemoveMember(last keeper) error = %v, want %v", err, household.ErrLastKeeper)
	}
}
