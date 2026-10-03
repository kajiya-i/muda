package purpose_test

import (
	"errors"
	"testing"

	"github.com/kajiya-i/muda/backend/internal/domain/purpose"
)

func TestBuiltinCodesRoundTrip(t *testing.T) {
	all := []purpose.Builtin{
		purpose.BuiltinFood, purpose.BuiltinDailyGoods, purpose.BuiltinClothing,
		purpose.BuiltinTransport, purpose.BuiltinHealth, purpose.BuiltinEducation,
		purpose.BuiltinHobby, purpose.BuiltinSocial, purpose.BuiltinHome, purpose.BuiltinOther,
	}
	seen := map[string]bool{}
	for _, b := range all {
		code := b.Code()
		if code == "" {
			t.Errorf("Builtin(%d).Code() is empty", b)
		}
		if seen[code] {
			t.Errorf("code %q is duplicated", code)
		}
		seen[code] = true

		got, err := purpose.ParseBuiltin(code)
		if err != nil {
			t.Fatal(err)
		}
		if got != b {
			t.Errorf("ParseBuiltin(%q) = %d, want %d", code, got, b)
		}
	}
}

func TestParseBuiltinUnknown(t *testing.T) {
	if _, err := purpose.ParseBuiltin("travel"); !errors.Is(err, purpose.ErrUnknownBuiltin) {
		t.Errorf("ParseBuiltin(unknown) error = %v, want %v", err, purpose.ErrUnknownBuiltin)
	}
}

func TestNewCustomID(t *testing.T) {
	if _, err := purpose.NewCustomID(""); !errors.Is(err, purpose.ErrEmptyCustomID) {
		t.Errorf("NewCustomID(\"\") error = %v, want %v", err, purpose.ErrEmptyCustomID)
	}
}
