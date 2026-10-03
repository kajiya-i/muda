package consultation

import (
	"slices"

	"github.com/kajiya-i/muda/backend/internal/domain/household"
	"github.com/kajiya-i/muda/backend/internal/domain/ledger"
	"github.com/kajiya-i/muda/backend/internal/domain/like"
	"github.com/kajiya-i/muda/backend/internal/domain/money"
	"github.com/kajiya-i/muda/backend/internal/domain/purpose"
)

// Details は、相談を出したときに決まり、あとから変わらない相談の内容。
type Details struct {
	requester      household.MemberID
	item           string
	amount         money.PositiveMoney
	purpose        purpose.Ref
	note           string
	route          like.Route
	overThrillLine bool
}

// Requester は、相談した家族を返す。
func (d Details) Requester() household.MemberID { return d.requester }

// Item は、ほしいものを返す。
func (d Details) Item() string { return d.item }

// Amount は、相談した金額を返す。
func (d Details) Amount() money.PositiveMoney { return d.amount }

// Purpose は、つかいみちを返す。
func (d Details) Purpose() purpose.Ref { return d.purpose }

// Note は、ひとことを返す。
func (d Details) Note() string { return d.note }

// Route は、いいねのルートを返す。
func (d Details) Route() like.Route { return d.route }

// OverThrillLine は、相談した金額がどきどきラインを超えていたかを返す。
func (d Details) OverThrillLine() bool { return d.overThrillLine }

// State は、相談のようす。したがきはイベントにしないので、ここには含まない。
// 時間切れは、いいね済みのまま報告の期限を過ぎた状態として、時刻から導出する（IsExpired）。
//
//sumtype:decl
type State interface {
	isState()
	// Details は、相談の内容を返す。
	Details() Details
}

// AwaitingLikes は、いいね待ち。いいねが 2 つ必要な相談で、1 つ目のいいねをもらった途中の段階も含む。
type AwaitingLikes struct {
	details   Details
	likes     []household.MemberID
	firstLike *FirstLike
}

// LikedState は、いいね済み。
type LikedState struct {
	details   Details
	likes     []household.MemberID
	deadlines Deadlines
}

// PassedState は、見送り。
type PassedState struct {
	details Details
	likes   []household.MemberID
	passer  household.MemberID
	pass    like.ConsultationPass
}

// WithdrawnState は、とりやめ。
type WithdrawnState struct {
	details Details
	reason  WithdrawalReason
}

// PurchasedState は、買ったよ。
type PurchasedState struct {
	details Details
	amount  money.PositiveMoney
	method  PaymentMethod
}

// RepaidState は、おかえし済み。
type RepaidState struct {
	details   Details
	amount    money.PositiveMoney
	repayment ledger.RepaymentID
}

func (AwaitingLikes) isState()  {}
func (LikedState) isState()     {}
func (PassedState) isState()    {}
func (WithdrawnState) isState() {}
func (PurchasedState) isState() {}
func (RepaidState) isState()    {}

func (s AwaitingLikes) Details() Details  { return s.details }
func (s LikedState) Details() Details     { return s.details }
func (s PassedState) Details() Details    { return s.details }
func (s WithdrawnState) Details() Details { return s.details }
func (s PurchasedState) Details() Details { return s.details }
func (s RepaidState) Details() Details    { return s.details }

// Likes は、これまでにいいねしたおさいふ係を返す。
func (s AwaitingLikes) Likes() []household.MemberID { return slices.Clone(s.likes) }

// FirstLike は、1 つ目のいいねで決まったことを返す。まだいいねがなければ false を返す。
func (s AwaitingLikes) FirstLike() (FirstLike, bool) {
	if s.firstLike == nil {
		return FirstLike{}, false
	}
	return *s.firstLike, true
}

// Likes は、いいねしたおさいふ係を返す。じぶんでいいねのときは空である。
func (s LikedState) Likes() []household.MemberID { return slices.Clone(s.likes) }

// Deadlines は、買える期限と報告の期限を返す。
func (s LikedState) Deadlines() Deadlines { return s.deadlines }

// Passer は、見送ったおさいふ係を返す。
func (s PassedState) Passer() household.MemberID { return s.passer }

// Pass は、見送りの理由とコメントを返す。
func (s PassedState) Pass() like.ConsultationPass { return s.pass }

// Reason は、とりやめになったわけを返す。
func (s WithdrawnState) Reason() WithdrawalReason { return s.reason }

// Amount は、買った金額を返す。
func (s PurchasedState) Amount() money.PositiveMoney { return s.amount }

// Method は、払い方を返す。
func (s PurchasedState) Method() PaymentMethod { return s.method }

// Amount は、買った金額を返す。
func (s RepaidState) Amount() money.PositiveMoney { return s.amount }

// Repayment は、おかえしの ID を返す。
func (s RepaidState) Repayment() ledger.RepaymentID { return s.repayment }

// Requirement は、1 つ目のいいねで決まった必要ないいねを返す。
func (f FirstLike) Requirement() like.LikesFromKeepers { return f.requirement }

// RemainingBudget は、1 つ目のいいねのときのやりくりの残りを返す。
func (f FirstLike) RemainingBudget() money.Money { return f.remainingBudget }
