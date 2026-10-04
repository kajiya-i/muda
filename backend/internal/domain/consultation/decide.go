package consultation

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/kajiya-i/muda/backend/internal/domain/household"
	"github.com/kajiya-i/muda/backend/internal/domain/like"
	"github.com/kajiya-i/muda/backend/internal/domain/money"
	"github.com/kajiya-i/muda/backend/internal/domain/purpose"
)

var (
	// ErrEmptyItem は、ほしいものが空のときのエラー。
	ErrEmptyItem = errors.New("item is empty")
	// ErrNotAwaitingLikes は、いいね待ちではない相談にいいねや見送りをしようとしたときのエラー。
	ErrNotAwaitingLikes = errors.New("consultation is not awaiting likes")
	// ErrNotLiker は、いいねのルートに含まれない家族がいいねや見送りをしようとしたときのエラー。
	ErrNotLiker = errors.New("member cannot like or pass this consultation")
	// ErrAlreadyLiked は、すでにいいねした家族が、もう一度いいねや見送りをしようとしたときのエラー。
	ErrAlreadyLiked = errors.New("member has already liked this consultation")
	// ErrCannotWithdraw は、とりやめられないようすの相談をとりやめようとしたときのエラー。
	ErrCannotWithdraw = errors.New("consultation cannot be withdrawn")
	// ErrNotRequester は、相談した家族ではない家族が、相談した家族だけができる操作をしようとしたときのエラー。
	ErrNotRequester = errors.New("member is not the requester")
)

// SubmitInput は、相談を出すときに家族が伝える内容。
type SubmitInput struct {
	Item    string
	Amount  money.PositiveMoney
	Purpose purpose.Ref
	Note    string
}

// Submit は、相談を出すことを決める。
// いいねのルートと、相談した金額がどきどきラインを超えているかは、出したときのおうちと
// どきどきラインで決めて記録する（docs/domain/like.md）。
func Submit(requester household.MemberID, in SubmitInput, h household.Household,
	thrillLine money.NonNegativeMoney, now time.Time,
) (Submitted, error) {
	if in.Item == "" {
		return Submitted{}, fmt.Errorf("submit: %w", ErrEmptyItem)
	}
	route, err := like.DecideRoute(h, requester)
	if err != nil {
		return Submitted{}, fmt.Errorf("submit: %w", err)
	}
	over, err := like.ExceedsThrillLine(in.Amount, thrillLine)
	if err != nil {
		return Submitted{}, fmt.Errorf("submit: %w", err)
	}
	return NewSubmitted(now, requester, in.Item, in.Amount, in.Purpose, in.Note, route, over), nil
}

// Like は、おさいふ係 liker のいいねを決める。
//
// remainingBudget は、今のやりくりの残りである。1 つ目のいいね（じぶんでいいねを含む）のときだけ
// 使い、必要ないいねの数を決めて記録する。2 つ目以降のいいねでは使わない。
// loc はおうちの時間で、いいねがそろったときの期限の計算に使う。
//
// いいねがそろうと、いいね済みになったことを表すイベントも合わせて返す。
func Like(c Consultation, liker household.MemberID, remainingBudget money.Money,
	loc *time.Location, now time.Time,
) ([]Event, error) {
	a, ok := c.state.(AwaitingLikes)
	if !ok {
		return nil, fmt.Errorf("like consultation %s: %w", c.id, ErrNotAwaitingLikes)
	}

	switch route := a.details.route.(type) {
	case like.SelfLike:
		if route.Member() != liker {
			return nil, fmt.Errorf("like consultation %s: %w", c.id, ErrNotLiker)
		}
		reasons, err := reasonsAtFirstLike(a.details, remainingBudget)
		if err != nil {
			return nil, fmt.Errorf("like consultation %s: %w", c.id, err)
		}
		requirement, err := selfRequirement(route, reasons)
		if err != nil {
			return nil, fmt.Errorf("like consultation %s: %w", c.id, err)
		}
		return []Event{NewSelfLiked(now, liker, NewDeadlines(now, loc), requirement, remainingBudget)}, nil

	case like.ByKeepers:
		if !route.CanLike(liker) {
			return nil, fmt.Errorf("like consultation %s: %w", c.id, ErrNotLiker)
		}
		if slices.Contains(a.likes, liker) {
			return nil, fmt.Errorf("like consultation %s: %w", c.id, ErrAlreadyLiked)
		}

		var given LikeGiven
		first, decided := a.FirstLike()
		if decided {
			given = NewLikeGiven(now, liker)
		} else {
			reasons, err := reasonsAtFirstLike(a.details, remainingBudget)
			if err != nil {
				return nil, fmt.Errorf("like consultation %s: %w", c.id, err)
			}
			keepers, err := keepersRequirement(route, reasons)
			if err != nil {
				return nil, fmt.Errorf("like consultation %s: %w", c.id, err)
			}
			first = NewFirstLike(keepers, remainingBudget)
			given = NewFirstLikeGiven(now, liker, first)
		}

		events := []Event{given}
		if len(a.likes)+1 >= first.Requirement().Count() {
			events = append(events, NewLiked(now, liker, NewDeadlines(now, loc)))
		}
		return events, nil
	}
	panic("consultation: unknown route")
}

