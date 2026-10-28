package commands_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	commands "github.com/holmeslin/learning-go-to-senior/examples/chapter5/10_flaky_tests/after"
)

// echo 是唯讀的測試資料：不會被任何測試修改，可以放心共用。
func echo(args []string) string { return strings.Join(append([]string{"echo:"}, args...), " ") }

// ANCHOR: test

func TestRegister(t *testing.T) {
	r := commands.New()

	if err := r.Register("echo", echo); err != nil {
		t.Fatalf("Register(%q) error = %v, want nil", "echo", err)
	}
	err := r.Register("echo", echo)
	if !errors.Is(err, commands.ErrDuplicate) {
		t.Errorf("Register(%q) twice error = %v, want %v", "echo", err, commands.ErrDuplicate)
	}
}

func TestRun(t *testing.T) {
	r := commands.New()
	if err := r.Register("echo", echo); err != nil {
		t.Fatalf("Register(%q) error = %v, want nil", "echo", err)
	}

	got, err := r.Run("echo", "hi")
	if err != nil {
		t.Fatalf("Run(%q) error = %v, want nil", "echo", err)
	}
	if want := "echo: hi"; got != want {
		t.Errorf("Run(%q) = %q, want %q", "echo", got, want)
	}
}

// ANCHOR_END: test

func TestRunWithoutArgs(t *testing.T) {
	r := commands.New()
	if err := r.Register("echo", echo); err != nil {
		t.Fatalf("Register(%q) error = %v, want nil", "echo", err)
	}

	got, err := r.Run("echo")
	if err != nil || got != "echo:" {
		t.Errorf("Run(%q) = %q, %v; want %q, nil", "echo", got, err, "echo:")
	}
}

func TestRunUnknown(t *testing.T) {
	_, err := commands.New().Run("echo", "hi")
	if !errors.Is(err, commands.ErrUnknown) {
		t.Errorf("Run(%q) on empty registry error = %v, want %v", "echo", err, commands.ErrUnknown)
	}
}

func TestNames(t *testing.T) {
	r := commands.New()
	for _, name := range []string{"stop", "echo", "start"} {
		if err := r.Register(name, echo); err != nil {
			t.Fatalf("Register(%q) error = %v, want nil", name, err)
		}
	}

	got := r.Names()
	want := []string{"echo", "start", "stop"}
	if !slices.Equal(got, want) {
		t.Errorf("Names() = %v, want %v", got, want)
	}
}
