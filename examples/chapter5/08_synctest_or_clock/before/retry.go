// Package retry 重試呼叫外部服務：每次嘗試有自己的逾時，失敗之後等一段退避時間再試。
package retry

import (
	"context"
	"fmt"
	"time"
)

// Policy 是重試的規則。
type Policy struct {
	Attempts int           // 最多嘗試幾次；小於 1 時當成 1
	Timeout  time.Duration // 每次嘗試的逾時
	Backoff  time.Duration // 第一次失敗後等多久；之後每次加倍
}

// ANCHOR: do

// Do 呼叫 fn，失敗就依 p 退避後重試。ctx 結束時立刻放棄，不等退避時間走完。
// 全部失敗時回傳包著最後一次錯誤的錯誤。
func Do(ctx context.Context, p Policy, fn func(context.Context) error) error {
	attempts := max(p.Attempts, 1)
	var err error
	for i := range attempts {
		if i > 0 {
			select {
			case <-time.After(p.Backoff << (i - 1)):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		actx, cancel := context.WithTimeout(ctx, p.Timeout)
		err = fn(actx)
		cancel()
		if err == nil {
			return nil
		}
	}
	return fmt.Errorf("after %d attempts: %w", attempts, err)
}

// ANCHOR_END: do
