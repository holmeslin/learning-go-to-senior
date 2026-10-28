// Package ipfilter 在一批請求位址裡濾掉封鎖名單內的位址。
package ipfilter

// ANCHOR: filter

// Filter 回傳 ips 裡不在 blocklist 的位址，保留原本的順序。
// 先把名單放進集合，每個位址只查一次集合。
func Filter(ips, blocklist []string) []string {
	blocked := make(map[string]struct{}, len(blocklist))
	for _, b := range blocklist {
		blocked[b] = struct{}{}
	}
	out := make([]string, 0, len(ips))
	for _, ip := range ips {
		if _, ok := blocked[ip]; !ok {
			out = append(out, ip)
		}
	}
	return out
}

// ANCHOR_END: filter

// FilterLinear 和 Filter 的結果相同，是改寫前的版本：每個位址都掃過整份名單。
// 保留它是為了讓測試與 benchmark 能對照新舊兩個版本。
func FilterLinear(ips, blocklist []string) []string {
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
