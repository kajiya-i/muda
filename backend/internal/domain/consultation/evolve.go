package consultation

import (
	"errors"
	"fmt"
	"slices"
	"time"
)

var (
	// ErrInvalidTransition は、今のようすに適用できないイベントを適用しようとしたときのエラー。
	// Decide が返すイベントでは起こらず、保存したイベントが壊れているときなどに起こる。
	ErrInvalidTransition = errors.New("invalid transition")
	// ErrNoEvents は、イベントが 1 つもない相談を読み戻そうとしたときのエラー。
	ErrNoEvents = errors.New("no events")
)

// Consultation は、相談。ID、いくつのイベントを適用したかを表すバージョン、今のようすを持つ。
type Consultation struct {
	id      ID
	version int
	state   State
}

// ID は、相談 ID を返す。
func (c Consultation) ID() ID { return c.id }

// Version は、適用したイベントの数を返す。保存するときの楽観的ロックに使う（ADR-0007）。
func (c Consultation) Version() int { return c.version }

// State は、今のようすを返す。
func (c Consultation) State() State { return c.state }

// Start は、相談を出したイベントから、相談を始める。
func Start(id ID, e Submitted) Consultation {
	return Consultation{
		id:      id,
		version: 1,
		state: AwaitingLikes{details: Details{
			requester: e.by, item: e.item, amount: e.amount, purpose: e.purpose,
			note: e.note, route: e.route, overThrillLine: e.overThrillLine,
		}},
	}
}

// Evolve は、相談にイベントを適用して、新しい相談を返す。受け取った相談は変えない。
// ルールの検証は Decide が行い、Evolve はイベントをそのまま適用する。
// ただし、今のようすに適用できないイベントは、ErrInvalidTransition を返す。
func Evolve(c Consultation, e Event) (Consultation, error) {
	next, err := apply(c.state, e)
	if err != nil {
		return Consultation{}, fmt.Errorf("evolve consultation %s from %T with %T: %w", c.id, c.state, e, err)
	}
	return Consultation{id: c.id, version: c.version + 1, state: next}, nil
}

// Replay は、保存したイベントを最初から順に適用して、相談を読み戻す。
// 最初のイベントは、相談を出したイベントでなければならない。
func Replay(id ID, events []Event) (Consultation, error) {
	if len(events) == 0 {
		return Consultation{}, fmt.Errorf("replay consultation %s: %w", id, ErrNoEvents)
	}
	first, ok := events[0].(Submitted)
	if !ok {
		return Consultation{}, fmt.Errorf("replay consultation %s: first event is %T: %w", id, events[0], ErrInvalidTransition)
	}
	c := Start(id, first)
	for _, e := range events[1:] {
		var err error
		if c, err = Evolve(c, e); err != nil {
			return Consultation{}, err
		}
	}
	return c, nil
}

// IsExpired は、now の時点で相談が時間切れか（いいね済みのまま報告の期限を過ぎたか）を返す。
func IsExpired(c Consultation, now time.Time) bool {
	liked, ok := c.state.(LikedState)
	return ok && liked.deadlines.ReportExpired(now)
}

func apply(s State, e Event) (State, error) {
	switch e := e.(type) {
	case Submitted:
		return nil, ErrInvalidTransition
	case LikeGiven:
		a, ok := s.(AwaitingLikes)
		if !ok {
			return nil, ErrInvalidTransition
		}
		likes := append(slices.Clone(a.likes), e.by)
		first := a.firstLike
		if e.first != nil {
			f := *e.first
			first = &f
		}
		return AwaitingLikes{details: a.details, likes: likes, firstLike: first}, nil
	case Liked:
		a, ok := s.(AwaitingLikes)
		if !ok {
			return nil, ErrInvalidTransition
		}
		return LikedState{details: a.details, likes: slices.Clone(a.likes), deadlines: e.deadlines}, nil
	case SelfLiked:
		a, ok := s.(AwaitingLikes)
		if !ok {
			return nil, ErrInvalidTransition
		}
		return LikedState{details: a.details, deadlines: e.deadlines}, nil
	case Passed:
		a, ok := s.(AwaitingLikes)
		if !ok {
			return nil, ErrInvalidTransition
		}
		return PassedState{details: a.details, likes: slices.Clone(a.likes), passer: e.by, pass: e.pass}, nil
	case Withdrawn:
		switch s.(type) {
		case AwaitingLikes, LikedState:
			return WithdrawnState{details: s.Details(), reason: e.reason}, nil
		default:
			return nil, ErrInvalidTransition
		}
	case PurchaseReported:
		l, ok := s.(LikedState)
		if !ok {
			return nil, ErrInvalidTransition
		}
		return PurchasedState{details: l.details, amount: e.amount, method: e.method}, nil
	case AdvanceRepaid:
		p, ok := s.(PurchasedState)
		if !ok || p.method != PaidByAdvance {
			return nil, ErrInvalidTransition
		}
		return RepaidState{details: p.details, amount: p.amount, repayment: e.repayment}, nil
	}
	panic("consultation: unknown event")
}
