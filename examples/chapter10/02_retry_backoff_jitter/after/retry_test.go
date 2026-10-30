package rates_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"syscall"
	"testing"
	"testing/synctest"
	"time"

	rates "github.com/holmeslin/learning-go-to-senior/examples/chapter10/02_retry_backoff_jitter/after"
)

// ANCHOR: fake

const target = "http://rates.test/usd"

// reply 決定假下游對第 call 次呼叫的回應；回傳錯誤代表連線層就失敗了。
type reply func(call int, w http.ResponseWriter, r *http.Request) error

// downstream 是假的下游：直接當 http.Client 的 Transport，不開 port，
// 並記下每次呼叫距離 start 的假時間。
type downstream struct {
	mu    sync.Mutex
	start time.Time
	at    []time.Duration
	reply reply
}

func newDownstream(r reply) *downstream { return &downstream{start: time.Now(), reply: r} }

func (d *downstream) RoundTrip(r *http.Request) (*http.Response, error) {
	d.mu.Lock()
	call := len(d.at)
	d.at = append(d.at, time.Since(d.start))
	d.mu.Unlock()
	rec := httptest.NewRecorder()
	if err := d.reply(call, rec, r); err != nil {
		return nil, err
	}
	return rec.Result(), nil
}

// status 固定回 code；retryAfter 不是空字串時帶 Retry-After 標頭。
func status(code int, retryAfter string) reply {
	return func(_ int, w http.ResponseWriter, _ *http.Request) error {
		if retryAfter != "" {
			w.Header().Set("Retry-After", retryAfter)
		}
		w.WriteHeader(code)
		_, err := io.WriteString(w, "ok")
		return err
	}
}

// failFirst 讓前 n 次呼叫走 bad，之後回 200。
func failFirst(n int, bad reply) reply {
	return func(call int, w http.ResponseWriter, r *http.Request) error {
		if call < n {
			return bad(call, w, r)
		}
		return status(http.StatusOK, "")(call, w, r)
	}
}

// sequence 讓第 i 次呼叫走 steps[i]，用完之後一直用最後一個。
func sequence(steps ...reply) reply {
	return func(call int, w http.ResponseWriter, r *http.Request) error {
		return steps[min(call, len(steps)-1)](call, w, r)
	}
}

// hang 模擬卡住的下游，refused 模擬連線被拒。
func hang(_ int, _ http.ResponseWriter, r *http.Request) error {
	<-r.Context().Done()
	return r.Context().Err()
}

func refused(int, http.ResponseWriter, *http.Request) error {
	return fmt.Errorf("dial tcp: %w", syscall.ECONNREFUSED)
}

// half 是測試用的 jitter：永遠取上限的一半，讓等待時間是確定值。
func half(limit time.Duration) time.Duration { return limit / 2 }

func client(d *downstream, p rates.Policy) *rates.Client {
	return &rates.Client{HTTP: &http.Client{Transport: d}, Policy: p}
}

// ANCHOR_END: fake

// ANCHOR: backoff

func TestBackoffDoublesUpToMax(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := newDownstream(failFirst(5, status(http.StatusServiceUnavailable, "")))
		c := client(d, rates.Policy{MaxAttempts: 6, Base: time.Second, Max: 3 * time.Second, Jitter: half})

		if _, err := c.Fetch(context.Background(), target); err != nil {
			t.Fatalf("Fetch() = %v, want success on the 6th attempt", err)
		}

		// 等待的上限依序是 1s、2s、3s、3s、3s（第三次起被 Max 截斷），jitter 取一半。
		want := []time.Duration{0, 500 * time.Millisecond, 1500 * time.Millisecond, 3 * time.Second, 4500 * time.Millisecond, 6 * time.Second}
		if !slices.Equal(d.at, want) {
			t.Errorf("calls at %v, want %v", d.at, want)
		}
	})
}

func TestJitterStaysWithinBound(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := rand.New(rand.NewPCG(1, 2))
		jitter := func(limit time.Duration) time.Duration { return time.Duration(r.Int64N(int64(limit))) }
		d := newDownstream(failFirst(4, status(http.StatusServiceUnavailable, "")))
		c := client(d, rates.Policy{MaxAttempts: 5, Base: time.Second, Max: 4 * time.Second, Jitter: jitter})

		if _, err := c.Fetch(context.Background(), target); err != nil {
			t.Fatal(err)
		}
		for i, limit := range []time.Duration{1, 2, 4, 4} {
			if gap := d.at[i+1] - d.at[i]; gap < 0 || gap >= limit*time.Second {
				t.Errorf("wait before retry %d = %v, want [0, %v)", i+1, gap, limit*time.Second)
			}
		}
	})
}

// ANCHOR_END: backoff

// ANCHOR: classify

func TestRetriesTimeoutAndConnectionRefused(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := newDownstream(sequence(hang, refused, status(http.StatusOK, "")))
		c := client(d, rates.Policy{MaxAttempts: 4, Timeout: 2 * time.Second, Base: time.Second, Jitter: half})

		if _, err := c.Fetch(context.Background(), target); err != nil {
			t.Fatalf("Fetch() = %v, want success on the 3rd attempt", err)
		}

		// 第一次卡到 2s 的單次逾時；等 0.5s 後第二次被拒；再等 1s 後成功。
		want := []time.Duration{0, 2500 * time.Millisecond, 3500 * time.Millisecond}
		if !slices.Equal(d.at, want) {
			t.Errorf("calls at %v, want %v", d.at, want)
		}
	})
}

