// Package like は、いいねのルールを表す（docs/domain/like.md）。
package like

import (
	"fmt"
	"slices"

	"github.com/kajiya-i/muda/backend/internal/domain/household"
)

// Route は、相談やおねがいのいいねの決まり方（いいねのルート）。
// ByKeepers と SelfLike のどちらかである。
//
//sumtype:decl
type Route interface {
	isRoute()
}

// ByKeepers は、おさいふ係がいいねするルート。いいねできるおさいふ係の一覧を持つ。
type ByKeepers struct {
	likers []household.MemberID
}

// SelfLike は、唯一のおさいふ係が自分の相談やおねがいにじぶんでいいねするルート。
type SelfLike struct {
	member household.MemberID
}

func (ByKeepers) isRoute() {}
func (SelfLike) isRoute()  {}

// Likers は、いいねできるおさいふ係の一覧を返す。
func (r ByKeepers) Likers() []household.MemberID {
	return slices.Clone(r.likers)
}

// CanLike は、member がいいねできるおさいふ係かどうかを返す。
func (r ByKeepers) CanLike(member household.MemberID) bool {
	return slices.Contains(r.likers, member)
}

// Member は、じぶんでいいねするおさいふ係を返す。
func (r SelfLike) Member() household.MemberID {
	return r.member
}

// DecideRoute は、相談やおねがいを出した家族 proposer と、そのときのおうちから、
// いいねのルートを決める。
//
//   - おさいふ係ではない家族が出したときは、参加中のおさいふ係がいいねする。
//   - おさいふ係が出し、ほかにもおさいふ係がいるときは、出した家族以外のおさいふ係がいいねする。
//   - 唯一のおさいふ係が出したときは、じぶんでいいねする。
//
// いいねのルートは、出した時点のおうちで決め、あとから変えない（docs/domain/like.md）。
func DecideRoute(h household.Household, proposer household.MemberID) (Route, error) {
	m, ok := h.Member(proposer)
	if !ok {
		return nil, fmt.Errorf("decide route for %s: %w", proposer, household.ErrMemberNotFound)
	}
	if m.Status() != household.StatusActive {
		return nil, fmt.Errorf("decide route for %s: %w", proposer, household.ErrMemberLeft)
	}

	keepers := h.ActiveKeepers()
	likers := make([]household.MemberID, 0, len(keepers))
	for _, k := range keepers {
		if k.ID() != proposer {
			likers = append(likers, k.ID())
		}
	}

	if len(likers) == 0 {
		// 参加中のおさいふ係は常に 1 人以上いるので、ここに来るのは、
		// 出した家族が唯一のおさいふ係のときだけである。
		return SelfLike{member: proposer}, nil
	}
	return ByKeepers{likers: likers}, nil
}
