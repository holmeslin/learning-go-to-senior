package preview_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"

	preview "github.com/holmeslin/learning-go-to-senior/examples/chapter11/07_ssrf_outbound_requests/before"
)

// ANCHOR: example

// dnsClient 回傳的 client 會把 map 裡的 "主機:port" 改連到指定的位址，
// 模擬 DNS 紀錄指向內部位址；其他連線照常。
func dnsClient(records map[string]string) *http.Client {
	var d net.Dialer
	return &http.Client{Transport: &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			if to, ok := records[addr]; ok {
				addr = to
			}
			return d.DialContext(ctx, network, addr)
		},
	}}
}

func Example() {
	internal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "internal-only: admin console")
	}))
	defer internal.Close()
	// 攻擊者的網站：任何路徑都轉址到 internal。
	attacker := httptest.NewServer(http.RedirectHandler(internal.URL+"/admin", http.StatusFound))
	defer attacker.Close()

	client := dnsClient(map[string]string{
		"internal.example:80": internal.Listener.Addr().String(),
		"attacker.example:80": attacker.Listener.Addr().String(),
	})
	for _, c := range []struct{ name, url string }{
		{"loopback literal", internal.URL},
		{"name resolving to loopback", "http://internal.example/"},
		{"redirect to loopback", "http://attacker.example/"},
	} {
		body, err := preview.Fetch(client, c.url)
		if err != nil {
			fmt.Printf("%s: rejected: %v\n", c.name, err)
			continue
		}
		fmt.Printf("%s: fetched %q\n", c.name, body)
	}
	// Output:
	// loopback literal: rejected: host is not allowed
	// name resolving to loopback: fetched "internal-only: admin console"
	// redirect to loopback: fetched "internal-only: admin console"
}

// ANCHOR_END: example

// ANCHOR: limits

func Example_limits() {
	big := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(make([]byte, 4<<20))
	}))
	defer big.Close()

	client := dnsClient(map[string]string{"big.example:80": big.Listener.Addr().String()})
	body, err := preview.Fetch(client, "http://big.example/")
	fmt.Println("read", len(body), "bytes into memory:", err)
	fmt.Println("http.DefaultClient timeout:", http.DefaultClient.Timeout)
	// Output:
	// read 4194304 bytes into memory: <nil>
	// http.DefaultClient timeout: 0s
}

// ANCHOR_END: limits
