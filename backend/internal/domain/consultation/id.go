// Package consultation は、相談を表す（docs/domain/consultation.md）。
package consultation

import (
	"errors"
	"fmt"
)

// ErrEmptyID は、空の相談 ID が渡されたときのエラー。
var ErrEmptyID = errors.New("consultation id is empty")

// ID は、相談を一意に表す ID。
type ID struct {
	value string
}

// NewID は、相談 ID を返す。空文字列なら ErrEmptyID を返す。
func NewID(value string) (ID, error) {
	if value == "" {
		return ID{}, fmt.Errorf("new consultation id: %w", ErrEmptyID)
	}
	return ID{value: value}, nil
}

// String は、相談 ID の文字列を返す。
func (id ID) String() string { return id.value }