// Pass は、おさいふ係 passer の見送りを決める。1 人でも見送れば、その相談は見送りになる。
// じぶんでいいねのルートの相談は、見送れない（相談した家族は、とりやめを使う）。
func Pass(c Consultation, passer household.MemberID, pass like.ConsultationPass, now time.Time) (Passed, error) {
	a, ok := c.state.(AwaitingLikes)
	if !ok {
		return Passed{}, fmt.Errorf("pass consultation %s: %w", c.id, ErrNotAwaitingLikes)
	}
	route, ok := a.details.route.(like.ByKeepers)
	if !ok || !route.CanLike(passer) {
		return Passed{}, fmt.Errorf("pass consultation %s: %w", c.id, ErrNotLiker)
	}
	if slices.Contains(a.likes, passer) {
		return Passed{}, fmt.Errorf("pass consultation %s: %w", c.id, ErrAlreadyLiked)
	}
	return NewPassed(now, passer, pass), nil
}

// Withdraw は、相談のとりやめを決める。とりやめられるのは、いいね待ちかいいね済みの相談である。
// 相談した家族がとりやめるときは、by は相談した家族でなければならない。
// 相談した家族がおうちから外れたためにとりやめるときは、by はその家族を外したおさいふ係である。
func Withdraw(c Consultation, by household.MemberID, reason WithdrawalReason, now time.Time) (Withdrawn, error) {
	switch c.state.(type) {
	case AwaitingLikes, LikedState:
	default:
		return Withdrawn{}, fmt.Errorf("withdraw consultation %s: %w", c.id, ErrCannotWithdraw)
	}
	if reason == WithdrawnByRequester && c.state.Details().requester != by {
		return Withdrawn{}, fmt.Errorf("withdraw consultation %s: %w", c.id, ErrNotRequester)
	}
	return NewWithdrawn(now, by, reason), nil
}

// reasonsAtFirstLike は、1 つ目のいいねのときに、いいねが 2 つ必要になる理由を決める。
// どきどきラインは相談を出したときに記録した結果を使い、やりくりの残りはこのときに判断する。
func reasonsAtFirstLike(d Details, remainingBudget money.Money) (like.Reasons, error) {
	overBudget, err := like.ExceedsRemainingBudget(d.amount, remainingBudget)
	if err != nil {
		return like.Reasons{}, err
	}
	return like.NewReasons(d.overThrillLine, overBudget), nil
}

var errUnexpectedRequirement = errors.New("unexpected requirement for the route")

func selfRequirement(route like.SelfLike, reasons like.Reasons) (like.LikeFromSelf, error) {
	switch r := like.ConsultationRequirement(route, reasons).(type) {
	case like.LikeFromSelf:
		return r, nil
	case like.LikesFromKeepers:
		return like.LikeFromSelf{}, errUnexpectedRequirement
	}
	panic("consultation: unknown requirement")
}

func keepersRequirement(route like.ByKeepers, reasons like.Reasons) (like.LikesFromKeepers, error) {
	switch r := like.ConsultationRequirement(route, reasons).(type) {
	case like.LikesFromKeepers:
		return r, nil
	case like.LikeFromSelf:
		return like.LikesFromKeepers{}, errUnexpectedRequirement
	}
	panic("consultation: unknown requirement")
}
