package like_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/kajiya-i/muda/backend/internal/domain/like"
)

func TestNewPass(t *testing.T) {
	tests := []struct {
		name    string
		reasons []like.ConsultationPassReason
		comment string
		want    []like.ConsultationPassReason
		wantErr error
	}{
		{
			name:    "one reason",
			reasons: []like.ConsultationPassReason{like.ConsultationCheaperAlternative},
			want:    []like.ConsultationPassReason{like.ConsultationCheaperAlternative},
		},
		{
			name:    "duplicated reasons are merged",
			reasons: []like.ConsultationPassReason{like.ConsultationRarelyUsed, like.ConsultationTooExpensive, like.ConsultationRarelyUsed},
			want:    []like.ConsultationPassReason{like.ConsultationTooExpensive, like.ConsultationRarelyUsed},
		},
		{
			name:    "no reason",
			wantErr: like.ErrNoPassReason,
		},
		{
			name:    "other without comment",
			reasons: []like.ConsultationPassReason{like.ConsultationOther},
			wantErr: like.ErrCommentRequired,
		},
		{
			name:    "other with comment",
			reasons: []like.ConsultationPassReason{like.ConsultationOther},
			comment: "来月の旅行のために節約したい",
			want:    []like.ConsultationPassReason{like.ConsultationOther},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := like.NewPass(tt.reasons, tt.comment)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewPass error = %v, want %v", err, tt.wantErr)
			}
			if err == nil && !slices.Equal(got.Reasons(), tt.want) {
				t.Errorf("Reasons = %v, want %v", got.Reasons(), tt.want)
			}
		})
	}
}

func TestNewProposalPassRequiresCommentForOther(t *testing.T) {
	if _, err := like.NewPass([]like.ProposalPassReason{like.ProposalOther}, ""); !errors.Is(err, like.ErrCommentRequired) {
		t.Errorf("NewPass(proposal other) error = %v, want %v", err, like.ErrCommentRequired)
	}
}

func TestPassReasonCodesRoundTrip(t *testing.T) {
	for r := like.ConsultationOverBudget; r <= like.ConsultationOther; r++ {
		got, err := like.ParseConsultationPassReason(r.Code())
		if err != nil || got != r {
			t.Errorf("ParseConsultationPassReason(%q) = %v, %v; want %v", r.Code(), got, err, r)
		}
	}
	for r := like.ProposalWalletCannotCover; r <= like.ProposalOther; r++ {
		got, err := like.ParseProposalPassReason(r.Code())
		if err != nil || got != r {
			t.Errorf("ParseProposalPassReason(%q) = %v, %v; want %v", r.Code(), got, err, r)
		}
	}
	if _, err := like.ParseConsultationPassReason("wallet_cannot_cover"); !errors.Is(err, like.ErrUnknownPassReason) {
		t.Errorf("proposal code parsed as consultation reason: error = %v, want %v", err, like.ErrUnknownPassReason)
	}
}
