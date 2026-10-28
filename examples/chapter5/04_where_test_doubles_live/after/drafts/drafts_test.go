package drafts_test

import (
	"errors"
	"testing"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter5/04_where_test_doubles_live/after/drafts"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter5/04_where_test_doubles_live/after/kv"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter5/04_where_test_doubles_live/after/kv/kvtest"
)

func TestRestoreReturnsAutosavedText(t *testing.T) {
	t.Parallel()
	store := kvtest.NewFake()
	if err := drafts.Autosave(store, "plan", "outline"); err != nil {
		t.Fatalf("Autosave() error = %v", err)
	}

	got, err := drafts.Restore(store, "plan")
	if err != nil || got != "outline" {
		t.Errorf("Restore() = %q, %v; want %q, nil", got, err, "outline")
	}
}

func TestRestoreWithoutDraftIsEmpty(t *testing.T) {
	t.Parallel()
	got, err := drafts.Restore(kvtest.NewFake(), "plan")
	if err != nil || got != "" {
		t.Errorf("Restore() = %q, %v; want empty string, nil", got, err)
	}
}

func TestAutosaveRejectsInvalidTitle(t *testing.T) {
	t.Parallel()
	err := drafts.Autosave(kvtest.NewFake(), "Q3/plan", "outline")
	if !errors.Is(err, kv.ErrInvalidKey) {
		t.Errorf("Autosave() error = %v, want kv.ErrInvalidKey", err)
	}
}

// ANCHOR: stub

// failingStore 每次都回同一個錯誤。這種回答 fake 給不出來：
// 磁碟壞掉不是 kv 契約的一部分，所以放在使用端的測試裡。
type failingStore struct{ err error }

func (s failingStore) Put(string, string) error   { return s.err }
func (s failingStore) Get(string) (string, error) { return "", s.err }

func TestRestoreReportsStoreFailure(t *testing.T) {
	t.Parallel()
	errDisk := errors.New("disk failure")

	got, err := drafts.Restore(failingStore{err: errDisk}, "plan")

	if !errors.Is(err, errDisk) {
		t.Errorf("Restore() error = %v, want %v", err, errDisk)
	}
	if got != "" {
		t.Errorf("Restore() = %q, want empty string", got)
	}
}

// ANCHOR_END: stub
