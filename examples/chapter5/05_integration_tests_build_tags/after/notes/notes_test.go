package notes_test

import (
	"errors"
	"testing"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter5/05_integration_tests_build_tags/after/notes"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "buy milk", "buy milk"},
		{"surrounding spaces", "  call ann \n", "call ann"},
		{"inner spaces kept", "a  b", "a  b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := notes.Normalize(tt.in)
			if err != nil || got != tt.want {
				t.Errorf("Normalize(%q) = %q, %v; want %q, nil", tt.in, got, err, tt.want)
			}
		})
	}
}

func TestNormalizeRejectsEmpty(t *testing.T) {
	for _, in := range []string{"", "   ", "\t\n"} {
		if got, err := notes.Normalize(in); !errors.Is(err, notes.ErrEmpty) {
			t.Errorf("Normalize(%q) = %q, %v; want error %v", in, got, err, notes.ErrEmpty)
		}
	}
}
