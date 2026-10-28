package ipfilter_test

import (
	"fmt"
	"testing"

	ipfilter "github.com/holmeslin/learning-go-to-senior/examples/chapter7/02_b_loop_benchstat/after"
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

func BenchmarkFilter(b *testing.B) {
	ips := addrs("10.0.0", 1000) // 準備工作放在迴圈外，不算進計時
	for _, n := range []int{4, 64, 1000} {
		blocklist := addrs("192.0.2", n) // 都不在 ips 裡：每個位址都得比完整份名單
		b.Run(fmt.Sprintf("impl=linear/blocklist=%d", n), func(b *testing.B) {
			for b.Loop() {
				ipfilter.FilterLinear(ips, blocklist)
			}
		})
		b.Run(fmt.Sprintf("impl=set/blocklist=%d", n), func(b *testing.B) {
			for b.Loop() {
				ipfilter.Filter(ips, blocklist)
			}
		})
	}
}

// ANCHOR_END: bench
