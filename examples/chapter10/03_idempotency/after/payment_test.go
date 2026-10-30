package payment_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"

	payment "github.com/holmeslin/learning-go-to-senior/examples/chapter10/03_idempotency/after"
)

// ANCHOR: post

// post 對 h 送一個帶冪等鍵的 POST /charges；key 為空字串時不帶標頭。
func post(h http.Handler, key, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/charges", strings.NewReader(body))
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// ANCHOR_END: post

const a1 = `{"order":"A-1","cents":1200}`

func TestRetryReturnsFirstResult(t *testing.T) {
	ledger := &payment.Ledger{}
	h := payment.NewStore().Wrap(payment.Handler(ledger))

	first := post(h, "k1", a1)
	retry := post(h, "k1", a1)

	if n := len(ledger.Charges()); n != 1 {
		t.Errorf("charges = %d, want 1", n)
	}
	if retry.Code != first.Code || retry.Body.String() != first.Body.String() {
		t.Errorf("retry = %d %q, want the first result %d %q",
			retry.Code, retry.Body, first.Code, first.Body)
	}
	if got := first.Header().Get("Idempotent-Replayed"); got != "" {
		t.Errorf("first Idempotent-Replayed = %q, want empty", got)
	}
	if got := retry.Header().Get("Idempotent-Replayed"); got != "true" {
		t.Errorf("retry Idempotent-Replayed = %q, want true", got)
	}
	if got, want := retry.Header().Get("Content-Type"), first.Header().Get("Content-Type"); got != want {
		t.Errorf("retry Content-Type = %q, want %q", got, want)
	}
}

func TestDifferentKeysAreIndependent(t *testing.T) {
	ledger := &payment.Ledger{}
	h := payment.NewStore().Wrap(payment.Handler(ledger))

	post(h, "k1", a1)
	post(h, "k2", a1)

	if n := len(ledger.Charges()); n != 2 {
		t.Errorf("charges = %d, want 2", n)
	}
}

func TestSameKeyDifferentBodyIsRejected(t *testing.T) {
	ledger := &payment.Ledger{}
	h := payment.NewStore().Wrap(payment.Handler(ledger))

	post(h, "k1", a1)
	rec := post(h, "k1", `{"order":"A-1","cents":9900}`)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rec.Code)
	}
	if cs := ledger.Charges(); len(cs) != 1 || cs[0].Cents != 1200 {
		t.Errorf("charges = %v, want only the first charge of 1200", cs)
	}
}

func TestMissingKeyIsRejected(t *testing.T) {
	ledger := &payment.Ledger{}
	h := payment.NewStore().Wrap(payment.Handler(ledger))

	rec := post(h, "", a1)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
	if n := len(ledger.Charges()); n != 0 {
		t.Errorf("charges = %d, want 0", n)
	}
}

// ANCHOR: concurrent

func TestConcurrentDuplicatesRunOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		var executions atomic.Int32
		slow := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executions.Add(1)
			<-release // 真的阻塞，直到測試放行
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte("done"))
		})
		h := payment.NewStore().Wrap(slow)

		recs := make([]*httptest.ResponseRecorder, 3)
		var wg sync.WaitGroup
		for i := range recs {
			wg.Go(func() { recs[i] = post(h, "k1", a1) })
		}
		synctest.Wait() // 三個請求都停住了：一個卡在 handler，其餘在等它的結果

		if n := executions.Load(); n != 1 {
			t.Errorf("executions while blocked = %d, want 1", n)
		}
		close(release)
		wg.Wait()

		if n := executions.Load(); n != 1 {
			t.Errorf("executions = %d, want 1", n)
		}
		for i, rec := range recs {
			if rec.Code != http.StatusCreated || rec.Body.String() != "done" {
				t.Errorf("request %d = %d %q, want 201 \"done\"", i, rec.Code, rec.Body)
			}
		}
	})
}

// ANCHOR_END: concurrent

func TestPanickingHandlerDoesNotBlockRetries(t *testing.T) {
	var executions atomic.Int32
	h := payment.NewStore().Wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		executions.Add(1)
		panic("boom")
	}))

	func() {
		defer func() { _ = recover() }() // net/http 會接住 panic；這裡自己接
		post(h, "k1", a1)
	}()
	rec := post(h, "k1", a1) // 若鍵一直處於「執行中」，這一行會永遠卡住

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	if n := executions.Load(); n != 1 {
		t.Errorf("executions = %d, want 1", n)
	}
}
