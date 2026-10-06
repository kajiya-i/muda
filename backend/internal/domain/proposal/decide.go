package proposal

import (
	"errors"
	"fmt"
	"time"

	"github.com/kajiya-i/muda/backend/internal/domain/household"
	"github.com/kajiya-i/muda/backend/internal/domain/ledger"
	"github.com/kajiya-i/muda/backend/internal/domain/like"
	"github.com/kajiya-i/muda/backend/internal/domain/settings"
)

var (
	// ErrNotAwaitingLike は、いいね待ちではないおねがいに、いいね・見送り・とりやめをしようとしたときのエラー。
	ErrNotAwaitingLike = errors.New("proposal is not awaiting a like")
	// ErrNotLiker は、いいねのルートに含まれない家族がいいねや見送りをしようとしたときのエラー。
	ErrNotLiker = errors.New("member cannot like or pass this proposal")
	// ErrNotProposer は、出した家族ではない家族がとりやめようとしたときのエラー。
	ErrNotProposer = errors.New("member is not the proposer")
)

// Outcome は、いいねがそろったときに起きる変更。ユースケースは、いいね済みのイベントと同じ
// トランザクションで、この変更を記録する（ADR-0007）。
//
//sumtype:decl
type Outcome interface {
	isOutcome()
}

// SettingsChanged は、おうちの設定の履歴に加える 1 行。
type SettingsChanged struct{ entry settings.Entry }

// TakenOut は、台帳に記録する、おさいふ → 拠出の移動。
type TakenOut struct{ transfer ledger.Transfer }

// HouseholdChanged は、おさいふ係を変えたあとのおうち。
type HouseholdChanged struct{ household household.Household }

func (SettingsChanged) isOutcome()  {}
func (TakenOut) isOutcome()         {}
func (HouseholdChanged) isOutcome() {}

// Entry は、おうちの設定の履歴に加える 1 行を返す。
func (o SettingsChanged) Entry() settings.Entry { return o.entry }

// Transfer は、台帳に記録する移動を返す。
func (o TakenOut) Transfer() ledger.Transfer { return o.transfer }

// Household は、変えたあとのおうちを返す。
func (o HouseholdChanged) Household() household.Household { return o.household }

// Submit は、おさいふ係 proposer がおねがいを出すことを決める。
func Submit(proposer household.MemberID, kind Kind, note string, ctx Context, now time.Time) (Submitted, error) {
	m, ok := ctx.Household.Member(proposer)
	if !ok || !m.IsActiveKeeper() {
		return Submitted{}, fmt.Errorf("submit proposal: %w", household.ErrNotKeeper)
	}
	route, err := like.DecideRoute(ctx.Household, proposer)
	if err != nil {
		return Submitted{}, fmt.Errorf("submit proposal: %w", err)
	}
	if err := Check(kind, ctx); err != nil {
		return Submitted{}, fmt.Errorf("submit proposal: %w", err)
	}
	return NewSubmitted(now, proposer, kind, note, route), nil
}

// Like は、liker のいいねを決める。いいねのときのおうちのようす ctx で、おねがいの内容を改めて確かめ、
// いいね済みのイベントと、いいねがそろったときに起きる変更を返す。
func Like(p Proposal, liker household.MemberID, ctx Context, now time.Time) (Event, Outcome, error) {
	a, ok := p.state.(AwaitingLike)
	if !ok {
		return nil, nil, fmt.Errorf("like proposal %s: %w", p.id, ErrNotAwaitingLike)
	}
	if !canLike(a.details.route, liker) {
		return nil, nil, fmt.Errorf("like proposal %s: %w", p.id, ErrNotLiker)
	}
	if err := Check(a.details.kind, ctx); err != nil {
		return nil, nil, fmt.Errorf("like proposal %s: %w", p.id, err)
	}

	effectiveAt := now
	var outcome Outcome
	switch k := a.details.kind.(type) {
	case ChangeMonthlyBudget:
		outcome = settingsChanged(settings.NewBudgetChange(k.amount), now, ctx)
	case ChangeThrillLine:
		outcome = settingsChanged(settings.NewThrillLineChange(k.amount), now, ctx)
	case ChangeTimeZone:
		outcome = settingsChanged(settings.NewTimeZoneChange(k.location), now, ctx)
	case TakeOut:
		outcome = TakenOut{transfer: ledger.TakeOut(k.amount)}
	case MakeKeeper:
		h, err := ctx.Household.MakeKeeper(k.member)
		if err != nil {
			return nil, nil, fmt.Errorf("like proposal %s: %w", p.id, err)
		}
		outcome = HouseholdChanged{household: h}
	case RemoveKeeper:
		h, err := ctx.Household.RemoveKeeper(k.member)
		if err != nil {
			return nil, nil, fmt.Errorf("like proposal %s: %w", p.id, err)
		}
		outcome = HouseholdChanged{household: h}
	}
	if s, ok := outcome.(SettingsChanged); ok {
		effectiveAt = s.entry.EffectiveAt()
	}

	switch a.details.route.(type) {
	case like.SelfLike:
		return NewSelfLiked(now, liker, effectiveAt), outcome, nil
	case like.ByKeepers:
		return NewLiked(now, liker, effectiveAt), outcome, nil
	}
	panic("proposal: unknown route")
}

// Pass は、passer の見送りを決める。じぶんでいいねのルートのおねがいは見送れない（出した家族は、とりやめを使う）。
func Pass(p Proposal, passer household.MemberID, pass like.ProposalPass, now time.Time) (Passed, error) {
	a, ok := p.state.(AwaitingLike)
	if !ok {
		return Passed{}, fmt.Errorf("pass proposal %s: %w", p.id, ErrNotAwaitingLike)
	}
	route, ok := a.details.route.(like.ByKeepers)
	if !ok || !route.CanLike(passer) {
		return Passed{}, fmt.Errorf("pass proposal %s: %w", p.id, ErrNotLiker)
	}
	return NewPassed(now, passer, pass), nil
}

// Withdraw は、出した家族 by のとりやめを決める。とりやめられるのは、いいね待ちのおねがいである。
func Withdraw(p Proposal, by household.MemberID, now time.Time) (Withdrawn, error) {
	a, ok := p.state.(AwaitingLike)
	if !ok {
		return Withdrawn{}, fmt.Errorf("withdraw proposal %s: %w", p.id, ErrNotAwaitingLike)
	}
	if a.details.proposer != by {
		return Withdrawn{}, fmt.Errorf("withdraw proposal %s: %w", p.id, ErrNotProposer)
	}
	return NewWithdrawn(now, by), nil
}

func canLike(route like.Route, liker household.MemberID) bool {
	switch r := route.(type) {
	case like.ByKeepers:
		return r.CanLike(liker)
	case like.SelfLike:
		return r.Member() == liker
	}
	panic("proposal: unknown route")
}

func settingsChanged(change settings.Change, now time.Time, ctx Context) SettingsChanged {
	at := settings.EffectiveAt(change, now, ctx.Settings.Location())
	return SettingsChanged{entry: settings.NewEntry(at, change)}
}
