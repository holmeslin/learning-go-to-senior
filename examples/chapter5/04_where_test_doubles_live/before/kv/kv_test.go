package kv_test

import (
	"errors"
	"testing"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter5/04_where_test_doubles_live/before/kv"
)

// kv 自己的測試只測真的 DirStore。沒有任何東西檢查別人抄的 fake。
func TestDirStore(t *testing.T) {
	t.Parallel()
	s := kv.NewDirStore(t.TempDir())

	if err := s.Put("a", "one"); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	if got, err := s.Get("a"); err != nil || got != "one" {
		t.Errorf("Get() = %q, %v; want %q, nil", got, err, "one")
	}
	if err := s.Put("a/b", "x"); !errors.Is(err, kv.ErrInvalidKey) {
		t.Errorf("Put(%q) error = %v, want kv.ErrInvalidKey", "a/b", err)
	}
}
