package orders_test

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	orders "github.com/holmeslin/learning-go-to-senior/examples/chapter4/03_query_timeout_layer/after"
)

func serve(ctx context.Context, h http.Handler) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(ctx, http.MethodGet, "/pending", nil))
	return rec
}

func TestHandlerOK(t *testing.T) {
	h := orders.NewHandler(orders.NewStore(newDB(t), time.Hour), time.Minute)

	rec := serve(t.Context(), h)
	if rec.Code != http.StatusOK || rec.Body.String() != "3\n" {
		t.Errorf("got %d %q; want 200 \"3\\n\"", rec.Code, rec.Body)
	}
}

// 邊界設的 deadline 會往下傳：資料庫忙碌時，handler 在自己的 deadline 回 504。
func TestHandlerDeadline(t *testing.T) {
	db := newDB(t)
	busy(t, db)
	h := orders.NewHandler(orders.NewStore(db, time.Hour), 100*time.Millisecond)

	if rec := serve(t.Context(), h); rec.Code != http.StatusGatewayTimeout {
		t.Errorf("code = %d; want %d", rec.Code, http.StatusGatewayTimeout)
	}
}

// 使用者已經離開（請求的 context 取消）：查詢不跑，也沒有人需要回應。
func TestHandlerClientGone(t *testing.T) {
	h := orders.NewHandler(orders.NewStore(newDB(t), time.Hour), time.Minute)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if rec := serve(ctx, h); rec.Body.Len() != 0 {
		t.Errorf("wrote %q for a canceled request; want nothing", rec.Body)
	}
}

// 其他錯誤回 500，細節只進日誌。
func TestHandlerFailure(t *testing.T) {
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	db := newDB(t)
	if _, err := db.Exec("DROP TABLE orders"); err != nil {
		t.Fatal(err)
	}
	h := orders.NewHandler(orders.NewStore(db, time.Hour), time.Minute)

	if rec := serve(t.Context(), h); rec.Code != http.StatusInternalServerError {
		t.Errorf("code = %d; want %d", rec.Code, http.StatusInternalServerError)
	}
}
