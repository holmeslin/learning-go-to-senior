package preview_test

import (
	"context"
	"net"
	"net/http"
	"net/netip"
	"testing"

	preview "github.com/holmeslin/learning-go-to-senior/examples/chapter11/07_ssrf_outbound_requests/after"
)

// ANCHOR: fakedns

// fakeResolver 回傳一個把所有主機名稱都解析成 127.0.0.1 的 *net.Resolver。
// 它在 loopback 的 UDP port 上自己當 DNS 伺服器，不會問真的 DNS。
func fakeResolver(t *testing.T) *net.Resolver {
	t.Helper()
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pc.Close() })
	go func() {
		buf := make([]byte, 512)
		for {
			n, from, err := pc.ReadFrom(buf)
			if err != nil {
				return
			}
			pc.WriteTo(answer(buf[:n]), from)
		}
	}()
	return &net.Resolver{PreferGo: true, Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return new(net.Dialer).DialContext(ctx, "udp", pc.LocalAddr().String())
	}}
}

// answer 把 DNS 查詢改成回應：A 查詢回 127.0.0.1，其他類型回空的答案。
func answer(q []byte) []byte {
	end := 12
	for q[end] != 0 {
		end += int(q[end]) + 1
	}
	end += 5 // 結尾的 0，加上 QTYPE 與 QCLASS
	resp := append([]byte(nil), q[:end]...)
	resp[2], resp[3] = 0x81, 0x80 // QR、RD、RA，RCODE 0
	resp[10], resp[11] = 0, 0     // 丟掉 EDNS 的附加記錄
	if q[end-4] == 0 && q[end-3] == 1 {
		resp[7] = 1 // ANCOUNT
		resp = append(resp, 0xc0, 0x0c, 0, 1, 0, 1, 0, 0, 0, 0, 0, 4, 127, 0, 0, 1)
	}
	return resp
}

// ANCHOR_END: fakedns

// ANCHOR: dns

func TestNameResolvingToLoopbackBlocked(t *testing.T) {
	srv, hits := serve(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("internal")) })
	_, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
	url := "http://files.example:" + port // 看起來是公開的名稱
	r := fakeResolver(t)

	// 對照組：放行那個位址就連得上，證明名稱確實解析成功。
	body, err := (&preview.Fetcher{Allow: only(srv), Resolver: r}).Fetch(context.Background(), url)
	if err != nil || string(body) != "internal" {
		t.Fatalf("with the address allowed: Fetch = %q, %v; want internal, nil", body, err)
	}

	hits.Store(0)
	_, err = (&preview.Fetcher{Resolver: r}).Fetch(context.Background(), url)
	if ap, ok := blocked(err); !ok || ap.Addr() != netip.MustParseAddr("127.0.0.1") {
		t.Fatalf("err = %v, want a *BlockedError for 127.0.0.1", err)
	}
	if n := hits.Load(); n != 0 {
		t.Errorf("server saw %d requests, want 0", n)
	}
}

// ANCHOR_END: dns
