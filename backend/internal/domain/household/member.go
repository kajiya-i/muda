// Package household は、おうちと家族を表す（docs/domain/household.md）
package household

import (
	"errors"
	"fmt"
)

// ErrEmptyMemberID は、空の家族 ID が渡されたときのエラー。
var ErrEmptyMemberID = errors.New("member id is empty")

// MemberID は、家族を一意に表す ID。
type MemberID struct {
	value string
}

// NewMemberID は、家族 ID を返す。空文字列なら ErrEmptyMemberID を返す。
func NewMemberID(value string) (MemberID, error) {
	if value == "" {
		return MemberID{}, fmt.Errorf("new member id: %w", ErrEmptyMemberID)
	}
	return MemberID{value: value}, nil
}

// String は、家族 ID の文字列を返す
func (id MemberID) String() string { return id.value }
