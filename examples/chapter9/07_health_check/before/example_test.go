package health_test

import (
	"fmt"
	"net/http/httptest"

	health "github.com/holmeslin/learning-go-to-senior/examples/chapter9/07_health_check/before"
)

// ANCHOR: example

// stuckDB 的 Ping 卡在 release 上，直到測試放行。
type stuckDB struct{ entered, release chan struct{} }

func (d stuckDB) Ping() error {
	d.entered <- struct{}{}
	<-d.release
	return nil
}

func Example() {
	db := stuckDB{entered: make(chan struct{}), release: make(chan struct{})}
	h := health.Handler(db)

	done := make(chan int)
	go func() {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", "/health", nil))
		done <- rec.Code
	}()

	<-db.entered // 請求已經進到 Ping，之後只有 release 能讓它回來
	select {
	case code := <-done:
		fmt.Println("responded", code)
	default:
		fmt.Println("no response while the database is stuck")
	}

	close(db.release)
	fmt.Println("responded", <-done, "once the database returned")
	// Output:
	// no response while the database is stuck
	// responded 200 once the database returned
}

// ANCHOR_END: example
