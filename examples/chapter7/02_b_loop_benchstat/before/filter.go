// Package ipfilter 在一批請求位址裡濾掉封鎖名單內的位址。
package ipfilter

import "strings"

// ANCHOR: filter

// Filter 回傳 ips 裡不在 blocklist 的位址，保留原本的順序。
// 每個位址都掃過整份名單。
func Filter(ips, blocklist []string) []string {
	out := make([]string, 0, len(ips))
	for _, ip := range ips {
		if !contains(blocklist, ip) {
			out = append(out, ip)
		}
	}
	return out
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// ANCHOR_END: filter

// ANCHOR: internal

// IsInternal 判斷 ip 是不是內網位址（10.x.x.x 或 192.168.x.x）。
func IsInternal(ip string) bool {
	return strings.HasPrefix(ip, "10.") || strings.HasPrefix(ip, "192.168.")
}

// ANCHOR_END: internal
