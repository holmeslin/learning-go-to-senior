package payment_test

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	payment "github.com/holmeslin/learning-go-to-senior/examples/chapter10/03_idempotency/before"
)

// ANCHOR: example

// dropFirstReply 讓第一個請求照常送達伺服器，卻把回應丟掉：
// client 看到的是逾時，無從得知伺服器有沒有扣款。
type dropFirstReply struct{ calls int }

func (d *dropFirstReply) RoundTrip(r *http.Request) (*http.Response, error) {
	resp, err := http.DefaultTransport.RoundTrip(r)
	d.calls++
	if err == nil && d.calls == 1 {
		resp.Body.Close()
		return nil, errors.New("timeout awaiting response headers")
	}
	return resp, err
}

func Example() {
	ledger := &payment.Ledger{}
	srv := httptest.NewServer(payment.Handler(ledger))
	defer srv.Close()
	client := &http.Client{Transport: &dropFirstReply{}}

	// 逾時就重試，這是本章〈重試、退避與 jitter〉談過的標準做法。
	for attempt := 1; attempt <= 2; attempt++ {
		resp, err := client.Post(srv.URL+"/charges", "application/json",
			strings.NewReader(`{"order":"A-1","cents":1200}`))
		if urlErr, ok := errors.AsType[*url.Error](err); ok {
			fmt.Printf("attempt %d: no reply (%v)\n", attempt, urlErr.Err)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("attempt %d: %d %s", attempt, resp.StatusCode, body)
	}

	for _, c := range ledger.Charges() {
		fmt.Println("ledger:", c.ID, c.Order, c.Cents)
	}
	// Output:
	// attempt 1: no reply (timeout awaiting response headers)
	// attempt 2: 201 {"id":"ch_2","order":"A-1","cents":1200}
	// ledger: ch_1 A-1 1200
	// ledger: ch_2 A-1 1200
}

// ANCHOR_END: example
