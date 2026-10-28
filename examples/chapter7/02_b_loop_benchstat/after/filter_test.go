package ipfilter_test

import (
	"fmt"
	"slices"
	"testing"

	ipfilter "github.com/holmeslin/learning-go-to-senior/examples/chapter7/02_b_loop_benchstat/after"
)

func ExampleFilter() {
	ips := []string{"10.0.0.1", "203.0.113.7", "10.0.0.2", "198.51.100.9"}
	blocklist := []string{"203.0.113.7", "198.51.100.9"}
	fmt.Println(ipfilter.Filter(ips, blocklist))
	// Output: [10.0.0.1 10.0.0.2]
}

// sink 接住結果，讓 AllocsPerRun 量到的函式呼叫不會被當成沒人用的程式碼。
var sink []string

func TestFilterMatchesLinear(t *testing.T) {
	tests := []struct {
		name      string
		ips       []string
		blocklist []string
		want      []string
	}{
		{"沒有輸入", nil, []string{"a"}, nil},
		{"沒有名單", []string{"a", "b"}, nil, []string{"a", "b"}},
		{"全部擋下", []string{"a", "b"}, []string{"b", "a"}, nil},
		{"部分擋下，保留順序", []string{"c", "a", "b", "a"}, []string{"a"}, []string{"c", "b"}},
		{"名單有重複", []string{"a", "b"}, []string{"a", "a"}, []string{"b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for name, filter := range map[string]func(ips, blocklist []string) []string{
				"Filter":       ipfilter.Filter,
				"FilterLinear": ipfilter.FilterLinear,
			} {
				if got := filter(tt.ips, tt.blocklist); !slices.Equal(got, tt.want) {
					t.Errorf("%s(%v, %v) = %v, want %v", name, tt.ips, tt.blocklist, got, tt.want)
				}
			}
		})
	}
}

// ANCHOR: allocs

func TestFilterAllocs(t *testing.T) {
	blocklist := addrs("192.0.2", 64)
	small, large := addrs("10.0.0", 10), addrs("10.0.0", 1000)

	// 舊版只配置輸出的 slice 一次。
	if got := testing.AllocsPerRun(100, func() { sink = ipfilter.FilterLinear(large, blocklist) }); got != 1 {
		t.Errorf("FilterLinear allocs = %v, want 1", got)
	}
	// 新版多了集合的配置，但次數不隨輸入的數量成長。
	allocsSmall := testing.AllocsPerRun(100, func() { sink = ipfilter.Filter(small, blocklist) })
	allocsLarge := testing.AllocsPerRun(100, func() { sink = ipfilter.Filter(large, blocklist) })
	if allocsSmall != allocsLarge {
		t.Errorf("Filter allocs: %d 個輸入 = %v, %d 個輸入 = %v, want 相同", len(small), allocsSmall, len(large), allocsLarge)
	}
}

// ANCHOR_END: allocs
