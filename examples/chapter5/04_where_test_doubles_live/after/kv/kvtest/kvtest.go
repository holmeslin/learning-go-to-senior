// Package kvtest 提供 kv 套件的 fake，以及任何 store（真的或 fake）都要通過的契約測試。
// 它之於 kv，就像 net/http/httptest 之於 net/http、testing/fstest 之於 io/fs。
package kvtest

import (
	"errors"
	"sync"
	"testing"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter5/04_where_test_doubles_live/after/kv"
)

// Store 是 TestStore 檢查的行為；它只是 TestStore 的參數型別，使用端不需要 import 它。
type Store interface {
	Put(key, value string) error
	Get(key string) (string, error)
}

// ANCHOR: fake

// Fake 是放在記憶體裡的 Store，規則和 kv.DirStore 一樣。
// 可以同時被多個 goroutine 使用。
type Fake struct {
	mu     sync.Mutex
	values map[string]string
}

// NewFake 回傳一個空的 Fake。
func NewFake() *Fake {
	return &Fake{values: map[string]string{}}
}

// Put 把 value 存在 key 底下。key 不合規定時回傳 kv.ErrInvalidKey。
func (f *Fake) Put(key, value string) error {
	if !kv.ValidKey(key) {
		return kv.ErrInvalidKey
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.values[key] = value
	return nil
}

// Get 回傳 key 底下的值。key 不合規定時回傳 kv.ErrInvalidKey，
// 沒有值時回傳 kv.ErrNotFound。
func (f *Fake) Get(key string) (string, error) {
	if !kv.ValidKey(key) {
		return "", kv.ErrInvalidKey
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.values[key]
	if !ok {
		return "", kv.ErrNotFound
	}
	return v, nil
}

// ANCHOR_END: fake

// ANCHOR: testfunc

// TestStore 對 newStore 造出來的 store 執行 kv 的契約。
// 每個子測試都會呼叫一次 newStore，它必須回傳空的 store。
func TestStore(t *testing.T, newStore func(t *testing.T) Store) {
	t.Run("get returns what put stored", func(t *testing.T) {
		s := newStore(t)
		if err := s.Put("a", "one"); err != nil {
			t.Fatalf("Put() error = %v", err)
		}
		if got, err := s.Get("a"); err != nil || got != "one" {
			t.Errorf("Get() = %q, %v; want %q, nil", got, err, "one")
		}
	})

	t.Run("put replaces the previous value", func(t *testing.T) {
		s := newStore(t)
		for _, v := range []string{"one", "two"} {
			if err := s.Put("a", v); err != nil {
				t.Fatalf("Put(%q) error = %v", v, err)
			}
		}
		if got, err := s.Get("a"); err != nil || got != "two" {
			t.Errorf("Get() = %q, %v; want %q, nil", got, err, "two")
		}
	})

	t.Run("get of a missing key is ErrNotFound", func(t *testing.T) {
		if _, err := newStore(t).Get("missing"); !errors.Is(err, kv.ErrNotFound) {
			t.Errorf("Get() error = %v, want kv.ErrNotFound", err)
		}
	})

	for _, key := range []string{"", "..", "a/b", "/a"} {
		t.Run("invalid key "+key, func(t *testing.T) {
			s := newStore(t)
			if err := s.Put(key, "x"); !errors.Is(err, kv.ErrInvalidKey) {
				t.Errorf("Put(%q) error = %v, want kv.ErrInvalidKey", key, err)
			}
			if _, err := s.Get(key); !errors.Is(err, kv.ErrInvalidKey) {
				t.Errorf("Get(%q) error = %v, want kv.ErrInvalidKey", key, err)
			}
		})
	}
}

// ANCHOR_END: testfunc
