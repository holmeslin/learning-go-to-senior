// Package ratelimit 是自製的限流：用 time.Sleep 拉開間隔，或每秒歸零的計數器。
package ratelimit

import (
	"sync"
	"time"
)

// ANCHOR: limiters

// SleepLimiter 在每次呼叫前睡一個固定的間隔，想把速率壓在每秒 n 個以內。
type SleepLimiter struct{ interval time.Duration }

// NewSleepLimiter 回傳每秒 perSecond 個的 SleepLimiter。
func NewSleepLimiter(perSecond int) *SleepLimiter {
	return &SleepLimiter{interval: time.Second / time.Duration(perSecond)}
}

// Wait 睡一個間隔再回來；呼叫端接著去呼叫下游。
func (l *SleepLimiter) Wait() { time.Sleep(l.interval) }

// WindowCounter 每個整數秒把計數歸零，一個視窗內最多放行 limit 個。
type WindowCounter struct {
	mu     sync.Mutex
	limit  int
	window int64 // 目前的視窗：Unix 時間的秒
	count  int   // 這個視窗已經放行幾個
}

// NewWindowCounter 回傳每秒最多放行 limit 個的 WindowCounter。
func NewWindowCounter(limit int) *WindowCounter { return &WindowCounter{limit: limit} }

// Allow 回報這個呼叫是否放行。
func (c *WindowCounter) Allow() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if now := time.Now().Unix(); now != c.window {
		c.window, c.count = now, 0
	}
	if c.count >= c.limit {
		return false
	}
	c.count++
	return true
}

// ANCHOR_END: limiters
