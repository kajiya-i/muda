package consultation

import (
	"time"

	"github.com/kajiya-i/muda/backend/internal/domain/household"
	"github.com/kajiya-i/muda/backend/internal/domain/ledger"
	"github.com/kajiya-i/muda/backend/internal/domain/like"
	"github.com/kajiya-i/muda/backend/internal/domain/money"
	"github.com/kajiya-i/muda/backend/internal/domain/purpose"
)

// 以下のコンストラクタは、保存したイベントを読み戻すときに使う。
// 新しいイベントは、Decide が作る。

// NewSubmitted は、相談を出したイベントを返す。
func NewSubmitted(at time.Time, by household.MemberID, item string, amount money.PositiveMoney,
	p purpose.Ref, note string, route like.Route, overThrillLine bool,
) Submitted {
	return Submitted{
		meta: meta{at: at, by: by}, item: item, amount: amount, purpose: p,
		note: note, route: route, overThrillLine: overThrillLine,
	}
}

// NewFirstLike は、1 つ目のいいねで決まったことを返す。
func NewFirstLike(requirement like.LikesFromKeepers, remainingBudget money.Money) FirstLike {
	return FirstLike{requirement: requirement, remainingBudget: remainingBudget}
}

// NewLikeGiven は、1 つ目ではないいいねのイベントを返す。
func NewLikeGiven(at time.Time, by household.MemberID) LikeGiven {
	return LikeGiven{meta: meta{at: at, by: by}}
}

// NewFirstLikeGiven は、1 つ目のいいねのイベントを返す。
func NewFirstLikeGiven(at time.Time, by household.MemberID, first FirstLike) LikeGiven {
	return LikeGiven{meta: meta{at: at, by: by}, first: &first}
}

// NewLiked は、必要ないいねがそろったイベントを返す。
func NewLiked(at time.Time, by household.MemberID, deadlines Deadlines) Liked {
	return Liked{meta: meta{at: at, by: by}, deadlines: deadlines}
}

// NewSelfLiked は、じぶんでいいねしたイベントを返す。
func NewSelfLiked(at time.Time, by household.MemberID, deadlines Deadlines,
	requirement like.LikeFromSelf, remainingBudget money.Money,
) SelfLiked {
	return SelfLiked{
		meta: meta{at: at, by: by}, deadlines: deadlines,
		requirement: requirement, remainingBudget: remainingBudget,
	}
}

// NewPassed は、見送ったイベントを返す。
func NewPassed(at time.Time, by household.MemberID, pass like.ConsultationPass) Passed {
	return Passed{meta: meta{at: at, by: by}, pass: pass}
}

// NewWithdrawn は、とりやめになったイベントを返す。
func NewWithdrawn(at time.Time, by household.MemberID, reason WithdrawalReason) Withdrawn {
	return Withdrawn{meta: meta{at: at, by: by}, reason: reason}
}

// NewPurchaseReported は、買ったよ報告のイベントを返す。
func NewPurchaseReported(at time.Time, by household.MemberID, amount money.PositiveMoney, method PaymentMethod) PurchaseReported {
	return PurchaseReported{meta: meta{at: at, by: by}, amount: amount, method: method}
}

// NewAdvanceRepaid は、たてかえのおかえしが済んだイベントを返す。
func NewAdvanceRepaid(at time.Time, by household.MemberID, repayment ledger.RepaymentID) AdvanceRepaid {
	return AdvanceRepaid{meta: meta{at: at, by: by}, repayment: repayment}
}
