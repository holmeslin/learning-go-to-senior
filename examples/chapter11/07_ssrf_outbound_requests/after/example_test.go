package preview_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"

	preview "github.com/holmeslin/learning-go-to-senior/examples/chapter11/07_ssrf_outbound_requests/after"
)

// ANCHOR: example

func ExampleFetcher_Fetch() {
	internal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "internal-only: admin console")
	}))
	defer internal.Close()
	// 公開的網站：任何路徑都轉址到 internal。
	public := httptest.NewServer(http.RedirectHandler(internal.URL+"/admin", http.StatusFound))
	defer public.Close()

	// 預設規則：loopback 一律擋下。
	_, err := (&preview.Fetcher{}).Fetch(context.Background(), internal.URL)
	be, _ := errors.AsType[*preview.BlockedError](err)
	fmt.Println("direct:", be.Addr.Addr())

	// 注入的允許清單只放行 public；轉址到 internal 在連線那一刻被擋下。
	f := &preview.Fetcher{MaxRedirects: 3, Allow: func(ap netip.AddrPort) bool {
		return ap.String() == public.Listener.Addr().String()
	}}
	_, err = f.Fetch(context.Background(), public.URL)
	be, _ = errors.AsType[*preview.BlockedError](err)
	fmt.Println("redirect:", be.Addr.Addr())
	// Output:
	// direct: 127.0.0.1
	// redirect: 127.0.0.1
}

// ANCHOR_END: example
