# goroutine 洩漏怎麼發生、怎麼在測試裡抓到

## 本集目標

認得 goroutine 洩漏最常見的幾種長相，並在測試裡用 `synctest.Test` 抓到：bubble 結束時還有 goroutine 卡住，測試就失敗，不必等正式環境的 goroutine 數量上升才發現。

## 情境

報價服務同時問好幾個供應商，先回來的先用：`Fastest` 拿到第一筆就回傳。洩漏是什麼，前作教過：〈[goroutine 洩漏](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter9/16_goroutine_leak.html)〉。`synctest` 的基本規則見第 5 章〈[測時間與並行：用 `testing/synctest`，還是自己注入時鐘](../chapter5/08_synctest_or_clock.md)〉。本章〈[取消與 `context` 的傳遞](04_cancellation_context.md)〉談過取消要傳進每個會阻塞的地方；這集談的是，沒做到的時候怎麼在測試裡抓到。

## 常見寫法

每個供應商一個 goroutine，結果送進同一條 channel，函式只收第一筆：

```go,ignore
{{#include ../../../examples/chapter6/09_goroutine_leak/before/quote.go:fastest}}
```

完整程式碼在 `examples/chapter6/09_goroutine_leak/before/`。測試很自然：兩個供應商，一個馬上回、一個卡著，斷言拿到快的那個。

## 問題在哪

這個測試會過。測試的程式如下，沒有放進 repo，因為它會讓測試結束時留下一個卡住的 goroutine，所以在 repo 外的暫存模組重現：

```text
func TestFastest(t *testing.T) {
	block := make(chan struct{}) // 永遠不關閉：慢的供應商一直卡著
	fast := func() string { return "fast" }
	slow := func() string { <-block; return "slow" }

	got := quote.Fastest([]quote.Source{slow, fast})
	if want := "fast"; got != want {
		t.Errorf("Fastest() = %q, want %q", got, want)
	}
}
```

用 `go test -race` 執行，輸出省略了耗時。

執行結果：

```text
ok  	quoterepro
```

慢的那個供應商之後回來，要把結果送進沒人再收的無緩衝 channel，永遠送不出去。每呼叫一次 `Fastest` 就多一個這樣的 goroutine，連同它參照的一切都不會被回收；普通的測試看不到，只會在正式環境慢慢變成記憶體與 goroutine 數量。

換成 `synctest.Test` 就看得到。下面這個測試屬於 after（見下一段），這裡對 before 的實作執行；輸出省略了耗時、goroutine 編號、位址、檔案路徑與 stack 中間各行。

執行結果：

```text
--- FAIL: TestFastestLeavesNoGoroutine
panic: deadlock: main bubble goroutine has exited but blocked goroutines remain [recovered, repanicked]
...
goroutine N [chan send (durable), synctest bubble 1]:
quoterepro.Fastest.func1()
FAIL	quoterepro
```

`goroutine N [chan send (durable)...]` 指出卡在哪：`Fastest` 裡那行送出。是 panic，不是一般的測試失敗，所以同一個套件裡之後的測試也不會跑；before 的洩漏只在這裡展示，repo 裡的測試不觸發它。

## 較好的做法

讓每個供應商在 channel 裡都有位置可放，沒人收也不會卡住，只改一行：

```go,ignore
{{#include ../../../examples/chapter6/09_goroutine_leak/after/quote.go:fastest}}
```

測試在 `synctest.Test` 裡跑：

```go,ignore
{{#include ../../../examples/chapter6/09_goroutine_leak/after/quote_test.go:leak}}
```

兩個地方值得留意。慢的供應商睡一小時，睡的是 bubble 的假時間，所以瞬間跑完，還能斷言 `Fastest` 剛好花 1 秒。結尾的 `time.Sleep(time.Hour)` 不能省：時間只在主 goroutine 還活著時前進，主 goroutine 一結束，睡著的輸家也算卡住，連修好的版本都會 panic；睡過去，才換得輸家跑到送出那一行，有沒有洩漏才分得出來。

