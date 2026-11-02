package signup

// Notifier 由呼叫端提供，用來通知被遞補的人。實作自己處理失敗（重試或記錄），
// 不影響遞補的結果。
//
// Event 不持有鎖就呼叫 Notify，所以實作可以很慢，也可以回頭呼叫同一個 Event。
type Notifier interface {
	// Notify 通知 who：他已從候補名單遞補為正式名單。
	Notify(who string)
}
