package profile_test

import (
	"errors"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	profile "github.com/holmeslin/learning-go-to-senior/examples/chapter2/02_wrap_or_expose/after"
)

// ANCHOR: test

func TestLoadHidesUnderlyingError(t *testing.T) {
	_, err := profile.Load(fstest.MapFS{}, "alice")

	if !errors.Is(err, profile.ErrNotFound) {
		t.Errorf("errors.Is(err, ErrNotFound) = false; err = %v", err)
	}
	if errors.Is(err, fs.ErrNotExist) {
		t.Error("errors.Is(err, fs.ErrNotExist) = true; the cause should not leak")
	}
	if pathErr, ok := errors.AsType[*fs.PathError](err); ok {
		t.Errorf("errors.AsType found *fs.PathError %v; the cause should not leak", pathErr)
	}
}

// ANCHOR_END: test

// deniedFS 讓每次開檔都失敗，模擬「找不到」以外的錯誤。
type deniedFS struct{}

func (deniedFS) Open(name string) (fs.File, error) {
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrPermission}
}

func TestLoadOtherErrorsKeepTextButNotChain(t *testing.T) {
	_, err := profile.Load(deniedFS{}, "alice")

	if err == nil {
		t.Fatal("Load() error = nil; want error")
	}
	if errors.Is(err, profile.ErrNotFound) {
		t.Error("errors.Is(err, ErrNotFound) = true; permission errors are not not-found")
	}
	if errors.Is(err, fs.ErrPermission) {
		t.Error("errors.Is(err, fs.ErrPermission) = true; the cause should not leak")
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("err = %q; want the cause text kept for the log", err)
	}
}

func TestLoadFound(t *testing.T) {
	fsys := fstest.MapFS{"alice.json": {Data: []byte(`{"theme":"dark"}`)}}

	got, err := profile.Load(fsys, "alice")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if want := `{"theme":"dark"}`; got != want {
		t.Errorf("Load() = %q; want %q", got, want)
	}
}
