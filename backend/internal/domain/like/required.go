package like

import (
	"github.com/kajiya-i/muda/backend/internal/domain/money"
)

// Reasons は、相談にいいねが 2 つ必要になる理由。家族に見えるように記録する。
type Reasons struct {
	overThrillLine      bool
	overRemainingBudget bool
}

// NewReasons は、どきどきラインを超えているか、やりくりの残りを超えているかから、
// 理由を返す。
func NewReasons(overThrillLine, overRemainingBudget bool) Reasons {
	return Reasons{overThrillLine: overThrillLine, overRemainingBudget: overRemainingBudget}
}

// OverThrillLine は、相談した金額がどきどきラインを超えているかを返す。
func (r Reasons) OverThrillLine() bool { return r.overThrillLine }

// OverRemainingBudget は、相談した金額がやりくりの残りを超えているかを返す。
func (r Reasons) OverRemainingBudget() bool { return r.overRemainingBudget }

// Any は、いいねが 2 つ必要になる理由が 1 つでもあるかを返す。
func (r Reasons) Any() bool { return r.overThrillLine || r.overRemainingBudget }

// ExceedsThrillLine は、相談した金額がどきどきラインを超えているかを返す。
// ちょうど同じ金額は、超えていないものとする（docs/domain/like.md）。
// 相談を出したときに判断する。
func ExceedsThrillLine(amount money.PositiveMoney, line money.NonNegativeMoney) (bool, error) {
	c, err := amount.Money().Compare(line.Money())
	if err != nil {
		return false, err
	}
	return c > 0, nil
}

// ExceedsRemainingBudget は、相談した金額がやりくりの残りを超えているかを返す。
// やりくりの残りがマイナスのときは、必ず超えている。1 つ目のいいねのときに判断する。
func ExceedsRemainingBudget(amount money.PositiveMoney, remaining money.Money) (bool, error) {
	c, err := amount.Money().Compare(remaining)
	if err != nil {
		return false, err
	}
	return c > 0, nil
}

// Requirement は、相談やおねがいが認められるために必要ないいね。
// LikesFromKeepers と LikeFromSelf のどちらかである。
//
//sumtype:decl
type Requirement interface {
	isRequirement()
}

// LikesFromKeepers は、おさいふ係から指定した数のいいねが必要なこと。
type LikesFromKeepers struct {
	count   int
	reasons Reasons
}

// LikeFromSelf は、じぶんでいいねで認められること。
type LikeFromSelf struct {
	reasons Reasons
}

func (LikesFromKeepers) isRequirement() {}
func (LikeFromSelf) isRequirement()     {}

// Count は、必要ないいねの数（1 か 2）を返す。
func (r LikesFromKeepers) Count() int { return r.count }

// Reasons は、いいねが 2 つ必要になる理由を返す。いいねする人が 1 人しかいないために
// 1 つになった場合も、理由は記録として残る。
func (r LikesFromKeepers) Reasons() Reasons { return r.reasons }

// Reasons は、いいねが 2 つ必要になる理由を返す。じぶんでいいねの場合も、
// 理由は家族に見える記録として残る。
func (r LikeFromSelf) Reasons() Reasons { return r.reasons }

// ConsultationRequirement は、相談のいいねのルートと理由から、必要ないいねを決める。
//
//   - 理由がなければ、いいねは 1 つ。
//   - 理由が 1 つでもあれば、いいねは 2 つ。ただし、いいねする人が 1 人しかいなければ 1 つ。
//   - じぶんでいいねのルートなら、理由にかかわらず、じぶんでいいねで認められる。
//
// 必要ないいねは、1 つ目のいいねのときに決まり、あとから変わらない（docs/domain/like.md）。
func ConsultationRequirement(route Route, reasons Reasons) Requirement {
	switch r := route.(type) {
	case ByKeepers:
		count := 1
		if reasons.Any() {
			count = 2
		}
		return LikesFromKeepers{count: min(count, len(r.likers)), reasons: reasons}
	case SelfLike:
		return LikeFromSelf{reasons: reasons}
	}
	panic("like: unknown route")
}

// ProposalRequirement は、おねがいのいいねのルートから、必要ないいねを決める。
// おねがいに必要ないいねは 1 つで、じぶんでいいねのルートならじぶんでいいねで認められる。
func ProposalRequirement(route Route) Requirement {
	switch route.(type) {
	case ByKeepers:
		return LikesFromKeepers{count: 1}
	case SelfLike:
		return LikeFromSelf{}
	}
	panic("like: unknown route")
}
