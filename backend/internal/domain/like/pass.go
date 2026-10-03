package like

import (
	"errors"
	"fmt"
	"slices"
)

var (
	// ErrNoPassReason は、見送りの理由が 1 つも選ばれていないときのエラー。
	ErrNoPassReason = errors.New("no pass reason")
	// ErrCommentRequired は、「その他」を選んだのにコメントがないときのエラー。
	ErrCommentRequired = errors.New("comment is required when other is selected")
	// ErrUnknownPassReason は、見送りの理由にないコードが渡されたときのエラー。
	ErrUnknownPassReason = errors.New("unknown pass reason")
)

// ConsultationPassReason は、相談の見送りの理由（docs/domain/like.md）。
type ConsultationPassReason int

// 相談の見送りの理由。
const (
	ConsultationOverBudget ConsultationPassReason = iota + 1
	ConsultationNotNeededNow
	ConsultationAlreadyHave
	ConsultationCheaperAlternative
	ConsultationWaitForBetterTiming
	ConsultationTooExpensive
	ConsultationRarelyUsed
	ConsultationDiscussTogether
	ConsultationOther
)

var consultationPassReasonCodes = map[ConsultationPassReason]string{
	ConsultationOverBudget:          "over_budget",
	ConsultationNotNeededNow:        "not_needed_now",
	ConsultationAlreadyHave:         "already_have",
	ConsultationCheaperAlternative:  "cheaper_alternative",
	ConsultationWaitForBetterTiming: "wait_for_better_timing",
	ConsultationTooExpensive:        "too_expensive",
	ConsultationRarelyUsed:          "rarely_used",
	ConsultationDiscussTogether:     "discuss_together",
	ConsultationOther:               "other",
}

// Code は、理由のコードを返す。
func (r ConsultationPassReason) Code() string { return consultationPassReasonCodes[r] }

func (r ConsultationPassReason) isOther() bool { return r == ConsultationOther }

// ParseConsultationPassReason は、コードに対応する相談の見送りの理由を返す。
func ParseConsultationPassReason(code string) (ConsultationPassReason, error) {
	return parseReason(consultationPassReasonCodes, code)
}

// ProposalPassReason は、おねがいの見送りの理由（docs/domain/like.md）。
type ProposalPassReason int

// おねがいの見送りの理由。
const (
	ProposalWalletCannotCover ProposalPassReason = iota + 1
	ProposalTooBigChange
	ProposalNotTheRightTime
	ProposalNeedMoreDetail
	ProposalDiscussTogether
	ProposalOther
)

var proposalPassReasonCodes = map[ProposalPassReason]string{
	ProposalWalletCannotCover: "wallet_cannot_cover",
	ProposalTooBigChange:      "too_big_change",
	ProposalNotTheRightTime:   "not_the_right_time",
	ProposalNeedMoreDetail:    "need_more_detail",
	ProposalDiscussTogether:   "discuss_together",
	ProposalOther:             "other",
}

// Code は、理由のコードを返す。
func (r ProposalPassReason) Code() string { return proposalPassReasonCodes[r] }

func (r ProposalPassReason) isOther() bool { return r == ProposalOther }

// ParseProposalPassReason は、コードに対応するおねがいの見送りの理由を返す。
func ParseProposalPassReason(code string) (ProposalPassReason, error) {
	return parseReason(proposalPassReasonCodes, code)
}

// passReason は、見送りの理由の型が満たす制約。
type passReason interface {
	~int
	isOther() bool
}

// Pass は、見送りの理由とコメント。理由は 1 つ以上あり、「その他」を選んだときは
// コメントがある。相談とおねがいで、理由の型が異なる。
type Pass[R passReason] struct {
	reasons []R
	comment string
}

// ConsultationPass は、相談の見送り。
type ConsultationPass = Pass[ConsultationPassReason]

// ProposalPass は、おねがいの見送り。
type ProposalPass = Pass[ProposalPassReason]

// NewPass は、見送りを返す。理由が 1 つもないときは ErrNoPassReason、「その他」を
// 選んだのにコメントが空のときは ErrCommentRequired を返す。重複した理由は 1 つにまとめる。
func NewPass[R passReason](reasons []R, comment string) (Pass[R], error) {
	if len(reasons) == 0 {
		return Pass[R]{}, fmt.Errorf("new pass: %w", ErrNoPassReason)
	}
	sorted := slices.Clone(reasons)
	slices.Sort(sorted)
	sorted = slices.Compact(sorted)
	if slices.ContainsFunc(sorted, R.isOther) && comment == "" {
		return Pass[R]{}, fmt.Errorf("new pass: %w", ErrCommentRequired)
	}
	return Pass[R]{reasons: sorted, comment: comment}, nil
}

// Reasons は、見送りの理由を返す。
func (p Pass[R]) Reasons() []R { return slices.Clone(p.reasons) }

// Comment は、補足のコメントを返す。
func (p Pass[R]) Comment() string { return p.comment }

func parseReason[R comparable](codes map[R]string, code string) (R, error) {
	for r, c := range codes {
		if c == code {
			return r, nil
		}
	}
	var zero R
	return zero, fmt.Errorf("pass reason %q: %w", code, ErrUnknownPassReason)
}
