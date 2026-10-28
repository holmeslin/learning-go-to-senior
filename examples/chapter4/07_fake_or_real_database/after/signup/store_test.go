package signup_test

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter4/07_fake_or_real_database/after/signup"
)

// ANCHOR: db

// newSQLStore 給每個測試一個全新的資料庫：暫存目錄裡的一個檔案，套用和正式環境同一份 schema。
func newSQLStore(t *testing.T) *signup.SQLStore {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Errorf("close database: %v", closeErr)
		}
	})
	if _, err = db.ExecContext(t.Context(), signup.Schema); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	return signup.NewSQLStore(db)
}

// ANCHOR_END: db

// ANCHOR: contract

// testStore 是 signup.Store 的契約：任何實作，不論是 fake 還是真的資料庫，都要通過。
func testStore(t *testing.T, newStore func(t *testing.T) signup.Store) {
	t.Run("assigns distinct ids", func(t *testing.T) {
		store := newStore(t)
		ann, errAnn := store.Add(t.Context(), "ann@example.com")
		bob, errBob := store.Add(t.Context(), "bob@example.com")
		if errAnn != nil || errBob != nil {
			t.Fatalf("Add() errors = %v, %v", errAnn, errBob)
		}
		if ann.ID == bob.ID {
			t.Errorf("both users got ID %d", ann.ID)
		}
	})

	t.Run("rejects duplicate email", func(t *testing.T) {
		store := newStore(t)
		if _, err := store.Add(t.Context(), "ann@example.com"); err != nil {
			t.Fatalf("first Add() error = %v", err)
		}
		if _, err := store.Add(t.Context(), "ann@example.com"); !errors.Is(err, signup.ErrEmailTaken) {
			t.Errorf("second Add() error = %v, want ErrEmailTaken", err)
		}
	})

	t.Run("ignores case in email", func(t *testing.T) {
		store := newStore(t)
		if _, err := store.Add(t.Context(), "ann@example.com"); err != nil {
			t.Fatalf("first Add() error = %v", err)
		}
		if _, err := store.Add(t.Context(), "Ann@Example.com"); !errors.Is(err, signup.ErrEmailTaken) {
			t.Errorf("second Add() error = %v, want ErrEmailTaken", err)
		}
	})
}

func TestFakeStore(t *testing.T) {
	t.Parallel()
	testStore(t, func(*testing.T) signup.Store { return newFakeStore() })
}

func TestSQLStore(t *testing.T) {
	t.Parallel()
	testStore(t, func(t *testing.T) signup.Store { return newSQLStore(t) })
}

// ANCHOR_END: contract
