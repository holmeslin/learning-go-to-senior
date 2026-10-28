# channel 與 mutex 怎麼選

## 本集目標

保護共享狀態時，判斷該用 channel 還是 mutex：狀態用 mutex 護住，channel 留給轉移資料的擁有權與發出訊號。也知道擁有狀態的 goroutine 在什麼情況下才合理，並讀得懂缺少同步時的 `-race` 報告。

## 情境

網站的點閱計數：每個請求的 handler 呼叫 `Inc(path)`，後台頁面呼叫 `Count(path)`，多個 goroutine 同時存取同一份各路徑的計數。前作教過〈[data race 與 `-race`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter9/03_data_race.html)〉、〈[`sync.Mutex`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter9/04_mutex.html)〉、〈[channel](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter9/08_channel.html)〉；這集只談兩者怎麼選。

## 常見寫法

Go 社群常說「用通訊來共享記憶體，不要用共享記憶體來通訊」，於是計數交給一個 goroutine 擁有，其他人用請求與回應 channel 存取：

```go,ignore
{{#include ../../../examples/chapter6/07_channel_or_mutex/before/counter.go:counter}}
```

用起來沒有問題。下面的 `Example` 讓四個 goroutine 同時寫入，輸出只有與順序無關的總數：

```go,ignore
{{#include ../../../examples/chapter6/07_channel_or_mutex/before/example_test.go:example}}
```

完整程式碼在 `examples/chapter6/07_channel_or_mutex/before/`。

## 問題在哪

先看沒有任何同步會怎樣。下面是同一個 `Counter` 拿掉保護的樣子，放在 repo 外的暫存模組；測試讓兩個 goroutine 各呼叫 100 次 `Inc("/home")`，用 `go test -race` 執行：

```text
package hits

type Counter struct {
	counts map[string]int
}

func (c *Counter) Inc(path string) {
	if c.counts == nil {
		c.counts = make(map[string]int)
	}
	c.counts[path]++
}

func (c *Counter) Count(path string) int {
	return c.counts[path]
}
```

執行結果（某一次）：

```text
==================
WARNING: DATA RACE
Read at 0x... by goroutine N:
  racerepro.(*Counter).Inc()
      counter.go:8
  racerepro.TestConcurrentInc.func1()
      counter_test.go:14

Previous write at 0x... by goroutine N:
  racerepro.(*Counter).Inc()
      counter.go:9
  racerepro.TestConcurrentInc.func1()
      counter_test.go:14

Goroutine N (running) created at:
  sync.(*WaitGroup).Go()
  ...
==================
--- FAIL: TestConcurrentInc
    testing.go:1865: race detected during execution of test
FAIL
```

報告的讀法：第一段是這次存取（讀，在 `counter.go` 第 8 行），第二段是沒有任何同步就發生在它之前的另一次存取（寫，第 9 行），兩個都是 `Inc`，來自不同的 goroutine；最後列出各 goroutine 從哪裡建立的。位址、goroutine 編號、行內偏移、完整路徑、耗時、測試印的 log，以及 `sync` 與 `testing` 內部的 stack 都省略了，模組名稱也簡寫；報告不只一份，這裡只留第一份，其餘是 map 在 runtime 內部被同樣兩個 goroutine 存取的報告，數量每次可能不同。測試因此失敗。

channel 版就是為了避免這件事，代價是：

- 長度：四個 channel、一個請求型別、一個 `loop`，換來的功能和 mutex 版一樣。
- 生命週期：用完要 `Close`，漏了就洩漏一個永遠等請求的 goroutine（偵測方式本章〈goroutine 洩漏怎麼發生、怎麼在測試裡抓到〉會談）。
- 關閉之後：`Close` 之後 `Inc` 沒有人接收，永遠卡住；在 `synctest.Test` 裡重現，得到 `panic: deadlock: main bubble goroutine has exited but blocked goroutines remain`。
- 零值不能用：沒有 `New` 就沒有 channel。

## 較好的做法

保護狀態用 mutex。狀態還是只有一份，差別是由鎖來排隊，而不是由一個 goroutine 排隊：

```go,ignore
{{#include ../../../examples/chapter6/07_channel_or_mutex/after/counter.go:counter}}
```

零值可用，沒有 `Close`，沒有額外的 goroutine。`Counter` 含 `sync.Mutex`，使用後不可複製，`go vet` 的 `copylocks` 會檢查。測試放在外部套件，讓多個 goroutine 讀寫交錯，最後檢查總數；少了鎖，這個測試在 `-race` 下就是上面那份報告：

```go,ignore
{{#include ../../../examples/chapter6/07_channel_or_mutex/after/counter_test.go:concurrent}}
```

channel 留給它擅長的事：轉移資料的擁有權、發出訊號。下面的 `buf` 送出之後，送的人不再碰它，收的人成為唯一擁有者，不需要鎖；`done` 只用來通知結束：

```go
package main

import "fmt"

func main() {
	rows := make(chan []string)
	done := make(chan struct{})
	go func() {
		defer close(done)
		fmt.Println("flush", len(<-rows), "rows")
	}()

	buf := []string{"a", "b"}
	rows <- buf
	buf = nil // 送出之後不再碰
	<-done
}
```

執行結果：

```text
flush 2 rows
```

## 取捨

擁有狀態的 goroutine 有它合理的地方：

- 狀態的變化本身是一連串事件，例如狀態機、依序處理的命令；操作順序就是 channel 的順序。
- 要和其他 channel 一起 `select`，例如同時等請求、逾時與停止訊號；mutex 沒辦法跟 channel 一起等。
- 狀態的所有權本來就跟某個長期運作的元件綁在一起，這個元件有 `Run(ctx)` 之類的生命週期。

`atomic` 適合單一數值，例如總點閱數用 `atomic.Int64` 就夠，不需要 mutex，見〈[`sync/atomic`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter9/06_atomic.html)〉。一旦要同時更新兩個以上的欄位並維持它們的關係（總數與各路徑的計數要一致），就回到 mutex。

mutex 也有代價：臨界區要短，不要在鎖裡呼叫會阻塞或會呼叫外部程式碼的函式，否則其他 goroutine 全部跟著等。鎖競爭是否成為瓶頸、怎麼量，第 7 章會談。

## 重點整理

- 保護共享狀態用 mutex：程式短、零值可用、沒有生命週期要管；channel 用在轉移資料的擁有權與發出訊號。
- 用 channel 保護狀態要付出額外的 goroutine、請求型別與 `Close`，忘了關或關了還呼叫都會卡住。
- 沒有同步的共享狀態，`-race` 會列出兩次沒有同步的存取：誰讀、誰寫、各從哪個 goroutine 建立。
- 擁有狀態的 goroutine 適合事件序列，或需要和其他 channel 一起 `select` 的情況；單一數值用 `atomic`。
- 並行的測試讓多個 goroutine 讀寫交錯、最後檢查總數，並在 `-race` 下執行。
