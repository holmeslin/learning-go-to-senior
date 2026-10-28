# fan-out / fan-in 的取捨

## 本集目標

看到「開一批 goroutine 再把結果收回來」時，判斷結果該怎麼收：大多數批次依索引寫進預先配置的 slice；需要邊收邊處理時，才用在全部結束後才關閉的結果 channel。

## 情境

服務要抓一批網址的頁面，同時抓，最後把頁面收在一起交給呼叫端。呼叫端要的是和輸入同序的結果，例如依序列表顯示。把工作分給多個 goroutine 叫 fan-out，把結果收回一處叫 fan-in。goroutine 與 channel 前作教過：〈[`sync.WaitGroup` 與 `wg.Go`](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter9/02_waitgroup.html)〉、〈[channel](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter9/08_channel.html)〉、〈[`close` 與 `range` channel](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter9/10_close_range.html)〉。這集只談收結果的方式。

## 常見寫法

每個網址一個 goroutine，結果送進 channel，收 `len(urls)` 次：

```go,ignore
{{#include ../../../examples/chapter6/05_fan_out_fan_in/before/fetch.go:fetchall}}
```

```go,ignore
{{#include ../../../examples/chapter6/05_fan_out_fan_in/before/fetch_test.go:test}}
```

數量已知，所以不用關閉 channel，測試也全過。完整程式碼在 `examples/chapter6/05_fan_out_fan_in/before/`。

## 問題在哪

**順序。** 結果依抵達的順序收集，先抓完的排前面。before 的測試只能排序後再比，所以看不出來。after 有一個順序測試（用 `synctest` 讓 `/a` 最慢、`/c` 最快，完成順序剛好相反），把它拿去測 before（在 repo 外的暫存模組）：

執行結果：

```text
--- FAIL: TestFetchAllKeepsInputOrder
    fetch_test.go:40: FetchAll() = [/c /b /a], want [/a /b /c]
FAIL
FAIL	repro/order
FAIL
```

省略了測試與套件後面的耗時。

**channel 由誰關。** 結果數量事先不知道時，不能收固定次數，只能 `for p := range results`，這時 channel 一定要有人關。下面兩種常見的錯，都會讓 `go test` 失敗，所以只在 repo 外重現。第一種是每個 goroutine 送完就關：

```text
go func() {
	results <- fetch(url)
	close(results) // 我做完了
}()
```

第一個完成的 goroutine 一關，`range` 就結束，`FetchAll` 只帶著 1 頁回傳；其他 goroutine 抓完再送出時：

執行結果：

```text
panic: send on closed channel
...
FAIL	repro/closeworker
```

省略了 panic 之後的 stack 與耗時。第二種是沒人關，`range` 永遠等不到結束。一般測試會卡到 `-timeout`（預設 10 分鐘）；在 `synctest` 的 bubble 裡會立刻變成 deadlock：

執行結果：

```text
--- FAIL: TestFetchAllReturnsEveryPage
panic: deadlock: all goroutines in bubble are blocked [recovered, repanicked]
...
FAIL	repro/noclose
```

同樣省略了 stack 與耗時。

**共享 slice。** 不用 channel，直接 `pages = append(pages, fetch(url))`，多個 goroutine 同時改同一個 slice，是 data race，`append` 還可能互相覆蓋而丟結果。`go test -race` 的報告如下，這是同一個報告裡的前兩段，位址與 goroutine 編號以 `...` 代替，其餘省略；哪個 goroutine 讀、哪個寫每次不一定相同：

執行結果（某一次）：

```text
WARNING: DATA RACE
Write at ... by goroutine ...:
  repro/race.FetchAll.func1()
      .../fetch.go:16 +0x114

Previous read at ... by goroutine ...:
  repro/race.FetchAll.func1()
      .../fetch.go:16 +0x68
...
```

加上 `sync.Mutex` 可以消除 race，但結果又回到抵達順序，和第一個問題一樣。

## 較好的做法

**依索引寫進預先配置的 slice。** 長度事先配好，每個 goroutine 只寫自己那一格：

```go,ignore
{{#include ../../../examples/chapter6/05_fan_out_fan_in/after/fetch.go:fetchall}}
```

不同的格子是不同的記憶體，沒有 race，不需要 channel 與 mutex；`wg.Wait` 就是同步點，回傳時所有的寫入都完成了。結果天生與輸入同序，順序測試通過，輸出也固定：

```go,ignore
{{#include ../../../examples/chapter6/05_fan_out_fan_in/after/fetch_test.go:order}}
```

```go,ignore
{{#include ../../../examples/chapter6/05_fan_out_fan_in/after/fetch_test.go:example}}
```

**全部結束後才關閉結果 channel。** 需要邊收邊處理時，改回傳 channel，並讓一個專門的 goroutine 等全部送完再關：

```go,ignore
{{#include ../../../examples/chapter6/05_fan_out_fan_in/after/fetch.go:stream}}
```

規則是：channel 由「知道所有 sender 都結束」的一方來關，不是由任何一個 sender。上面的測試讓 `/slow` 卡住，`/fast` 照樣先到：

```go,ignore
{{#include ../../../examples/chapter6/05_fan_out_fan_in/after/fetch_test.go:stream}}
```

## 取捨

大多數批次用 slice 就夠：數量已知，而且要整批結果才能往下走（排序、組成回應）。它簡單、有序、不用想誰關 channel。代價是要等最慢的那一個，才拿得到任何結果；一個很慢的網址拖住整批。

channel 版適合兩種情況：結果要邊收邊處理（第一頁到了就先處理，不必等最慢的）；或結果數量事先不知道，例如 producer 一邊走訪一邊產生工作。它的代價：

- 順序是完成的順序。要順序就得帶上索引，收完再排，那就是繞一圈回到 slice。
- 呼叫端必須讀到 channel 關閉。讀到一半就 `break`，還在送出的 goroutine 會永遠卡住，本章〈goroutine 洩漏怎麼發生、怎麼在測試裡抓到〉會談；用取消讓它們提早結束，是本章〈取消與 `context` 的傳遞〉的內容。

兩種寫法都是每個網址一個 goroutine，同時進行的數量等於批次大小，批次很大時要設上限，見本章〈`errgroup` 的並行上限〉。

不在乎順序、數量又已知時，before 其實是對的：例如只要彙總（總數、最大值），收 `len(urls)` 次就好，不用關 channel，也不用預先配置 slice。

只有一個階段，fan-out 之後等全部完成，用 slice 就夠。每一階段各自開 goroutine、用 channel 串成多階段，才是 pipeline，寫法見前作〈[pipeline](https://holmeslin.github.io/learning-go-from-zero/zh-TW/chapter9/18_pipeline.html)〉。為了並行而硬做成 pipeline，只是多出關閉與洩漏的風險。

## 重點整理

- 收結果的 channel 不能由任何一個 sender 關閉；沒人關，`range` 永遠等不到結束。
- 多個 goroutine 不能 `append` 同一個 slice；改成預先配好長度，每個 goroutine 寫自己的索引。
- 大多數批次用 slice：同序、沒有 channel、不用想關閉。
- 要邊收邊處理或數量不知道時，才用 channel，並讓一個 goroutine 等全部送完再關。
- channel 版的順序是完成的順序，而且呼叫端必須讀到關閉。
