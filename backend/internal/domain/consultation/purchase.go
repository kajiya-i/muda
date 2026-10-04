package consultation

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/kajiya-i/muda/backend/internal/domain/household"
	"github.com/kajiya-i/muda/backend/internal/domain/ledger"
	"github.com/kajiya-i/muda/backend/internal/domain/money"
)

var (
	// ErrNotLiked は、いいね済みではない相談に買ったよ報告をしようとしたときのエラー。
	ErrNotLiked = errors.New("consultation is not liked")
	// ErrReportExpired は、報告の期限を過ぎた相談に買ったよ報告をしようとしたときのエラー。
	ErrReportExpired = errors.New("report deadline has passed")
	// ErrOverConsultedAmount は、買った金額が相談した金額を超えているときのエラー。
	ErrOverConsultedAmount = errors.New("purchased amount exceeds the consulted amount")
	// ErrUnknownPaymentMethod は、払い方が分からないときのエラー。
	ErrUnknownPaymentMethod = errors.New("unknown payment method")
	// ErrNothingToRepay は、おかえしする買い物が 1 つもないときのエラー。
	ErrNothingToRepay = errors.New("nothing to repay")
	// ErrNotRepayable は、たてかえで買ったよになっていない相談をおかえししようとしたときのエラー。
	ErrNotRepayable = errors.New("consultation is not an unrepaid advance")
	// ErrDuplicateConsultation は、同じ相談を 1 回のおかえしに 2 回含めたときのエラー。
	ErrDuplicateConsultation = errors.New("consultation is included twice")
)

// ReportPurchase は、相談した家族 by の買ったよ報告を決める。あわせて、払い方に応じた台帳の移動を返す。
//
//   - おさいふから払ったときは、おさいふ → 支出
//   - たてかえで払ったときは、おかえし待ち（相談した家族） → 支出
//
// 買った金額は、相談した金額以下でなければならない。報告の期限を過ぎていたら報告できない。
// 買った日は受け取らない。正直に報告する限り、買ったのは必ずいいねがそろった月のうちだからである
// （docs/domain/consultation.md）。
func ReportPurchase(c Consultation, by household.MemberID, amount money.PositiveMoney,
	method PaymentMethod, now time.Time,
) (PurchaseReported, ledger.Transfer, error) {
	liked, ok := c.state.(LikedState)
	if !ok {
		return PurchaseReported{}, ledger.Transfer{}, fmt.Errorf("report purchase of %s: %w", c.id, ErrNotLiked)
	}
	d := liked.details
	if d.requester != by {
		return PurchaseReported{}, ledger.Transfer{}, fmt.Errorf("report purchase of %s: %w", c.id, ErrNotRequester)
	}
	if liked.deadlines.ReportExpired(now) {
		return PurchaseReported{}, ledger.Transfer{}, fmt.Errorf("report purchase of %s: %w", c.id, ErrReportExpired)
	}
	cmp, err := amount.Money().Compare(d.amount.Money())
	if err != nil {
		return PurchaseReported{}, ledger.Transfer{}, fmt.Errorf("report purchase of %s: %w", c.id, err)
	}
	if cmp > 0 {
		return PurchaseReported{}, ledger.Transfer{}, fmt.Errorf("report purchase of %s: %w", c.id, ErrOverConsultedAmount)
	}

	var transfer ledger.Transfer
	switch method {
	case PaidFromWallet:
		transfer = ledger.WalletPayment(amount, d.purpose)
	case PaidByAdvance:
		transfer = ledger.Advance(d.requester, amount, d.purpose)
	default:
		return PurchaseReported{}, ledger.Transfer{}, fmt.Errorf("report purchase of %s: %w", c.id, ErrUnknownPaymentMethod)
	}
	return NewPurchaseReported(now, by, amount, method), transfer, nil
}

// Repaid は、おかえしした 1 つの買い物について、起きるイベントと台帳の移動。
type Repaid struct {
	consultation ID
	event        AdvanceRepaid
	transfer     ledger.Transfer
}

// Consultation は、おかえしした相談の ID を返す。
func (r Repaid) Consultation() ID { return r.consultation }

// Event は、おかえしが済んだイベントを返す。
func (r Repaid) Event() AdvanceRepaid { return r.event }

// Transfer は、おさいふ → おかえし待ちの台帳の移動を返す。
func (r Repaid) Transfer() ledger.Transfer { return r.transfer }

// Repay は、おさいふ係 by が、家族 member のたてかえで買った買い物 cs をまとめておかえしすることを決める。
// 買い物ごとに、おかえしが済んだイベントと、おさいふ → おかえし待ちの台帳の移動を返す。
//
// 1 つの買い物は全額をおかえしする。おかえしは実際にお金を渡した事実の記録なので、
// おうちのおさいふのお金が足りなくても記録する（docs/domain/wallet.md）。
func Repay(cs []Consultation, member household.MemberID, h household.Household,
	by household.MemberID, repayment ledger.RepaymentID, now time.Time,
) ([]Repaid, error) {
	keeper, ok := h.Member(by)
	if !ok || !keeper.IsActiveKeeper() {
		return nil, fmt.Errorf("repay %s: %w", repayment, household.ErrNotKeeper)
	}
	if len(cs) == 0 {
		return nil, fmt.Errorf("repay %s: %w", repayment, ErrNothingToRepay)
	}

	results := make([]Repaid, 0, len(cs))
	seen := make([]ID, 0, len(cs))
	for _, c := range cs {
		if slices.Contains(seen, c.id) {
			return nil, fmt.Errorf("repay %s with %s: %w", repayment, c.id, ErrDuplicateConsultation)
		}
		seen = append(seen, c.id)

		p, ok := c.state.(PurchasedState)
		if !ok || p.method != PaidByAdvance {
			return nil, fmt.Errorf("repay %s with %s: %w", repayment, c.id, ErrNotRepayable)
		}
		if p.details.requester != member {
			return nil, fmt.Errorf("repay %s with %s: %w", repayment, c.id, ErrNotRequester)
		}
		results = append(results, Repaid{
			consultation: c.id,
			event:        NewAdvanceRepaid(now, by, repayment),
			transfer:     ledger.Repayment(member, p.amount),
		})
	}
	return results, nil
}
