package commands_test

import (
	"strings"
	"testing"

	commands "github.com/holmeslin/learning-go-to-senior/examples/chapter5/10_flaky_tests/before"
)

func echo(args []string) string { return strings.Join(append([]string{"echo:"}, args...), " ") }

// ANCHOR: test

func TestRegister(t *testing.T) {
	if err := commands.Register("echo", echo); err != nil {
		t.Fatalf("Register(%q) error = %v, want nil", "echo", err)
	}
}

func TestRun(t *testing.T) {
	got, err := commands.Run("echo", "hi")
	if err != nil {
		t.Fatalf("Run(%q) error = %v, want nil", "echo", err)
	}
	if want := "echo: hi"; got != want {
		t.Errorf("Run(%q) = %q, want %q", "echo", got, want)
	}
}

// ANCHOR_END: test
