package push_test

import (
	"context"
	"encoding/json/v2"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/push"
)

// ANCHOR: fake

// roundTripFunc 讓一個函式充當 http.RoundTripper：測試不開 socket，供應商的行為由函式決定。
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// status 回傳一個固定狀態碼的假供應商。
func status(code int) roundTripFunc {
	return func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
	}
}

var notification = push.Notification{
	Key:         "reminder-7-42",
	DeviceToken: "device-token-1",
	EventID:     7,
	Title:       "活動提醒",
	Body:        "你報名的活動將在一天後開始。",
}

// send 用 rt 當傳輸，送出上面那則通知。
func send(rt http.RoundTripper) error {
	return push.NewClient("http://provider.test", time.Second, rt).Send(context.Background(), notification)
}

// ANCHOR_END: fake

// ANCHOR: classify

func TestSendClassifiesFailures(t *testing.T) {
	t.Parallel()
	tests := []struct {
		code      int
		permanent bool
		transient bool
	}{
		{code: http.StatusOK},
		{code: http.StatusAccepted},
		{code: http.StatusBadRequest, permanent: true},
		{code: http.StatusUnauthorized, permanent: true},
		{code: http.StatusNotFound, permanent: true},
		{code: http.StatusGone, permanent: true},
		{code: http.StatusTooManyRequests, transient: true}, // 4xx 之中只有 429 值得再試
		{code: http.StatusInternalServerError, transient: true},
		{code: http.StatusBadGateway, transient: true},
		{code: http.StatusServiceUnavailable, transient: true},
		{code: http.StatusTemporaryRedirect, permanent: true}, // 不跟著轉址：token 不能送到沒設定過的地方
	}
	for _, tt := range tests {
		t.Run(http.StatusText(tt.code), func(t *testing.T) {
			t.Parallel()
			err := send(status(tt.code))
			if got := errors.Is(err, push.ErrPermanent); got != tt.permanent {
				t.Errorf("Send() with %d: Is(ErrPermanent) = %v, want %v (error: %v)", tt.code, got, tt.permanent, err)
			}
			if got := errors.Is(err, push.ErrTransient); got != tt.transient {
				t.Errorf("Send() with %d: Is(ErrTransient) = %v, want %v (error: %v)", tt.code, got, tt.transient, err)
			}
			if !tt.permanent && !tt.transient && err != nil {
				t.Errorf("Send() with %d error = %v, want nil", tt.code, err)
			}
		})
	}
}

// ANCHOR_END: classify

func TestSendTransportErrorIsTransient(t *testing.T) {
	t.Parallel()
	errRefused := errors.New("connection refused")
	err := send(roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, errRefused }))
	if !errors.Is(err, push.ErrTransient) || !errors.Is(err, errRefused) {
		t.Errorf("Send() error = %v, want transient and wrapping the transport error", err)
	}
}

// ANCHOR: timeout

func TestSendTimesOutPerRequest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// 供應商不回應，直到請求的 context 結束；逾時只看請求自己的期限，不看呼叫端。
		hang := roundTripFunc(func(r *http.Request) (*http.Response, error) {
			<-r.Context().Done()
			return nil, r.Context().Err()
		})
		start := time.Now()
		err := push.NewClient("http://provider.test", 3*time.Second, hang).Send(t.Context(), notification)

		if !errors.Is(err, push.ErrTransient) || !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Send() error = %v, want transient and DeadlineExceeded", err)
		}
		if got := time.Since(start); got != 3*time.Second {
			t.Errorf("Send() took %v, want the 3s request timeout", got)
		}
	})
}

// ANCHOR_END: timeout

// ANCHOR: request

func TestSendRequestShape(t *testing.T) {
	t.Parallel()
	var got *http.Request
	var body []byte
	err := send(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		got = r
		body, _ = io.ReadAll(r.Body)
		return status(http.StatusOK)(r)
	}))
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if got.Method != http.MethodPost || got.URL.String() != "http://provider.test/push" {
		t.Errorf("request = %s %s, want POST http://provider.test/push", got.Method, got.URL)
	}
	// 冪等鍵帶在標頭，重試用同一個鍵，供應商才認得出是同一則。
	if key := got.Header.Get("Idempotency-Key"); key != "reminder-7-42" {
		t.Errorf("Idempotency-Key = %q, want reminder-7-42", key)
	}
	if ct := got.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("body %s is not JSON: %v", body, err)
	}
	if payload["device_token"] != "device-token-1" || payload["event_id"] != float64(7) {
		t.Errorf("body = %v, want the device token and event id", payload)
	}
}

// ANCHOR_END: request

func TestSendBadBaseURLIsPermanent(t *testing.T) {
	t.Parallel()
	err := push.NewClient("://bad", time.Second, status(http.StatusOK)).Send(context.Background(), notification)
	if !errors.Is(err, push.ErrPermanent) {
		t.Errorf("Send() error = %v, want permanent: retrying cannot fix a bad URL", err)
	}
}
