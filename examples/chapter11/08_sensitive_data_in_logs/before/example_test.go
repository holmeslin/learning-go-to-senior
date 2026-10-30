package wallet_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	wallet "github.com/holmeslin/learning-go-to-senior/examples/chapter11/08_sensitive_data_in_logs/before"
)

// ANCHOR: example

func ExampleNewHandler() {
	dropTime := func(groups []string, a slog.Attr) slog.Attr {
		if len(groups) == 0 && a.Key == slog.TimeKey {
			return slog.Attr{}
		}
		return a
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{ReplaceAttr: dropTime}))
	h := wallet.NewHandler(logger)

	body := `{"user_id":"u-42","password":"example-not-a-real-password","card_number":"1234567890123456"}`
	req := httptest.NewRequest(http.MethodPost, "/cards", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer example-not-a-real-token")
	h.ServeHTTP(httptest.NewRecorder(), req)
	// Output:
	// level=INFO msg="bind card" request="{UserID:u-42 Password:example-not-a-real-password CardNumber:1234567890123456}" headers="map[Authorization:[Bearer example-not-a-real-token] Content-Type:[application/json]]"
}

// ANCHOR_END: example
