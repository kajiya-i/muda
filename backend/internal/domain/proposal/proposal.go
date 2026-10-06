package proposal

import (
	"errors"
	"fmt"
	"time"

	"github.com/kajiya-i/muda/backend/internal/domain/household"
	"github.com/kajiya-i/muda/backend/internal/domain/like"
)

var (
	// ErrInvalidTransition は、今のようすに適用できないイベントを適用しようとしたときのエラー。
	ErrInvalidTransition = errors.New("invalid transition")
	// ErrNoEvents は、イベントが 1 つもないおねがいを読み戻そうとしたときのエラー。
	ErrNoEvents = errors.New("no events")
)

// Details は、おねがいを出したときに決まり、あとから変わらない内容。
type Details struct {
	proposer household.MemberID
	kind     Kind
	note     string
	route    like.Route
}

// Proposer は、おねがいを出した家族を返す。
func (d Details) Proposer() household.MemberID { return d.proposer }

// Kind は、おねがいの種類と内容を返す。
func (d Details) Kind() Kind { return d.kind }

// Note は、ひとことを返す。
func (d Details) Note() string { return d.note }

// Route は、いいねのルートを返す。
func (d Details) Route() like.Route { return d.route }

// State は、おねがいのようす。
//
//sumtype:decl
type State interface {
	isState()
	// Details は、おねがいの内容を返す。
	Details() Details
}

// AwaitingLike は、いいね待ち。
type AwaitingLike struct{ details Details }

// LikedState は、いいね済み。変更が起きたあとのようす。
type LikedState struct {
	details     Details
	liker       household.MemberID
	effectiveAt time.Time
}

// PassedState は、見送り。
type PassedState struct {
	details Details
	passer  household.MemberID
	pass    like.ProposalPass
}

// WithdrawnState は、とりやめ。
type WithdrawnState struct{ details Details }

func (AwaitingLike) isState()   {}
func (LikedState) isState()     {}
func (PassedState) isState()    {}
func (WithdrawnState) isState() {}

func (s AwaitingLike) Details() Details   { return s.details }
func (s LikedState) Details() Details     { return s.details }
func (s PassedState) Details() Details    { return s.details }
func (s WithdrawnState) Details() Details { return s.details }

// Liker は、いいねした家族を返す。じぶんでいいねのときは、出した家族である。
func (s LikedState) Liker() household.MemberID { return s.liker }

// EffectiveAt は、変更が有効になる瞬間を返す。
func (s LikedState) EffectiveAt() time.Time { return s.effectiveAt }

// Passer は、見送った家族を返す。
func (s PassedState) Passer() household.MemberID { return s.passer }

// Pass は、見送りの理由とコメントを返す。
func (s PassedState) Pass() like.ProposalPass { return s.pass }

// Proposal は、おねがい。ID、適用したイベントの数を表すバージョン、今のようすを持つ。
type Proposal struct {
	id      ID
	version int
	state   State
}

// ID は、おねがい ID を返す。
func (p Proposal) ID() ID { return p.id }

// Version は、適用したイベントの数を返す。
func (p Proposal) Version() int { return p.version }

// State は、今のようすを返す。
func (p Proposal) State() State { return p.state }

// Start は、おねがいを出したイベントから、おねがいを始める。
func Start(id ID, e Submitted) Proposal {
	return Proposal{id: id, version: 1, state: AwaitingLike{details: Details{
		proposer: e.by, kind: e.kind, note: e.note, route: e.route,
	}}}
}

// Evolve は、おねがいにイベントを適用して、新しいおねがいを返す。受け取ったおねがいは変えない。
// 今のようすに適用できないイベントは、ErrInvalidTransition を返す。
func Evolve(p Proposal, e Event) (Proposal, error) {
	a, ok := p.state.(AwaitingLike)
	if !ok {
		return Proposal{}, fmt.Errorf("evolve proposal %s from %T with %T: %w", p.id, p.state, e, ErrInvalidTransition)
	}
	var next State
	switch e := e.(type) {
	case Submitted:
		return Proposal{}, fmt.Errorf("evolve proposal %s with %T: %w", p.id, e, ErrInvalidTransition)
	case Liked:
		next = LikedState{details: a.details, liker: e.by, effectiveAt: e.effectiveAt}
	case SelfLiked:
		next = LikedState{details: a.details, liker: e.by, effectiveAt: e.effectiveAt}
	case Passed:
		next = PassedState{details: a.details, passer: e.by, pass: e.pass}
	case Withdrawn:
		next = WithdrawnState(a)
	}
	return Proposal{id: p.id, version: p.version + 1, state: next}, nil
}

// Replay は、保存したイベントを最初から順に適用して、おねがいを読み戻す。
func Replay(id ID, events []Event) (Proposal, error) {
	if len(events) == 0 {
		return Proposal{}, fmt.Errorf("replay proposal %s: %w", id, ErrNoEvents)
	}
	first, ok := events[0].(Submitted)
	if !ok {
		return Proposal{}, fmt.Errorf("replay proposal %s: first event is %T: %w", id, events[0], ErrInvalidTransition)
	}
	p := Start(id, first)
	for _, e := range events[1:] {
		var err error
		if p, err = Evolve(p, e); err != nil {
			return Proposal{}, err
		}
	}
	return p, nil
}