func TestDoesNotRetryBadRequest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := newDownstream(status(http.StatusBadRequest, ""))
		c := client(d, rates.Policy{MaxAttempts: 5, Base: time.Second})

		start := time.Now()
		_, err := c.Fetch(context.Background(), target)

		if se, ok := errors.AsType[*rates.StatusError](err); !ok || se.Code != http.StatusBadRequest {
			t.Errorf("Fetch() error = %v, want *StatusError with 400", err)
		}
		if len(d.at) != 1 || time.Since(start) != 0 {
			t.Errorf("400 was called %d times over %v, want once with no wait", len(d.at), time.Since(start))
		}
	})
}

// ANCHOR_END: classify

// ANCHOR: wait

func TestHonorsRetryAfter(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := newDownstream(failFirst(1, status(http.StatusServiceUnavailable, "3")))
		c := client(d, rates.Policy{MaxAttempts: 3, Base: time.Second, Jitter: half})

		if _, err := c.Fetch(context.Background(), target); err != nil {
			t.Fatal(err)
		}

		// Retry-After 的 3s 是下限，jitter 加在上面：3s + 0.5s。
		if want := []time.Duration{0, 3500 * time.Millisecond}; !slices.Equal(d.at, want) {
			t.Errorf("calls at %v, want %v", d.at, want)
		}
	})
}

func TestStopsWhenDeadlineIsTooClose(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := newDownstream(status(http.StatusServiceUnavailable, "10"))
		c := client(d, rates.Policy{MaxAttempts: 5, Base: time.Second, Jitter: half})
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		start := time.Now()
		_, err := c.Fetch(ctx, target)

		// Retry-After 要等 10s，deadline 只剩 5s：不如現在就回錯誤，讓呼叫端有時間做別的事。
		if se, ok := errors.AsType[*rates.StatusError](err); !ok || se.Code != http.StatusServiceUnavailable {
			t.Errorf("Fetch() error = %v, want *StatusError with 503", err)
		}
		if len(d.at) != 1 || time.Since(start) != 0 {
			t.Errorf("%d calls over %v, want 1 call and no wait", len(d.at), time.Since(start))
		}
	})
}

func TestStopsWhenContextEndsDuringWait(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := newDownstream(status(http.StatusServiceUnavailable, "10"))
		c := client(d, rates.Policy{MaxAttempts: 5, Base: time.Second, Jitter: half})
		ctx, cancel := context.WithCancel(context.Background())
		time.AfterFunc(time.Second, cancel)

		start := time.Now()
		_, err := c.Fetch(ctx, target)

		if !errors.Is(err, context.Canceled) || time.Since(start) != time.Second {
			t.Errorf("Fetch() = %v after %v, want context.Canceled after exactly 1s", err, time.Since(start))
		}
	})
}

// ANCHOR_END: wait

// ANCHOR: spread

const clients = 10

// retryPeak 讓 clients 個 client 同時遇到第一輪全失敗，回傳重試時最多有幾次呼叫落在同一個時間點。
func retryPeak(t *testing.T, jitterFor func(client int) func(time.Duration) time.Duration) int {
	peak := 0
	synctest.Test(t, func(t *testing.T) {
		d := newDownstream(failFirst(clients, status(http.StatusServiceUnavailable, "")))
		var wg sync.WaitGroup
		for i := range clients {
			c := client(d, rates.Policy{MaxAttempts: 2, Base: time.Second, Jitter: jitterFor(i)})
			wg.Go(func() { _, _ = c.Fetch(context.Background(), target) })
		}
		wg.Wait()

		perInstant := map[time.Duration]int{}
		for _, at := range d.at[clients:] {
			perInstant[at]++
			peak = max(peak, perInstant[at])
		}
	})
	return peak
}

func TestJitterSpreadsRetries(t *testing.T) {
	noJitter := func(int) func(time.Duration) time.Duration {
		return func(limit time.Duration) time.Duration { return limit }
	}
	seeded := func(i int) func(time.Duration) time.Duration {
		r := rand.New(rand.NewPCG(uint64(i), 1)) // 每個 client 一個 *rand.Rand，各自固定 seed
		return func(limit time.Duration) time.Duration { return time.Duration(r.Int64N(int64(limit))) }
	}

	if got := retryPeak(t, noJitter); got != clients {
		t.Errorf("without jitter, %d retries at the same instant, want all %d", got, clients)
	}
	if got := retryPeak(t, seeded); got != 1 {
		t.Errorf("with jitter, %d retries at the same instant, want 1", got)
	}
}

// ANCHOR_END: spread

func TestGivesUpAfterMaxAttempts(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := newDownstream(status(http.StatusServiceUnavailable, ""))
		c := client(d, rates.Policy{MaxAttempts: 3, Base: time.Second, Jitter: half})

		_, err := c.Fetch(context.Background(), target)

		if se, ok := errors.AsType[*rates.StatusError](err); !ok || se.Code != http.StatusServiceUnavailable {
			t.Errorf("Fetch() error = %v, want *StatusError with 503", err)
		}
		if len(d.at) != 3 {
			t.Errorf("%d calls, want 3", len(d.at))
		}
	})
}
