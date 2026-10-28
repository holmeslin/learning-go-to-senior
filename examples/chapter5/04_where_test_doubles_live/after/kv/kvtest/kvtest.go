// Package kvtest provides a fake for package kv and a contract test that
// any store, real or fake, must pass. It plays the same role for kv as
// net/http/httptest does for net/http and testing/fstest does for io/fs.
package kvtest

import (
	"errors"
	"sync"
	"testing"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter5/04_where_test_doubles_live/after/kv"
)

// Store is the behavior TestStore checks.
type Store interface {
	Put(key, value string) error
	Get(key string) (string, error)
}

// ANCHOR: fake

// Fake is an in-memory Store that follows the same rules as kv.DirStore.
// It is safe for concurrent use.
type Fake struct {
	mu     sync.Mutex
	values map[string]string
}

// NewFake returns an empty Fake.
func NewFake() *Fake {
	return &Fake{values: map[string]string{}}
}

// Put stores value under key. It returns kv.ErrInvalidKey if key is not valid.
func (f *Fake) Put(key, value string) error {
	if !kv.ValidKey(key) {
		return kv.ErrInvalidKey
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.values[key] = value
	return nil
}

// Get returns the value under key. It returns kv.ErrInvalidKey if key is
// not valid and kv.ErrNotFound if nothing is stored.
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

// TestStore runs the kv contract against stores made by newStore.
// newStore is called once per subtest and must return an empty store.
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
