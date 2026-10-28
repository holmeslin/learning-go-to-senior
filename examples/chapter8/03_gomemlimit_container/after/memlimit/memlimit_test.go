package memlimit_test

import (
	"errors"
	"io/fs"
	"math"
	"runtime/debug"
	"testing"
	"testing/fstest"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter8/03_gomemlimit_container/after/memlimit"
)

// ANCHOR: limit

func TestLimit(t *testing.T) {
	cases := []struct {
		name    string
		fsys    fs.FS
		want    int64
		wantErr error
	}{
		{"number", fstest.MapFS{"memory.max": {Data: []byte("1000\n")}}, 900, nil},
		{"no newline", fstest.MapFS{"memory.max": {Data: []byte("1000")}}, 900, nil},
		{"max", fstest.MapFS{"memory.max": {Data: []byte("max\n")}}, 0, memlimit.ErrNoLimit},
		{"missing file", fstest.MapFS{}, 0, fs.ErrNotExist},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := memlimit.Limit(c.fsys)
			if got != c.want || !errors.Is(err, c.wantErr) {
				t.Fatalf("Limit = %d, %v; want %d, %v", got, err, c.want, c.wantErr)
			}
		})
	}
	for _, bad := range []string{"", "abc", "12MiB", "0", "-1"} {
		fsys := fstest.MapFS{"memory.max": {Data: []byte(bad)}}
		if _, err := memlimit.Limit(fsys); err == nil || errors.Is(err, memlimit.ErrNoLimit) {
			t.Errorf("Limit(%q) error = %v, want a format error", bad, err)
		}
	}
}

// ANCHOR_END: limit

// ANCHOR: apply

// setProcessLimit 把行程的 memory limit 設成 limit，測試結束時還原。
func setProcessLimit(t *testing.T, limit int64) {
	t.Helper()
	prev := debug.SetMemoryLimit(limit)
	t.Cleanup(func() { debug.SetMemoryLimit(prev) })
}

func TestApply(t *testing.T) {
	fsys := fstest.MapFS{"memory.max": {Data: []byte("268435456\n")}}

	t.Run("applies", func(t *testing.T) {
		setProcessLimit(t, math.MaxInt64) // 當作沒有人設過
		got, err := memlimit.Apply(fsys)
		if err != nil || got != 241591911 || debug.SetMemoryLimit(-1) != got {
			t.Fatalf("Apply = %d, %v; current limit %d", got, err, debug.SetMemoryLimit(-1))
		}
	})
	t.Run("keeps an explicit limit", func(t *testing.T) {
		setProcessLimit(t, 1<<30) // 例如部署設定裡的 GOMEMLIMIT
		if _, err := memlimit.Apply(fsys); !errors.Is(err, memlimit.ErrAlreadySet) || debug.SetMemoryLimit(-1) != 1<<30 {
			t.Fatalf("Apply error = %v, current limit %d", err, debug.SetMemoryLimit(-1))
		}
	})
	t.Run("no cgroup limit", func(t *testing.T) {
		setProcessLimit(t, math.MaxInt64)
		unlimited := fstest.MapFS{"memory.max": {Data: []byte("max")}}
		if _, err := memlimit.Apply(unlimited); !errors.Is(err, memlimit.ErrNoLimit) || debug.SetMemoryLimit(-1) != math.MaxInt64 {
			t.Fatalf("Apply error = %v, current limit %d", err, debug.SetMemoryLimit(-1))
		}
	})
}

// ANCHOR_END: apply