其他常見的洩漏，長相都一樣：goroutine 等一個永遠不會發生的事。在 bubble 裡，下面每一種都會得到和上面一樣的 `deadlock` panic：

- `range` 一條沒人關的 channel：`for v := range in` 在 `in` 關閉前不會結束。誰送誰關，見本章〈[fan-out / fan-in 的取捨](05_fan_out_fan_in.md)〉。
- 沒有呼叫 `cancel` 的 `context`：goroutine 等 `<-ctx.Done()`，卻沒人取消。`go vet` 的 `lostcancel` 抓得到直接丟掉 `cancel` 的寫法，抓不到回傳 `cancel` 之後呼叫端忘了用的情況。修正見本章〈[取消與 `context` 的傳遞](04_cancellation_context.md)〉。
- 沒有出口的 `time.Ticker` 迴圈：Go 1.23 起沒人參照的 `Ticker` 會被回收，洩漏的是 `for range t.C` 那個 goroutine；迴圈要和 `ctx.Done()` 或停止信號一起 `select`。

## 取捨

`synctest` 抓洩漏有它的邊界：

- 只抓得到 durably blocked 的 goroutine：bubble 裡建立的 channel、`sync.Cond`、`WaitGroup`、`time.Sleep`。卡在網路 I/O、系統呼叫或 `sync.Mutex` 的不算，因為 `testing/synctest` 文件說它們可能被 bubble 外的事件叫醒。連真的 socket 的程式，要先換成 `net.Pipe` 這類行程內的連線。
- 要在 bubble 裡建構物件：channel 在 bubble 外建立就不算 durably blocked。
- 只抓測試實際走過的路徑；沒有測到「輸家」，就沒有輸家可洩漏。
- `t.Context()` 會在 bubble 結束前被取消，所以由它衍生的 `ctx` 會把「沒人取消」的洩漏藏起來；要驗證取消，用 `context.WithCancel(context.Background())`，由測試自己呼叫 `cancel`。

比較 `runtime.NumGoroutine` 的前後數量也是常見做法，但這個數字包含 runtime 與 testing 自己開的 goroutine、其他平行測試開的 goroutine，而且 goroutine 結束不是同步的：想比得準就得加等待，又回到「賭時間」。第三方的 goleak 也能在測試結束時列出還活著的 goroutine，這裡只提名稱。正式環境怎麼看 goroutine 數量，第 9 章〈`/debug/pprof` 與診斷端點〉會談。

after 也不是沒有代價。緩衝大小等於供應商數量，輸家仍會跑完，繼續占著它用到的連線與下游配額；要讓輸家提早停下來，得把 `ctx` 傳進 `Source`，見本章〈[取消與 `context` 的傳遞](04_cancellation_context.md)〉。供應商若可能永遠不回，緩衝救不了它，一樣要靠取消。而 before 在一個只呼叫一次、回傳後行程就結束的小工具裡，洩漏沒有後果，不值得多加一行。

## 重點整理

- 洩漏是 goroutine 在等一個永遠不會發生的事：沒人接收的送出、沒人關閉的 `range`、沒人取消的 `ctx`、沒有出口的 `Ticker` 迴圈。
- 普通測試看不到洩漏；`synctest.Test` 在 bubble 結束時還有 goroutine 卡住就 panic，並指出卡在哪一行。
- 洩漏測試的寫法：在 bubble 裡建構、用假時間跑完，結尾讓時間走過所有計時，別讓睡著的 goroutine 被誤判。
- `synctest` 抓不到卡在網路 I/O、系統呼叫與 mutex 的 goroutine，也只抓測試走過的路徑。
- 比較 `runtime.NumGoroutine` 的數量容易不穩定；正式環境的觀察留給第 9 章。
