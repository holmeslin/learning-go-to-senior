// Package crawl 依序抓取一批網址，把結果交給呼叫端；呼叫端取消之後，抓取會停下來。
package crawl

import "context"

// Fetch 抓取 url 的內容。會阻塞的實作要自己留意 ctx。
type Fetch func(ctx context.Context, url string) string

// ANCHOR: pages

// Pages 依序抓取 urls，把內容送進回傳的 channel。
// 全部抓完或 ctx 結束時關閉 channel；關閉後呼叫端用 ctx.Err() 分辨是哪一種。
func Pages(ctx context.Context, urls []string, fetch Fetch) <-chan string {
	out := make(chan string)
	go func() {
		defer close(out)
		for _, u := range urls {
			if ctx.Err() != nil { // 每一輪開始前先看一眼
				return
			}
			page := fetch(ctx, u)
			select { // 送出會阻塞，所以要和 Done 一起等
			case out <- page:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

// ANCHOR_END: pages

// CheckEvery 是 Sum 每處理幾個元素檢查一次 ctx。
const CheckEvery = 1000

// ANCHOR: sum

// Sum 把 values 加總。迴圈不會阻塞，沒有人能叫醒它，所以每 CheckEvery 個元素
// 自己看一次 ctx；取消之後最多還會多算 CheckEvery-1 個元素。
func Sum(ctx context.Context, values []int) (int, error) {
	total := 0
	for i, v := range values {
		if i%CheckEvery == 0 {
			if err := ctx.Err(); err != nil {
				return 0, err
			}
		}
		total += v
	}
	return total, nil
}

// ANCHOR_END: sum
