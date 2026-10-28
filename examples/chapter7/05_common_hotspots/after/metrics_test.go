package accesslog_test

import (
	"sync"
	"testing"

	accesslog "github.com/holmeslin/learning-go-to-senior/examples/chapter7/05_common_hotspots/after"
	old "github.com/holmeslin/learning-go-to-senior/examples/chapter7/05_common_hotspots/before"
)

// 改寫前後的計數必須一致。
func TestObserveMatchesOld(t *testing.T) {
	var m accesslog.Metrics
	var o old.Metrics
	for _, path := range []string{"/Home", "/home", "/api", "/home"} {
		m.Observe("GET", path)
		o.Observe("GET", path)
	}
	m.Observe("POST", "/home")
	o.Observe("POST", "/home")

	for _, key := range []string{"GET /Home", "GET /home", "GET /api", "POST /home", "GET /none"} {
		if got, want := m.Count(key), o.Count(key); got != want {
			t.Errorf("Count(%q) = %d, want %d", key, got, want)
		}
	}
	if got, want := m.Count("GET /home"), 2; got != want {
		t.Errorf("Count(GET /home) = %d, want %d", got, want)
	}
}

// 多個 goroutine 同時寫入與讀取；少了鎖，這個測試在 -race 下會失敗。
func TestObserveConcurrent(t *testing.T) {
	const workers, perWorker = 8, 500
	var m accesslog.Metrics

	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for range perWorker {
				m.Observe("GET", "/home")
				_ = m.Count("GET /home")
			}
		})
	}
	wg.Wait()

	if got, want := m.Count("GET /home"), workers*perWorker; got != want {
		t.Errorf("Count(GET /home) = %d, want %d", got, want)
	}
}
