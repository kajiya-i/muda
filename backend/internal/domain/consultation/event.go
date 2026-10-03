package consultation

import (
	"time"

	"github.com/kajiya-i/muda/backend/internal/domain/household"
	"github.com/kajiya-i/muda/backend/internal/domain/ledger"
	"github.com/kajiya-i/muda/backend/internal/domain/like"
	"github.com/kajiya-i/muda/backend/internal/domain/money"
	"github.com/kajiya-i/muda/backend/internal/domain/purpose"
)

// Event は、相談のようすが変わった事実（docs/domain/consultation.md）。
// イベント ID やバージョンなど、保存のための項目は持たず、ドメインの内容だけを持つ。
//
//sumtype:decl
type Event interface {
	isEvent()
	// OccurredAt は、イベントが起きた日時を返す。
	OccurredAt() time.Time
	// By は、イベントを起こした家族を返す。
	By() household.MemberID
}

// meta は、すべてのイベントが持つ、起きた日時と起こした家族。
type meta struct {
	at time.Time
	by household.MemberID
}

// OccurredAt は、イベントが起きた日時を返す。
func (m meta) OccurredAt() time.Time { return m.at }

// By は、イベントを起こした家族を返す。
func (m meta) By() household.MemberID { return m.by }

// Submitted は、相談を出したこと（ConsultationSubmitted）。
type Submitted struct {
	meta
	item           string
	amount         money.PositiveMoney
	purpose        purpose.Ref
	note           string
	route          like.Route
	overThrillLine bool
}

// LikeGiven は、おさいふ係がいいねしたこと（ConsultationLikeGiven）。
// 1 つ目のいいねでは、必要ないいねと、そのときのやりくりの残りを持つ。
type LikeGiven struct {
	meta
	first *FirstLike
}

// FirstLike は、1 つ目のいいねのときに決まり、記録されること。
type FirstLike struct {
	requirement     like.LikesFromKeepers
	remainingBudget money.Money
}

// Liked は、必要ないいねがそろったこと（ConsultationLiked）。
type Liked struct {
	meta
	deadlines Deadlines
}

// SelfLiked は、唯一のおさいふ係がじぶんでいいねしたこと（ConsultationSelfLiked）。
type SelfLiked struct {
	meta
	deadlines Deadlines
}

// Passed は、おさいふ係が見送ったこと（ConsultationPassed）。
type Passed struct {
	meta
	pass like.ConsultationPass
}

// WithdrawalReason は、とりやめになったわけ。
type WithdrawalReason int

const (
	// WithdrawnByRequester は、相談した家族がとりやめたこと。
	WithdrawnByRequester WithdrawalReason = iota + 1
	// WithdrawnBecauseRequesterLeft は、相談した家族がおうちから外れたこと。
	WithdrawnBecauseRequesterLeft
)

// Withdrawn は、とりやめになったこと（ConsultationWithdrawn）。
type Withdrawn struct {
	meta
	reason WithdrawalReason
}

// PaymentMethod は、払い方。
type PaymentMethod int

const (
	// PaidFromWallet は、おさいふから払ったこと。
	PaidFromWallet PaymentMethod = iota + 1
	// PaidByAdvance は、たてかえで払ったこと。
	PaidByAdvance
)

// PurchaseReported は、買ったよ報告をしたこと（PurchaseReported）。
type PurchaseReported struct {
	meta
	amount money.PositiveMoney
	method PaymentMethod
}

// AdvanceRepaid は、たてかえのおかえしが済んだこと（AdvanceRepaid）。
type AdvanceRepaid struct {
	meta
	repayment ledger.RepaymentID
}

func (Submitted) isEvent()        {}
func (LikeGiven) isEvent()        {}
func (Liked) isEvent()            {}
func (SelfLiked) isEvent()        {}
func (Passed) isEvent()           {}
func (Withdrawn) isEvent()        {}
func (PurchaseReported) isEvent() {}
func (AdvanceRepaid) isEvent()    {}

// Item は、ほしいものを返す。
func (e Submitted) Item() string { return e.item }

// Amount は、相談した金額を返す。
func (e Submitted) Amount() money.PositiveMoney { return e.amount }

// Purpose は、つかいみちを返す。
func (e Submitted) Purpose() purpose.Ref { return e.purpose }

// Note は、ひとことを返す。
func (e Submitted) Note() string { return e.note }

// Route は、いいねのルートを返す。
func (e Submitted) Route() like.Route { return e.route }

// OverThrillLine は、相談した金額がどきどきラインを超えていたかを返す。
func (e Submitted) OverThrillLine() bool { return e.overThrillLine }

// First は、1 つ目のいいねで決まったことを返す。1 つ目のいいねでなければ false を返す。
func (e LikeGiven) First() (FirstLike, bool) {
	if e.first == nil {
		return FirstLike{}, false
	}
	return *e.first, true
}

// Deadlines は、買える期限と報告の期限を返す。
func (e Liked) Deadlines() Deadlines { return e.deadlines }

// Deadlines は、買える期限と報告の期限を返す。
func (e SelfLiked) Deadlines() Deadlines { return e.deadlines }

// Pass は、見送りの理由とコメントを返す。
func (e Passed) Pass() like.ConsultationPass { return e.pass }

// Reason は、とりやめになったわけを返す。
func (e Withdrawn) Reason() WithdrawalReason { return e.reason }

// Amount は、買った金額を返す。
func (e PurchaseReported) Amount() money.PositiveMoney { return e.amount }

// Method は、払い方を返す。
func (e PurchaseReported) Method() PaymentMethod { return e.method }

// Repayment は、おかえしの ID を返す。
func (e AdvanceRepaid) Repayment() ledger.RepaymentID { return e.repayment }
