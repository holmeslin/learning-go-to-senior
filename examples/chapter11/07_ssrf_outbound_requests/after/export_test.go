package preview

import "net/netip"

// Allowed 讓外部測試套件直接測連線檢查的判斷，不必真的連線。
func Allowed(f *Fetcher, ap netip.AddrPort) bool { return f.allowed(ap) }
