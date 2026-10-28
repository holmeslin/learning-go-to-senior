package fetch_test

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	fetch "github.com/holmeslin/learning-go-to-senior/examples/chapter6/02_errgroup/before"
)

// ANCHOR: overwrite

// TestFetchAllKeepsLastError 把錯誤的行為鎖住：a 先失敗，回傳的卻是後失敗的 b。
func TestFetchAllKeepsLastError(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		delay := map[string]time.Duration{"a": 10 * time.Millisecond, "b": 20 * time.Millisecond}
		get := func(ctx context.Context, src string) error {
			time.Sleep(delay[src])
			return errors.New("fetch " + src + ": failed")
		}

		err := fetch.FetchAll(t.Context(), []string{"a", "b"}, get)
		if got, want := err.Error(), "fetch b: failed"; got != want {
			t.Errorf("FetchAll() = %q, want %q", got, want)
		}
	})
}

// ANCHOR_END: overwrite
