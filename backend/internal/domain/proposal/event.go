package proposal

import (
	"time"

	"github.com/kajiya-i/muda/backend/internal/domain/household"
	"github.com/kajiya-i/muda/backend/internal/domain/like"
)

// Event は、おねがいのようすが変わった事実（docs/domain/proposal.md）。
//
//sumtype:decl
type Event interface {
	isEvent()
	// OccurredAt は、イベントが起きた日時を返す。
	OccurredAt() time.Time
	// By は、イベントを起こした家族を返す。
	By() household.MemberID
}

type meta struct {
	at time.Time
	by household.MemberID
}

// OccurredAt は、イベントが起きた日時を返す。
func (m meta) OccurredAt() time.Time { return m.at }

// By は、イベントを起こした家族を返す。
func (m meta) By() household.MemberID { return m.by }

// Submitted は、おねがいを出したこと（ProposalSubmitted）。
type Submitted struct {
	meta
	kind  Kind
	note  string
	route like.Route
}

// Liked は、いいねする人がいいねしたこと（ProposalLiked）。
type Liked struct {
	meta
	effectiveAt time.Time
}

// SelfLiked は、唯一のおさいふ係がじぶんでいいねしたこと（ProposalSelfLiked）。
type SelfLiked struct {
	meta
	effectiveAt time.Time
}

// Passed は、いいねする人が見送ったこと（ProposalPassed）。
type Passed struct {
	meta
	pass like.ProposalPass
}

// Withdrawn は、出した家族がとりやめたこと（ProposalWithdrawn）。
type Withdrawn struct {
	meta
}

func (Submitted) isEvent() {}
func (Liked) isEvent()     {}
func (SelfLiked) isEvent() {}
func (Passed) isEvent()    {}
func (Withdrawn) isEvent() {}

// NewSubmitted は、おねがいを出したイベントを返す。保存したイベントを読み戻すときに使う。
func NewSubmitted(at time.Time, by household.MemberID, kind Kind, note string, route like.Route) Submitted {
	return Submitted{meta: meta{at: at, by: by}, kind: kind, note: note, route: route}
}

// NewLiked は、いいねしたイベントを返す。保存したイベントを読み戻すときに使う。
func NewLiked(at time.Time, by household.MemberID, effectiveAt time.Time) Liked {
	return Liked{meta: meta{at: at, by: by}, effectiveAt: effectiveAt}
}

// NewSelfLiked は、じぶんでいいねしたイベントを返す。保存したイベントを読み戻すときに使う。
func NewSelfLiked(at time.Time, by household.MemberID, effectiveAt time.Time) SelfLiked {
	return SelfLiked{meta: meta{at: at, by: by}, effectiveAt: effectiveAt}
}

// NewPassed は、見送ったイベントを返す。保存したイベントを読み戻すときに使う。
func NewPassed(at time.Time, by household.MemberID, pass like.ProposalPass) Passed {
	return Passed{meta: meta{at: at, by: by}, pass: pass}
}

// NewWithdrawn は、とりやめたイベントを返す。保存したイベントを読み戻すときに使う。
func NewWithdrawn(at time.Time, by household.MemberID) Withdrawn {
	return Withdrawn{meta: meta{at: at, by: by}}
}

// Kind は、おねがいの種類と内容を返す。
func (e Submitted) Kind() Kind { return e.kind }

// Note は、ひとことを返す。
func (e Submitted) Note() string { return e.note }

// Route は、いいねのルートを返す。
func (e Submitted) Route() like.Route { return e.route }

// EffectiveAt は、変更が有効になる瞬間を返す。
func (e Liked) EffectiveAt() time.Time { return e.effectiveAt }

// EffectiveAt は、変更が有効になる瞬間を返す。
func (e SelfLiked) EffectiveAt() time.Time { return e.effectiveAt }

// Pass は、見送りの理由とコメントを返す。
func (e Passed) Pass() like.ProposalPass { return e.pass }
