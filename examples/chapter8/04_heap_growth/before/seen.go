package ingest

import "errors"

// ANCHOR: seen

// IDLen 是通知開頭請求代號的位元組數。
const IDLen = 16

// ErrShortPayload 表示通知比請求代號還短。
var ErrShortPayload = errors.New("payload shorter than request id")

// Seen 記住收過的請求代號，供稽核與重送比對。
type Seen struct {
	ids [][]byte
}

// Add 記住 payload 開頭的請求代號。
func (s *Seen) Add(payload []byte) error {
	if len(payload) < IDLen {
		return ErrShortPayload
	}
	s.ids = append(s.ids, payload[:IDLen])
	return nil
}

// ANCHOR_END: seen

// IDs 回傳目前記住的全部請求代號。
func (s *Seen) IDs() [][]byte {
	return s.ids
}
