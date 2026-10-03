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

// Role は、家族の役割。
type Role int

const (
	// RoleKeeper は、おさいふ係。いいねできる。
	RoleKeeper Role = iota + 1
	// RoleNonKeeper は、おさいふ係ではない家族。
	RoleNonKeeper
)

// Status は、家族のようす。
type Status int

const (
	// StatusActive は、おうちに参加している。
	StatusActive Status = iota + 1
	// StatusLeft は、おうちから外れた。過去の記録からは見える。
	StatusLeft
)

// Member は、おうちに属する家族。
type Member struct {
	id     MemberID
	role   Role
	status Status
}

// ID は、家族 ID を返す。
func (m Member) ID() MemberID { return m.id }

// Role は、家族の役割を返す。
func (m Member) Role() Role { return m.role }

// Status は、家族のようすを返す。
func (m Member) Status() Status { return m.status }

// IsActiveKeeper は、参加中のおさいふ係かどうかを返す。
func (m Member) IsActiveKeeper() bool {
	return m.status == StatusActive && m.role == RoleKeeper
}
