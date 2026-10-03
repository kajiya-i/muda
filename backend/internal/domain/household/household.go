package household

import (
	"errors"
	"fmt"
	"slices"
)

var (
	// ErrMemberNotFound は、おうちにいない家族を指定したときのエラー。
	ErrMemberNotFound = errors.New("member not found")
	// ErrDuplicateMember は、すでにおうちにいる家族を加えようとしたときのエラー。
	ErrDuplicateMember = errors.New("member already exists")
	// ErrMemberLeft は、おうちから外れた家族を操作しようとしたときのエラー。
	ErrMemberLeft = errors.New("member has left")
	// ErrAlreadyKeeper は、すでにおさいふ係の家族をおさいふ係にしようとしたときのエラー。
	ErrAlreadyKeeper = errors.New("member is already a keeper")
	// ErrNotKeeper は、おさいふ係ではない家族をおさいふ係から外そうとしたときのエラー。
	ErrNotKeeper = errors.New("member is not a keeper")
	// ErrLastKeeper は、最後のおさいふ係を外そうとしたときのエラー。
	ErrLastKeeper = errors.New("household must have at least one active keeper")
)

// Household は、アプリを使う単位であるおうち。
// おうちには、参加中のおさいふ係が常に 1 人以上いる（docs/domain/household.md）。
// 値を変える操作は、受け取ったおうちを変えずに、新しいおうちを返す。
type Household struct {
	members []Member
}

// New は、founder を最初のおさいふ係とするおうちを返す。
func New(founder MemberID) Household {
	return Household{members: []Member{{id: founder, role: RoleKeeper, status: StatusActive}}}
}

// Members は、外れた家族を含む、おうちのすべての家族を返す。
func (h Household) Members() []Member {
	return slices.Clone(h.members)
}

// Member は、指定した家族を返す。おうちにいなければ false を返す。
func (h Household) Member(id MemberID) (Member, bool) {
	i := h.index(id)
	if i < 0 {
		return Member{}, false
	}
	return h.members[i], true
}

// ActiveKeepers は、参加中のおさいふ係を返す。
func (h Household) ActiveKeepers() []Member {
	keepers := make([]Member, 0, len(h.members))
	for _, m := range h.members {
		if m.IsActiveKeeper() {
			keepers = append(keepers, m)
		}
	}
	return keepers
}

// AddMember は、おさいふ係ではない家族として id を加えたおうちを返す。
func (h Household) AddMember(id MemberID) (Household, error) {
	if h.index(id) >= 0 {
		return Household{}, fmt.Errorf("add member %s: %w", id, ErrDuplicateMember)
	}
	members := slices.Clone(h.members)
	members = append(members, Member{id: id, role: RoleNonKeeper, status: StatusActive})
	return Household{members: members}, nil
}

// MakeKeeper は、id をおさいふ係にしたおうちを返す。
// いいねが必要なことは、この関数の呼び出し側で扱う。
func (h Household) MakeKeeper(id MemberID) (Household, error) {
	return h.update(id, func(m Member) (Member, error) {
		if m.role == RoleKeeper {
			return Member{}, ErrAlreadyKeeper
		}
		m.role = RoleKeeper
		return m, nil
	})
}

// RemoveKeeper は、id をおさいふ係から外したおうちを返す。
// 最後のおさいふ係は外せない。いいねが必要なことは、この関数の呼び出し側で扱う。
func (h Household) RemoveKeeper(id MemberID) (Household, error) {
	return h.update(id, func(m Member) (Member, error) {
		if m.role != RoleKeeper {
			return Member{}, ErrNotKeeper
		}
		m.role = RoleNonKeeper
		return m, nil
	})
}

// RemoveMember は、id をおうちから外したおうちを返す。外した家族のデータは消さず、
// ようすを「外れた」にする。最後のおさいふ係は外せない。
func (h Household) RemoveMember(id MemberID) (Household, error) {
	return h.update(id, func(m Member) (Member, error) {
		m.status = StatusLeft
		return m, nil
	})
}

// update は、参加中の家族 id に change を適用し、必ず守る制約を確かめたうえで、
// 新しいおうちを返す。
func (h Household) update(id MemberID, change func(Member) (Member, error)) (Household, error) {
	i := h.index(id)
	if i < 0 {
		return Household{}, fmt.Errorf("member %s: %w", id, ErrMemberNotFound)
	}
	if h.members[i].status == StatusLeft {
		return Household{}, fmt.Errorf("member %s: %w", id, ErrMemberLeft)
	}
	changed, err := change(h.members[i])
	if err != nil {
		return Household{}, fmt.Errorf("member %s: %w", id, err)
	}
	members := slices.Clone(h.members)
	members[i] = changed
	next := Household{members: members}
	if len(next.ActiveKeepers()) == 0 {
		return Household{}, fmt.Errorf("member %s: %w", id, ErrLastKeeper)
	}
	return next, nil
}

func (h Household) index(id MemberID) int {
	return slices.IndexFunc(h.members, func(m Member) bool { return m.id == id })
}
