// Package passwd 把密碼雜湊成可以存進資料庫的金鑰。
package passwd

import (
	"crypto/pbkdf2"
	"crypto/sha256"
)

// KeyLen 是雜湊結果的長度，單位是位元組。
const KeyLen = 32

// ANCHOR: hasher

// Hasher 用 PBKDF2-HMAC-SHA256 雜湊密碼。
// 所有雜湊都交給同一個 goroutine 做，避免登入尖峰把所有核心吃滿。
type Hasher struct {
	iter int
	jobs chan job
}

type job struct {
	password string
	salt     []byte
	reply    chan result
}

type result struct {
	key []byte
	err error
}

// New 回傳迭代 iter 次的 Hasher，並啟動負責雜湊的 goroutine；用完要呼叫 Close。
func New(iter int) *Hasher {
	h := &Hasher{iter: iter, jobs: make(chan job)}
	go func() {
		for j := range h.jobs {
			key, err := pbkdf2.Key(sha256.New, j.password, j.salt, h.iter, KeyLen)
			j.reply <- result{key, err}
		}
	}()
	return h
}

// Hash 把密碼與鹽雜湊成長度 KeyLen 的金鑰，可以同時被多個 goroutine 呼叫。
func (h *Hasher) Hash(password string, salt []byte) ([]byte, error) {
	reply := make(chan result, 1)
	h.jobs <- job{password, salt, reply}
	r := <-reply
	return r.key, r.err
}

// Close 讓負責雜湊的 goroutine 結束；Close 之後不可再呼叫 Hash。
func (h *Hasher) Close() {
	close(h.jobs)
}

// ANCHOR_END: hasher
