// Package settings 載入報表服務的設定；秘密包在 Secret 裡，印出來、記進日誌或序列化成 JSON 時都會被遮蔽。
package settings

import "log/slog"

// redacted 是秘密在任何輸出裡顯示的樣子。
const redacted = "[REDACTED]"

// ANCHOR: secret

// Secret 是不該被印出來的字串。欄位是小寫，套件外只能用 Reveal 取值，
// 所以誤把它當 string 用會編譯失敗，而不是悄悄把秘密帶進輸出。
type Secret struct{ value string }

// String 讓 %v、%s 與 Println 印出遮蔽後的文字。
func (Secret) String() string { return redacted }

// GoString 讓 %#v 印出遮蔽後的文字。
func (Secret) GoString() string { return redacted }

// LogValue 讓 slog 在交給 handler 之前就把值換成遮蔽後的字串。
func (Secret) LogValue() slog.Value { return slog.StringValue(redacted) }

// MarshalJSON 讓 encoding/json 輸出遮蔽後的字串。
func (Secret) MarshalJSON() ([]byte, error) { return []byte(`"` + redacted + `"`), nil }

// Reveal 回傳真正的值，只在把秘密交給需要它的 client 時呼叫。
// 全專案只有少數幾個呼叫處，code review 時可以直接搜尋。
func (s Secret) Reveal() string { return s.value }

// ANCHOR_END: secret
