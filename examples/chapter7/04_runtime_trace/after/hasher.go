// Package passwd 把密碼雜湊成可以存進資料庫的金鑰。
package passwd

import (
	"crypto/pbkdf2"
	"crypto/sha256"
)

// KeyLen 是雜湊結果的長度，單位是位元組。
const KeyLen = 32

// ANCHOR: hasher

// Hasher 用 PBKDF2-HMAC-SHA256 雜湊密碼，可以同時被多個 goroutine 使用。
type Hasher struct {
	iter int
}

// New 回傳迭代 iter 次的 Hasher；次數越多，破解越貴，每次雜湊也越慢。
func New(iter int) *Hasher {
	return &Hasher{iter: iter}
}

// Hash 把密碼與鹽雜湊成長度 KeyLen 的金鑰。
func (h *Hasher) Hash(password string, salt []byte) ([]byte, error) {
	return pbkdf2.Key(sha256.New, password, salt, h.iter, KeyLen)
}

// ANCHOR_END: hasher
