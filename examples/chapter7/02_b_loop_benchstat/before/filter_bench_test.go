package ipfilter_test

import (
	"fmt"
	"testing"

	ipfilter "github.com/holmeslin/learning-go-to-senior/examples/chapter7/02_b_loop_benchstat/before"
)

// ANCHOR: bench

// addrs 產生 n 個 prefix.N 形式的位址，例如 addrs("10.0.0", 2) 是 10.0.0.0 和 10.0.0.1。
func addrs(prefix string, n int) []string {
	s := make([]string, n)
	for i := range s {
		s[i] = fmt.Sprintf("%s.%d", prefix, i)
	}
	return s
}

func BenchmarkIsInternal(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ipfilter.IsInternal("10.1.2.3") // 結果沒有人用
	}
}

func BenchmarkFilter(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ips := addrs("10.0.0", 1000) // 準備工作也在計時範圍內
		blocklist := addrs("192.0.2", 64)
		ipfilter.Filter(ips, blocklist)
	}
}

// ANCHOR_END: bench
