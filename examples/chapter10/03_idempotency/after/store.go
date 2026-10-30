package payment

import (
	"bytes"
	"crypto/sha256"
	"io"
	"net/http"
	"sync"
)

// ANCHOR: store

// Store 記錄每個冪等鍵的第一次結果。這是記憶體版，只適合示範：
// 沒有保存期限、重啟就消失、多個副本之間不共用。正式環境用資料庫的唯一限制，見本集取捨。
type Store struct {
	mu      sync.Mutex
	entries map[string]*entry
}

// NewStore 建立一個空的 Store。
func NewStore() *Store { return &Store{entries: map[string]*entry{}} }

// entry 是一個鍵的狀態：done 關閉之前，第一個請求還在執行；關閉之後，resp 就是定案的結果。
type entry struct {
	fingerprint [sha256.Size]byte // 第一次請求內容的雜湊，用來抓出「同鍵不同內容」
	done        chan struct{}
	resp        recorded
}

// recorded 是記下來的回應，同時實作 http.ResponseWriter 用來接 next 的輸出。
// 它的方法在檔案後段。
type recorded struct {
	status int
	header http.Header
	body   bytes.Buffer
}

// claim 回傳鍵對應的 entry；first 為 true 代表這個請求是第一個，必須負責執行。
// 查詢與登記在同一把鎖裡完成，所以同時進來的請求只有一個會拿到 first。
func (s *Store) claim(key string, fp [sha256.Size]byte) (e *entry, first bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e, ok := s.entries[key]; ok {
		return e, false
	}
	e = &entry{fingerprint: fp, done: make(chan struct{})}
	s.entries[key] = e
	return e, true
}

// ANCHOR_END: store

// ANCHOR: wrap

// Wrap 回傳加上冪等處理的 handler。請求必須帶 Idempotency-Key：
// 第一個帶這個鍵的請求交給 next 執行並記下結果；之後同鍵同內容的請求（包含同時進來的）
// 不再執行，等第一個完成後拿到同一份回應，並帶 Idempotent-Replayed: true；
// 同鍵但內容不同回 422，沒有鍵回 400。
func (s *Store) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("Idempotency-Key")
		if key == "" {
			http.Error(w, "missing Idempotency-Key", http.StatusBadRequest)
			return
		}
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
		if err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))

		fp := sha256.Sum256(body)
		e, first := s.claim(key, fp)
		if e.fingerprint != fp {
			http.Error(w, "idempotency key reused with a different request", http.StatusUnprocessableEntity)
			return
		}
		if first {
			e.run(next, r)
		} else {
			select {
			case <-e.done:
			case <-r.Context().Done():
				return // 等的人自己放棄了，不影響別人
			}
		}
		e.resp.replay(w, !first)
	})
}

// run 執行 next 並記下回應，最後一定關閉 done，讓等待的請求醒來。
func (e *entry) run(next http.Handler, r *http.Request) {
	defer close(e.done)
	defer func() {
		if e.resp.status == 0 { // next 在寫出回應之前就 panic 了
			e.resp.status = http.StatusInternalServerError
			e.resp.body.WriteString("internal error\n")
		}
	}()
	next.ServeHTTP(&e.resp, r)
}

// ANCHOR_END: wrap

func (c *recorded) Header() http.Header {
	if c.header == nil {
		c.header = http.Header{}
	}
	return c.header
}

func (c *recorded) WriteHeader(code int) {
	if c.status == 0 {
		c.status = code
	}
}

func (c *recorded) Write(b []byte) (int, error) {
	c.WriteHeader(http.StatusOK)
	return c.body.Write(b)
}

// replay 把記下來的回應原樣寫給 w；replayed 為 true 時加上 Idempotent-Replayed 標頭。
func (c *recorded) replay(w http.ResponseWriter, replayed bool) {
	for k, v := range c.header {
		w.Header()[k] = v
	}
	if replayed {
		w.Header().Set("Idempotent-Replayed", "true")
	}
	w.WriteHeader(c.status)
	w.Write(c.body.Bytes())
}
