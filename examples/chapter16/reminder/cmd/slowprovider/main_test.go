package main

import "testing"

func TestCheckLoopback(t *testing.T) {
	t.Parallel()
	tests := []struct {
		addr string
		ok   bool
	}{
		{"127.0.0.1:9000", true},
		{"localhost:9000", true},
		{"[::1]:9000", true},
		{"0.0.0.0:9000", false},
		{":9000", false}, // 沒寫主機＝所有網卡
		{"192.168.1.10:9000", false},
		{"127.0.0.1", false}, // 沒有 port
	}
	for _, tt := range tests {
		if err := checkLoopback(tt.addr); (err == nil) != tt.ok {
			t.Errorf("checkLoopback(%q) error = %v, want ok = %t", tt.addr, err, tt.ok)
		}
	}
}
