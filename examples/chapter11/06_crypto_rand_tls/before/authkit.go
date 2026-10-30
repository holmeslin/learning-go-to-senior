// Package authkit 是刻意示範的漏洞，不要照用：
// 用 math/rand/v2 產生重設碼、用 == 比對 token、client 關掉憑證驗證。
// 可以參考的寫法在 after 套件。
package authkit

import (
	"crypto/tls"
	"fmt"
	"math/rand/v2"
	"net/http"
	"time"
)

// ANCHOR: token

// CodeGenerator 用 math/rand/v2 的 PCG 產生密碼重設碼，種子是服務啟動時間的秒數。
type CodeGenerator struct{ r *rand.Rand }

// NewCodeGenerator 回傳以 start 為種子的 CodeGenerator；服務啟動時傳入 time.Now()。
func NewCodeGenerator(start time.Time) *CodeGenerator {
	return &CodeGenerator{r: rand.New(rand.NewPCG(uint64(start.Unix()), 0))}
}

// Next 回傳下一組重設碼：8 個十六進位字元。
func (g *CodeGenerator) Next() string { return fmt.Sprintf("%08x", g.r.Uint32()) }

// SameToken 回報使用者送來的 token 是否等於資料庫裡的那一個。
func SameToken(got, want string) bool { return got == want }

// ANCHOR_END: token

// ANCHOR: client

// NewClient 回傳關掉憑證驗證的 http.Client，連自簽憑證的內部服務時「就能動了」。
func NewClient() *http.Client {
	return &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}}
}

// ANCHOR_END: client
